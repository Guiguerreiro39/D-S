package blocktree

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
)

type nodeHash [32]byte

// GetNode gets a node given its hash
func (nh nodeHash) getNode(t *Tree) *Node { //maybe needs locks
	val, _ := t.nodeSet[nh]
	return val
}

// GetNode gets a node given its hash (must be in the tree)
func (nh nodeHash) getParent(t *Tree) nodeHash {
	return nh.getNode(t).Parent
}

// HashNode hashes a node
func HashNode(n *Node) nodeHash {
	json, err := json.Marshal(n)
	check(err)

	return sha256.Sum256(json)
}

func eqH(nh1, nh2 nodeHash) bool {
	return bytes.Equal(nh1[:], nh2[:])
}

// NodeSet is a type for set of nodes
type NodeSet = map[nodeHash]struct{}
