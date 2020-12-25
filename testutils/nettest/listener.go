package nettest

import (
	"net"
	"sync"
)

type listener struct {
	ch        chan net.Conn
	addr      net.Addr
	closeOnce sync.Once
}

func NewListener() (net.Listener, net.Conn) {
	s, c := net.Pipe()
	l := &listener{
		ch:   make(chan net.Conn, 1),
		addr: s.LocalAddr(),
	}
	l.ch <- s
	return l, c
}

func (l *listener) Accept() (net.Conn, error) {
	return <-l.ch, nil
}

func (l *listener) Addr() net.Addr {
	return l.addr
}

func (l *listener) Close() error {
	l.closeOnce.Do(func() {
		close(l.ch)
	})
	return nil
}
