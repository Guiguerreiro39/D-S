package account

import (
	"fmt"
	"sync"
)

// Ledger is synchronized account map
type Ledger struct {
	Accounts map[string]int
	clock    uint64
	lock     sync.RWMutex
}

// NewLedger is a constructor of ledgers
func NewLedger() *Ledger {
	var l Ledger
	l.Accounts = make(map[string]int, 1)
	return &l
}

// Transaction is method of ledger that applies a transaction to itself
func (l *Ledger) Transaction(t Transaction) {
	l.lock.Lock()
	defer l.lock.Unlock()

	_, found := l.Accounts[t.From]

	if l.checkBalance(t) {
		if !found {
			l.Accounts[t.From] += t.Amount
		} else {
			l.Accounts[t.From] -= t.Amount
			l.Accounts[t.To] += t.Amount
		}

		l.clock++
	}
}

// checkBalance confirms a transacrtion won't put someone out of balance
// without locks as private for use inside other locked func
func (l *Ledger) checkBalance(t Transaction) bool {

	val, found := l.Accounts[t.From]

	if !found {
		return true
	}

	if val < t.Amount {
		return false
	}

	return true
}

// CheckBalance confirms a transacrtion won't put someone out of balance
func (l *Ledger) CheckBalance(t Transaction) bool {
	l.lock.RLock()
	defer l.lock.RUnlock()

	return l.checkBalance(t)
}

// GetClock return the ledger clock
func (l *Ledger) GetClock() uint64 {
	l.lock.RLock()
	defer l.lock.RUnlock()

	return l.clock
}

func (l *Ledger) String() string {
	l.lock.RLock()
	defer l.lock.RUnlock()

	return fmt.Sprintln(l.Accounts)
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}
