package server

import (
	"crypto/tls"
	"io"
	"net"

	"github.com/Frizz925/gilgamesh/utils"
	"github.com/Frizz925/gilgamesh/worker"
	"go.uber.org/zap"
)

type Config struct {
	WorkerConfig worker.Config

	Logger   *zap.Logger
	PoolSize int
}

type Server struct {
	noCopy utils.NoCopy //nolint:unused,structcheck

	log  *zap.Logger
	pool *worker.Pool
}

func New(cfg Config) *Server {
	if cfg.Logger == nil {
		panic("Logger is required")
	}
	return &Server{
		log:  cfg.Logger,
		pool: worker.NewPool(cfg.PoolSize, cfg.WorkerConfig),
	}
}

func (s *Server) ListenAndServe(addr string) error {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	return s.Serve(l)
}

func (s *Server) ListenAndServeTLS(addr string, config *tls.Config) error {
	l, err := tls.Listen("tcp", addr, config)
	if err != nil {
		return err
	}
	return s.Serve(l)
}

func (s *Server) Serve(l net.Listener) error {
	s.log.Info("Gilgamesh is serving", zap.String("addr", l.Addr().String()))
	for {
		conn, err := l.Accept()
		if err != nil {
			return err
		}
		w := s.pool.Get()
		go func() {
			if err := w.ServeConn(conn); err != nil && err != io.EOF {
				s.log.Error("Serve error", zap.Error(err))
			}
			s.pool.Put(w)
			if err := conn.Close(); err != nil {
				s.log.Error("Serve closing conn error", zap.Error(err))
			}
		}()
	}
}
