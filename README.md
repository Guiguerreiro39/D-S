# Distributed Systems and Security

## Exercise 2.3 
### Implement a toy peer-to-peer network.

This exercise asks you to program in Go a toy example of a peer-to-peer flooding network for send-
ing strings around. The peer-to-peer network should then be used to build a
distributed chat room. The chat room client should work as follows:

1. It runs as a command line program.
2. When it starts up it asks for the IP address and port number of an existing peer on the network. If the IP address or port is invalid or no peer is found at the address, the client starts its own new network with only itself as member.
3. Then the client prints its own IP address and the port on which it waits for connections.
4. Then it will iteratively prompt the user for text strings.
5. When the user types a text string at any connected client, then it will eventually be printed at all other clients.
6. Only the text string should be printed, no information about who sent it.

The system should be implemented as follows:

1. When a client connects to an existing peer, it will keep a TCP connection to that peer.
2. Then the client opens its own port where it waits for incoming TCP connections.
3. All the connections will be treated the same, they will be used for both sending and receiving strings.
4. It keeps a set of messages that it already sent. In Go you can make a set as a map var MessagesSent map[string]bool. You just map the strings that were sent to true. Initially all of them are set to false, so the set is initially empty, as it should be.
5. When a string is typed by the user or a string arrives on any of its connections, the client checks if it is already sent. If so, it does nothing. Otherwise it adds it to MessagesSent and then sends it on all its connections. (Remember concurrency control. Probably several go-routines will access the set at the same time. Make sure that does not give problems.)
6. Whenever a message is added to MessagesSent, also print it for the user to see.

Add this to your report:

1. Test you system and describe how you tested it.
2. Argue that you system has eventual consistency in the sense that if all clients stop typing, then eventually all clients will print the same set of strings.

## Exercise 4.5
### Implement a simple peer-to-peer ledger

Modify your code from Exercise 2.3 to add the following features:
1. The system now no longer broadcasts strings and prints them. Instead it implements a distributed ledger. Each client keeps a Ledger.
2. Each client can make Transactions. When they do all other peers eventually update their ledger with the transaction.
3. The system should ensure eventual consistency, i.e., if all clients stop sending transactions, then all ledgers will eventually be in the same correct state.

```go
package account
type Transaction struct {
    ID string
    From string
    To string
    Amount int
}

func (l *Ledger) Transaction(t *Transaction) {
    l.lock.Lock() ; defer l.lock.Unlock()
    l.Accounts[t.From] -= t.Amount
    l.Accounts[t.To] += t.Amount
}
```

4. Your system only has to work if there are two phases: first all the peers connect,
then they make transactions. But if you want to accommodate for later comers
a way to do it is to let each client keep a list of all the transactions it saw and
then forward them to clients that log in late.

Implement as follows:
1. Keep a sorted list of peers.
2. When connecting to a peer, ask for its list of peers.
3. Then add yourself to your own list.
4. Then connect to the ten peers after you on the list (with wrap around).
5. Then broadcast your own presence.
6. When a new presence is broadcast, add it to your list of peers.
7. When a transaction is made, broadcast the Transaction object.
8. When a transaction is received, update the local Ledger object.

Add this to your report:
1. Test you system and describe how you tested it.
2. Discuss whether connection to the next ten peers is a good strategy with respect to connectivity. In particular, if the network has 1000 peers, how many connections need to break to partition the network?
3. Argue that your system has eventual consistency if all processes are correct and the system is run in two-phase mode.
4. Assume we made the following change to the system: When a transaction arrives, it is rejected if the receiving account goes below 0. Does your system
still have eventual consistency? Why or why not?

## Exercise 5.11 
### RSA encryption 

Create a Go package with methods KeyGen, Encrypt and Decrypt, that implement RSA key generation, encryption and de-
cryption. Your solution should use integers from the math/big package.

The KeyGen method should take as input an integer k, such that the bit length
of the generated modulus n = pq is k. The primes p, q do not need to be primes
with certainty, they only need to be “probable primes”.

The public exponent e should be 3 (the smallest possible value, which gives
the fastest possible encryption). This means that the primes p, q that you output
must be such that
$$gcd(3, p − 1) = gcd(3, q − 1) = 1$$

Recall that e = 3 and d must satisfy that 3d mod (p − 1)(q − 1) = 1. Another
way to express this is to say that d must be the inverse of 3 modulo (p − 1)(q − 1),
this is written
d = 3 −1 mod (p − 1)(q − 1).

This way to express the condition will be useful when computing d.
Facts you may find useful:
* Other than standard methods for addition and multiplication, Mod and ModInverse will be useful.
* To generate cryptographically secure randomness, use crypto/rand. In particular, the function Prime 
from the crypto/rand package may be helpful to you here.

Test your solution by verifying (at least) that your modulus has the required
length and that encryption followed by decryption of a few random plaintexts
outputs the original plaintexts. Note that plaintexts and ciphtertexts in RSA are
basically numbers in a certain interval. So it is sufficient to test if encryption of
a number followed by decryption returns the original number. You do not need
to, for instance, convert character strings to numbers.

Implement methods EncryptToFile and DecryptFromFile that encrypt and
decrypt using AES in counter mode, using a key that is supplied as input. The
EncryptToFile method should take as input a file name and should write the
ciphertext to the file. Conversely the DecryptFromFile method should read the
ciphertext from the file specified, decrypt and output the plaintext.

Test your solution by encrypting a secret RSA key to a file. Then decrypt from
the file, and check that the result can be used for RSA decryption.

## Exercise 6.10 
### RSA signatures

Extend your Go package from Exercise 5.11 so that it can generate and verify RSA signatures, where the message is first
hashed with SHA-256 and then the hash value is signed using RSA, as described
in Sec. 6.4. The hashing can be done with the crypto/sha256 package.
Note: international standards for signatures always demand that the hash value
is padded in some way before being passed to RSA, but you can ignore this here.
Thus the hash value (which will be returned as a byte array) can be converted to
an integer directly. Such direct conversion should not be done in a real application.
In addition to the code, your solution should contain the following:

1. Verify that you can both generate and verify the signature on at least one message. Also modify the message and check that your verification rejects.
2. Measure the speed at which you can hash, in bits per second. For this you should time the hashing of messages much longer than a hash value, in order
to get realistic data – say 10KB;
3. Measure the time you code spends to produce an RSA signature on a hash value when using a 2000 bit RSA key;
4. Assume you had to process the entire message using RSA. Use the result from question 3 to compute the speed at which you could do this (in bits per second).

Hint: one of the RSA operations you timed in question 3 would allow you to
process about 2000 bits. Compare your result to the speed you measured in
question 2. Does it look like hashing makes signing more efficient?

## Exercise 6.13 
(Implement a Simple Peer-to-Peer Ledger)

Modify your code from Exercise 4.5 to add the following features:

1. The system still keeps a Ledger (see Fig. 4.6).
2. Each client can make SignedTransactions (see Fig. 6.1), i.e., what is broad-
cast is now objects of the type SignedTransaction.
3. The sender and receive of a transaction are now RSA public keys encoded
as strings. The client can only make a transaction if it knows the secret key
corresponding to the sending account. This ensure that only the owner of the
account can take money from the account. In a bit more detail, you have to
find a way to encode and decode RSA public keys into the string type. If
we call the encoding of pk by the name enc(pk), then the amount that ”be-
longs” to pk is Accounts[enc(pk)]. To transfer money from pk one makes a
SignedTransaction where pk is encoded and put in the From-field. An encod-
ing of the RSA public key to receive the amount is placed in the To-field. All
the fields (save Signature) are then signed under pk (using the corresponding
secret key) and the signature is placed in Signature. A SignedTransaction is
valid if the signature is valid. Only valid transactions are executed. The invalid
transactions are simply ignored.

Implement as in Exercise 4.5 with these additions:

```go
package account
type SignedTransaction struct {
	ID
	string // Any string
	From
	string // A verification key coded as a string
	To
	string // A verification key coded as a string
	Amount
	int
	// Amount to transfer
	Signature string // Potential signature coded as string
}

func (l *Ledger) SignedTransaction(t *SignedTransaction) {
	l.lock.Lock()
	defer l.lock.Unlock()
	/* We verify that the t.Signature is a valid RSA
	 * signature on the rest of the fields in t under
	 * the public key t.From.
	 */
	validSignature := true
	if validSignature {
		l.Accounts[t.From] -= t.Amount
		l.Accounts[t.To] += t.Amount
	}
}
```

1. When a transaction is made, broadcast the SignedTransaction object.
2. When a transaction is received, update the local Ledger object if the SignedTransaction
is has a valid signature and the amount is non-negative.

Add this to your report:

1. How you TA can easily run your system, how is it started, what kind of com-
mands does it take and so on.
2. Test your system and describe how you tested it.
3. If the test is automated, which is preferable, then describe how the TA can run
the test.

You do not have to:

1. Handle overdraft, i.e., we allow that accounts become negative.
2. Protection against cheating parties (neither Byzantine errors nor crash errors).

## Exercise 9.11 
(Software Wallet) 

Use your solution from Exercise 6.10 to create a software wallet for an RSA secret key. 
It should have these functions:

1. Generate(filename string, password string) string which generates a
public key and secret key and places the secret key on disk in the file with
filename in filename encrypted under the password in password. The function
returns the public key.
2. Sign(filename string, password string, msg []byte) Signature which
if the filename and password match that of Generate will sign msg and return
the signature. You pick what the type Signature is.

Your solution should:

1. Make measures that make it costly for an adversary which gets its hands on
the keyfile to bruteforce the password.
2. Describe clearly what measure have been taken.
3. Explain why the system was designed the way it was and, in particular, argue
why the system achieves the desired security properties.
4. Test the system and describe how it was tested.
5. Describe how your TA can run the system and how to run the test.

## Exercise 10.1 
(Total Order by Sequencer) 

Start from your solution in Exercise 6.13 and make it into a system with total order using the following idea:

1. Your system runs in two phases. In phase 1 the peers connect to the network. In phase 2 they can send signed transactions.
2. The peer that started the network is a designated sequencer.
3. The sequencer creates a special RSA key pair called the sequencer key pair.
4. When connecting to a network the new client is informed who is the sequencer.
5. It is the order in which the sequencer received the transactions that counts. This is communicated to the other peers as follows: Every 10 seconds the sequencer will take the transactions that it saw, but which have so far not been sequenced. Then it puts the IDs of those transactions into a block. A block has a block number and an ordered list of IDs, []string. It numbers the blocks 0, 1, . . . in the order they are sent. The sequencer signs the block and sends the block on the network.
6. A client will accept a block if and only if it has the next block number it has not seen yet and the block is signed by the sequencer.
7. All clients process the transactions they receive in the order chosen by the sequencer.
8. A transaction is ignored if it would make the sending account negative.

Your solution should describe:

* How your system was designed and why.
* How the TA can run your system.
* Your test must try to send transaction at the same time at different peers to see if the system handles concurrent transactions correctly. A suggestion for one test could be: Have an account with 1, 000 coins on it. Have a program P 1 which at replica 1 repeatedly executes a transaction moving 1 coin from account A
to B. It should send the transaction 1000 times. Have a program P 2 which at replica 2 repeatedly executes a transaction moving 1 coin from account A to C.
It should send the transaction 1000 times. Run the two programs at the same time. Make sure that all replicas see the same number of coins end up on all
accounts. check that account A is 0 when it is all done. If your system is too slow to do 2000 transactions, pick a lower number. But it is important that
the test runs long enough that both programs are running at the time where account A hits 0.
* How the system was tested and how to run your test if you did an automatic test.

Your do not need to:
* Handle errors, neither Byzantine nor crash errors. In particular, if the sequencer crashes, then the system is allowed to die.
* You do not have to make your test automatic, but it is recommended

## Exercise 11.2 
(Static proof-of-stake) 

Start from your solution in Exercise 6.13 and use parts of your code from Exercise 10.1. The code in Exercise 6.13 should already be a distributed ledger with authenticated transactions. However, it does not have total order. Change it such that it gives a total order of all transactions and rejects transactions that would bring an account into minus. Do it by adding a proof-of-stake, tree-based, totally-ordered broadcast. Implement total-order using a tree-based block-chain as the one in Section 11.7 based on proof-of-stake as described in Section 11.9.2. You should not implement finalization or dynamic parameters.

In a bit more detail, implement it as follows:

1. The initial seed Seed is picked by you and hardcoded into the genesis block.
2. Transactions are conducted in the unit AU.
3. The genesis block contains ten special public keys which by definition have $10^6$ AUs on them. All other accounts have 0 AU on them initially. These ten accounts are generated by you, and you know the secret keys.
4. Transactions are in positive, integral AUs.
5. A transaction must send at least 1 AU to be valid.
6. A block can contain any number of transactions, and might even contain as little as no transactions if there are none to add to the block.
7. SlotLength is 1 second. You might set it larger if your signatures are very slow to compute. Recall that you need to compute one signature per slot.
8. To take part in the lottery and making blocks you need an account in the ledger with a positive balance. Your number of tickets is the balance of the account. Throughout the system your number of tickets is the balance in the genesis block, so only the ten accounts you created can be part of running the system.
9. The signature keys used in the lottery is the same as those used in the ledger system.
10. Make the system run with 10 peers.
11. Set the hardness such that your system creates a new block about every 10 seconds.
12. A block is not added to the tree unless all transactions are correctly signed and valid (they make no account go below 0 at any point).
13. When a transaction is made, the receiver gets 1 AU less than what was sent. This is a transaction fee.
14. When a new block is made, then the account of the block creator gets 10 AU plus one AU for each transaction in the block.

Your report should include:

1. How can the TA run your code.
2. How did you test your code. Remember to test also against some of the peers being malicious.
3. How did you test that agreement was achieved.
4. When the system is not under attack, how many transactions per second can the system handled. A transaction is not counted as done until it has been ordered and the balance of the accounts have been updated with that transaction.