package services

import (
	"fmt"
	"time"

	. "../account"
	"../aesrsa"
	bt "../blocktree"
	"k8s.io/apimachinery/pkg/util/wait"
)

// Tree is the blockchain tree
var Tree *bt.Tree

// ProcessNodes implements the tree protocol
func ProcessNodes(sequencerCh <-chan Transaction, blockCh <-chan bt.SignedNode, keys *aesrsa.RSAKeyPair, quitCh <-chan struct{}) {
	defer Wg.Done()

	oldSeq := make([]string, 0)
	seq := make([]string, 0)

	var winner *bt.Node
	nodeOfSlot := bt.NodeSet{}

	timer := make(chan struct{})
	go pollSlotNumber(timer, quitCh)

	for {
		select {
		case <-timer:
			nodeOfSlot = bt.NodeSet{}

			// use winner for currentSlot-1
			if winner != nil {
				Tree.ConsiderLeaf(winner)
				fmt.Println(Tree.GetLedger())
				winner = nil
			} else { // if no winner but there were transaction then save them
				if len(oldSeq[:]) > 0 {
					seq = append(oldSeq, seq...)
				}
			}

			// make own node for current slot (just ended)
			if len(seq[:]) > 0 {
				n := bt.NewNode(Tree.GetSeed(), Tree.GetCurrentSlot(), seq, keys, Tree.GetHead())
				if Tree.Partecipating(n) {
					sn := bt.NewSignedNode(*n, keys.Private)
					go broadcastNode(*sn)
					winner = n
					nodeOfSlot[bt.HashNode(n)] = struct{}{}
				}
			}
			oldSeq = seq
			seq = make([]string, 0)

		case t := <-sequencerCh:
			if Tree.ConsiderTransaction(t, seq) {
				seq = append(seq, t.ID)
			}
		case sn := <-blockCh:
			if n := &sn.Node; isNewSlot(n) && !alreadySeenInSlot(n, nodeOfSlot) && Tree.CheckIsNext(n) && sn.VerifyNode() {
				nodeOfSlot[bt.HashNode(n)] = struct{}{}
				if winner == nil || Tree.CompareValueOfNodes(n, winner) {
					winner = n
				}
				go broadcastNode(sn)
			}
		case <-quitCh:
			return //Done
		}
	}
}

func isNewSlot(n *bt.Node) bool {
	return Tree.BelongsToCurrentSlot(n)
}

func alreadySeenInSlot(n *bt.Node, nodeOfSlot bt.NodeSet) bool {
	_, found := nodeOfSlot[bt.HashNode(n)]
	return found
}

func broadcastNode(sn bt.SignedNode) {
	Wg.Add(1)
	defer Wg.Done()

	var w WhatType = sn
	for enc := range PeerList.IterEnc() {
		enc.Encode(&w)
	}
}

func pollSlotNumber(timer chan<- struct{}, quitCh <-chan struct{}) {
	Wg.Add(1)
	defer Wg.Done()

	oldSlot := Tree.GetCurrentSlot()
	for {
		select {
		case <-quitCh:
			return
		default:
			wait.PollInfinite(time.Millisecond*100, wait.ConditionFunc(func() (bool, error) {
				return Tree.GetCurrentSlot() > oldSlot, nil
			}))
			oldSlot = Tree.GetCurrentSlot()

			timer <- struct{}{}
		}
	}
}
