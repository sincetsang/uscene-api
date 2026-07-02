package sdcall

import (
	"TMA/pkg/sdsync"
)

var (
	Lock  = sdsync.Lock
	LockR = sdsync.LockR
	LockW = sdsync.LockW
)
