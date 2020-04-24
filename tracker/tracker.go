package tracker

import (
	"mika/store"
	"sync"
)

type Tracker struct {
	store *store.DataStore

	// Whitelist and whitelist lock
	WhitelistMutex *sync.RWMutex
	Whitelist      []string
}

func New() *Tracker {
	return &Tracker{}
}
