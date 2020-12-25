package nettest

import "net"

type listener struct {
	net.Conn
}

func NewListener() (net.Listener, net.Conn) {
	s, c := net.Pipe()
	return &listener{s}, c
}

func (l *listener) Accept() (net.Conn, error) {
	return l.Conn, nil
}

func (l *listener) Addr() net.Addr {
	return l.LocalAddr()
}
