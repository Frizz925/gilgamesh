package server

import (
	"crypto/tls"
	"errors"
	"net"
	"sync/atomic"

	"github.com/Frizz925/gilgamesh/utils"
	"github.com/Frizz925/gilgamesh/worker"
	"go.uber.org/zap"
)

var ErrServerAlreadyStopped = errors.New("server already stopped")

type Config struct {
	WorkerConfig worker.Config
	Logger       *zap.Logger
	PoolSize     int
	TLSConfig    *tls.Config
}

type Server struct {
	noCopy utils.NoCopy //nolint:unused,structcheck

	logger    *zap.Logger
	pool      *worker.Pool
	tlsConfig atomic.Value
}

func New(cfg Config) *Server {
	if cfg.Logger == nil {
		panic("Logger is required")
	}
	s := &Server{
		logger: cfg.Logger,
		pool:   worker.NewPool(cfg.PoolSize, cfg.WorkerConfig),
	}
	if cfg.TLSConfig != nil {
		s.tlsConfig.Store(cfg.TLSConfig)
	}
	return s
}

func (s *Server) Serve(l net.Listener) error {
	return s.serve(l, false)
}

func (s *Server) ServeTLS(l net.Listener) error {
	return s.serve(l, true)
}

func (s *Server) Close() {
	s.pool.Close()
}

func (s *Server) serve(l net.Listener, isTLS bool) error {
	log := s.logger.With(zap.String("listener", l.Addr().String()))
	log.Info("Gilgamesh service started")
	defer log.Info("Gilgamesh service stopped")
	for {
		c, err := l.Accept()
		if err != nil {
			return err
		}
		if isTLS {
			tc := s.tlsConfig.Load().(*tls.Config)
			c = tls.Server(c, tc)
		}
		go s.serveConn(c)
	}
}

func (s *Server) serveConn(c net.Conn) {
	w := s.pool.Get()
	w.ServeConn(c)
	s.pool.Put(w)
}
