package sdsync

import (
	"sync"
)

func Lock(mtx *sync.Mutex, f func()) {
	if mtx != nil {
		mtx.Lock()
		defer mtx.Unlock()
	}
	f()
}

func LockW(mtx *sync.RWMutex, f func()) {
	if mtx != nil {
		mtx.Lock()
		defer mtx.Unlock()
	}
	f()
}

func LockR(mtx *sync.RWMutex, f func()) {
	if mtx != nil {
		mtx.RLock()
		defer mtx.RUnlock()
	}
	f()
}
