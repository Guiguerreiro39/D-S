package main

import (
	"encoding/gob"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"sync"
	"time"

	heap "github.com/emirpasic/gods/trees/binaryheap"
	"github.com/emirpasic/gods/utils"
	"gopkg.in/alecthomas/kingpin.v2"
	"k8s.io/apimachinery/pkg/util/wait"

	. "./account"
	"./aesrsa"
	. "./peers"
	"./services"
)

var localPeer Peer
var localKeys *aesrsa.RSAKeyPair

var peersList = NewList()
var ledger = NewLedger()
var past = NewPastMap()

var sequencer aesrsa.RSAKey
var sequencerSecret aesrsa.RSAKey

var inTransit = NewTransactionMap()

var wg sync.WaitGroup

func main() {

	var (
		sk = kingpin.Flag("public-key", "Use predefined keys: private key file").Short('c').String()
		pk = kingpin.Flag("secret-key", "Use predefined keys: public key file").Short('s').String()

		server     = kingpin.Command("server", "Create your own network")
		portServer = server.Flag("port", "Port of server.").Short('p').Default("4444").Int()

		peer = kingpin.Command("peer", "Connect to a peer in a pre-existing network.")
		ip   = peer.Arg("ip", "IP address of Peer.").Required().IP()
		port = peer.Arg("port", "Port of Peer.").Required().Int()
	)

	kingpin.CommandLine.HelpFlag.Short('h')

	cmd := kingpin.Parse()

	var listenCh = make(chan SignedTransaction)
	var blockCh = make(chan SignedBlock)

	if *sk != "" && *pk != "" {
		skey, _ := ioutil.ReadFile(*sk)
		pkey, _ := ioutil.ReadFile(*pk)

		localKeys = &aesrsa.RSAKeyPair{
			Public:  aesrsa.KeyFromString(string(pkey)),
			Private: aesrsa.KeyFromString(string(skey))}
	} else {
		createKeys()
	}

	gob.Register(&SignedBlock{})
	gob.Register(&SignedTransaction{})

	switch cmd {
	case "server":
		createNetwork(*portServer, listenCh, blockCh)

	case "peer":
		firstPeer := Peer{
			IP:   ip.String(),
			Port: *port}
		connectToNetwork(firstPeer, listenCh, blockCh)
	}

	startServices(listenCh, blockCh)
}

func startServices(listenCh chan SignedTransaction, blockCh chan SignedBlock) {
	var sequencerCh = make(chan Transaction)
	var quitCh = make(chan struct{})

	wg.Add(1)
	go beServer(listenCh, blockCh, quitCh)

	wait.PollInfinite(time.Second*10, wait.ConditionFunc(func() (bool, error) {
		return peersList.Length() > 1, nil
	}))

	wg.Add(3)
	go processTransactions(listenCh, sequencerCh, quitCh)
	go processBlocks(blockCh, quitCh)
	go services.Write(listenCh, attachNextID, quitCh, &wg)

	if checkIfSequencer() {
		wg.Add(1)
		go beSequencer(sequencerCh, blockCh, quitCh)
	}

	<-quitCh
	connect(&localPeer)
	wg.Wait()
}

func createKeys() {
	var err error

	localKeys, err = aesrsa.KeyGen(2048)
	if err != nil {
		fmt.Println(err.Error())
	}

	fmt.Println("Your secret key is:")
	fmt.Println(aesrsa.KeyToString(localKeys.Private))
	fmt.Println("Your public key is:")
	fmt.Println(aesrsa.KeyToString(localKeys.Public))
}

func connectToNetwork(peer Peer, listenCh chan<- SignedTransaction, blockCh chan<- SignedBlock) {
	conn1, err := connect(&peer)

	if err != nil {
		panic(err.Error())
	}

	localPeer = GetLocalPeer(peer.Port+1, aesrsa.KeyToString(localKeys.Public))
	fmt.Println("Connection to the network Succesfull")
	peersList.SortedInsert(&localPeer)
	handleFirstConn(conn1, listenCh, blockCh)
	fmt.Println("Your IP is:", localPeer.IP, "with open port:", localPeer.GetPort())
}

func createNetwork(port int, listenCh chan<- SignedTransaction, blockCh chan<- SignedBlock) {
	localPeer = GetLocalPeer(port, aesrsa.KeyToString(localKeys.Public))
	peersList.SortedInsert(&localPeer)
	becomeSequencer()
	fmt.Println("Initializing your own network")
	fmt.Println("Your IP is:", localPeer.IP, "with open port:", localPeer.GetPort())
}

func becomeSequencer() {
	keyPair, err := aesrsa.KeyGen(2048)

	if err != nil {
		fmt.Println(err.Error())
	}

	sequencer = keyPair.Public
	sequencerSecret = keyPair.Private
}

func checkIfSequencer() bool {
	return sequencerSecret != aesrsa.RSAKey{}
}

func connect(peer *Peer) (net.Conn, error) {
	if peer.IP == "<nil>" {
		return nil, errors.New("IP is not valid")
	}
	return net.Dial("tcp", peer.GetAddress())
}

func handleFirstConn(conn net.Conn, listenCh chan<- SignedTransaction, blockCh chan<- SignedBlock) {
	enc := gob.NewEncoder(conn)
	dec := gob.NewDecoder(conn)

	// asking for list of peers
	signalAsk(enc)

	getSequencer(dec)

	p := &Peer{}
	err := dec.Decode(p)
	for p.Port != -1 {
		if err == nil {
			peersList.SortedInsert(p)
		}
		p = &Peer{}
		err = dec.Decode(p)
	}
	conn.Close()

	// broadcasting ourselves
	i := 0
	for p := range peersList.IterWrap(&localPeer) {
		if *p != localPeer {
			if i >= 10 {
				break
			}
			conn, err := connect(p)
			if err == nil {
				p.AddConn(conn)
				enc = p.GetEnc()
				signalNoAsk(enc)
				wg.Add(1)
				go handleConn(p, listenCh, blockCh)
			}
			i++
		}
	}

}

// ask for list of peers
func signalAsk(enc *gob.Encoder) {
	enc.Encode(Peer{IP: "", Port: -1})
}

// signal not asking for list of peers
func signalNoAsk(enc *gob.Encoder) {
	enc.Encode(localPeer)
}

// getSequencer receive the sequencer's public key
func getSequencer(dec *gob.Decoder) {
	key := aesrsa.RSAKey{}
	err := dec.Decode(&key)
	if err != nil {
		panic(err)
	}

	sequencer = key
}

func beServer(listenCh chan<- SignedTransaction, blockCh chan<- SignedBlock, quitCh <-chan struct{}) {
	defer fmt.Println("server closed")
	defer wg.Done()

	ln, err := net.Listen("tcp", ":"+localPeer.GetPort())
	if err != nil {
		fmt.Println("Fatal server error")
		panic(-1)
	}
	defer ln.Close()

	for {
		conn, _ := ln.Accept()
		select {
		case _, open := <-quitCh:
			if !open {
				ln.Close()
				closeAllConn()
				return //Done
			}
		default:
			if p, firstConn := checkAsk(conn); !firstConn {
				wg.Add(1)
				go handleConn(p, listenCh, blockCh)
			}
		}
	}

}

func closeAllConn() {
	for conn := range peersList.IterConn() {
		conn.Close()
	}
}

// check if the peer asks for list of peers
func checkAsk(conn net.Conn) (*Peer, bool) {
	dec := gob.NewDecoder(conn)
	p := &Peer{}
	err := dec.Decode(p)
	if err == nil {
		if p.Port == -1 {
			enc := gob.NewEncoder(conn)

			enc.Encode(sequencer)

			for p := range peersList.Iter() {
				enc.Encode(*p)
			}

			enc.Encode(Peer{Port: -1})
			return &Peer{}, true
		}
		p.AddConn(conn)
		p.AddDec(dec)
		peersList.SortedInsert(p)
		return p, false
	}
	return &Peer{}, true
}

func handleConn(peer *Peer, listenCh chan<- SignedTransaction, blockCh chan<- SignedBlock) {
	defer wg.Done()
	defer peer.Close()

	fmt.Println("Connected to", peer)

	dec := peer.GetDec()

	for {
		var obj WhatType
		err := dec.Decode(&obj)
		if err != nil {
			fmt.Println(err)
			fmt.Println("Closed connection to", peer)
			peersList.Remove(peer)
			break //Done
		} else {
			switch obj.WhatType() {
			case "SignedTransaction":
				listenCh <- *obj.(*SignedTransaction)
			case "SignedBlock":
				blockCh <- *obj.(*SignedBlock)
			}
		}
	}
}

func processTransactions(listenCh <-chan SignedTransaction, sequencerCh chan<- Transaction, quitCh <-chan struct{}) {
	defer wg.Done()

	for {
		select {
		case st := <-listenCh:
			if t := st.ExtractTransaction(); !isOld(t) && isVerified(st) && ledger.CheckBalance(t) {
				inTransit.AddTransaction(t)
				past.AddPast(t, true)
				if checkIfSequencer() {
					sequencerCh <- t
				}
				broadcast(st)
			}
		case <-quitCh:
			return //Done
		}
	}
}

func isOld(t Transaction) bool {
	if val, found := past.GetPast(t); found && val {
		return true
	}
	return false
}

func isVerified(st SignedTransaction) bool {
	return st.VerifyTransaction() && st.Amount > 0
}

func attachNextID(t Transaction) Transaction {
	t.ID = fmt.Sprintf("%d-%s", past.GetPastLength(), localPeer.GetAddress())
	past.AddPast(t, false)
	return t
}

func broadcast(st SignedTransaction) {
	var w WhatType = st
	for enc := range peersList.IterEnc() {
		enc.Encode(&w)
	}
}

// processBlocks applys blocks of transactions to the ledger
func processBlocks(blockCh <-chan SignedBlock, quitCh <-chan struct{}) {
	defer wg.Done()

	comp := func(a, b interface{}) int {
		b1 := a.(Block)
		b2 := b.(Block)
		return utils.IntComparator(b1.Number, b2.Number)
	}

	pq := heap.NewWith(comp)
	defer pq.Clear()

	lastBlock := -1

	for {
		select {
		case sb := <-blockCh:
			if b := sb.ExtractBlock(); sb.VerifyBlock(sequencer) && isFuture(b, lastBlock) {
				pq.Push(b)
				broadcastBlock(sb)
				lastBlock = applyAllValidBlocks(pq, lastBlock)
				fmt.Println(ledger) //TODO better print
			}
		case <-quitCh:
			return //Done
		}
	}

}

// isFuture tells if it's already been processed
func isFuture(b Block, lastBlock int) bool {
	fmt.Println(b.Number, lastBlock)
	return b.Number >= lastBlock+1
}

// isNext tells if it's the next block to be processed
func isNext(b Block, lastBlock int) bool {
	return b.Number == lastBlock+1
}

// Applys every transaction from a block
func updateLedger(b Block) {
	for _, id := range b.TransList {
		ledger.Transaction(inTransit.GetTransaction(id))
	}
}

func applyAllValidBlocks(pq *heap.Heap, lastBlock int) int {
	if !pq.Empty() {
		tmp, full := pq.Peek()
		min := tmp.(Block)
		for full && isNext(min, lastBlock) {
			tmp, _ := pq.Pop()
			min = tmp.(Block)

			updateLedger(min)
			lastBlock++

			tmp, full = pq.Peek()
			if full {
				min = tmp.(Block)
			}
		}
	}

	return lastBlock
}

// broadcast a signed block
func broadcastBlock(sb SignedBlock) {
	var w WhatType = sb
	for enc := range peersList.IterEnc() {
		enc.Encode(&w)
	}
}

// beSequencer add the beheaviour of a sequencer to the peer
func beSequencer(sequencerCh <-chan Transaction, blockCh chan<- SignedBlock, quitCh <-chan struct{}) {
	defer wg.Done()

	fmt.Println("You are the Sequencer")

	var n int
	ticker := time.NewTicker(time.Second * 10)

	for {
		seq := make([]string, 0)
		endBlock := false
		for !endBlock {
			select {
			case <-ticker.C:
				if len(seq[:]) > 0 {
					sb := NewSignedBlock(n, seq, sequencerSecret)
					broadcastBlock(*sb)
					blockCh <- *sb
					n++
					endBlock = true
				}
			case t := <-sequencerCh:
				seq = append(seq, t.ID)
			case <-quitCh:
				return //Done
			}
		}
	}
}
