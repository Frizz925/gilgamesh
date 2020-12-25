package worker

import (
	"sync"

	"github.com/Frizz925/gilgamesh/utils"
)

type Pool struct {
	noCopy utils.NoCopy //nolint:unused,structcheck

	ch           chan *Worker
	pool         *sync.Pool
	preallocated bool
}

func NewPool(size int, cfg Config) *Pool {
	p := &Pool{
		preallocated: size > 0,
	}
	if p.preallocated {
		p.ch = make(chan *Worker, size)
		for i := 0; i < size; i++ {
			p.ch <- New(cfg)
		}
	} else {
		p.pool = &sync.Pool{
			New: func() interface{} {
				return New(cfg)
			},
		}
	}
	return p
}

func (p *Pool) Get() *Worker {
	if !p.preallocated {
		return p.pool.Get().(*Worker)
	}
	return <-p.ch
}

func (p *Pool) Put(w *Worker) {
	if !p.preallocated {
		p.pool.Put(w)
		return
	}
	select {
	case p.ch <- w:
	default:
		panic("Pre-allocated pool is full!")
	}
}

func (p *Pool) Close() {
	if !p.preallocated {
		return
	}
	// Drain the pre-allocated workers
	for {
		select {
		case <-p.ch:
		default:
			return
		}
	}
}
