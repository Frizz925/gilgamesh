package nettest

import (
	"errors"
	"net"
	"sync"
)

var ErrListenerClosed = errors.New("listener closed")

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
	c, ok := <-l.ch
	if ok {
		return c, nil
	}
	return nil, ErrListenerClosed
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
