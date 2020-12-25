package utils

import "sync/atomic"

type AtomicBool struct {
	v uint32
}

func (a *AtomicBool) Get() bool {
	return atomic.LoadUint32(&a.v) == 1
}

func (a *AtomicBool) Set(b bool) {
	var val uint32 = 0
	if b {
		val = 1
	}
	atomic.StoreUint32(&a.v, val)
}
