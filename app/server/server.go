package server

import (
	"crypto/tls"
	"fmt"

	"github.com/Frizz925/gilgamesh/app"
	"github.com/Frizz925/gilgamesh/server"
	"github.com/Frizz925/gilgamesh/worker"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

func Start() error {
	cfg, err := app.LoadConfig()
	if err != nil {
		return fmt.Errorf("config load: %+v", err)
	}

	logger, err := zap.NewDevelopment()
	if err != nil {
		return fmt.Errorf("logger init: %+v", err)
	}
	defer logger.Sync() //nolint:errcheck

	s := server.New(server.Config{
		Logger:   logger,
		PoolSize: cfg.Proxy.Worker.PoolCount,
		WorkerConfig: worker.Config{
			Logger:          logger,
			ReadBufferSize:  cfg.Proxy.Worker.ReadBuffer,
			WriteBufferSize: cfg.Proxy.Worker.WriteBuffer,
		},
	})

	var g errgroup.Group
	if err := serveHTTP(s, cfg.Proxy.Ports, &g); err != nil {
		return fmt.Errorf("http listener: %+v", err)
	}
	if err := serveHTTPS(s, cfg.Proxy.SSL, &g); err != nil {
		return fmt.Errorf("https listener: %+v", err)
	}
	return g.Wait()
}

func serveHTTP(s *server.Server, ports []int, g *errgroup.Group) error {
	for _, port := range ports {
		addr := portToAddr(port)
		g.Go(func() error {
			return s.ListenAndServe(addr)
		})
	}
	return nil
}

func serveHTTPS(s *server.Server, cfg app.ProxySSL, g *errgroup.Group) error {
	if len(cfg.Ports) < 0 {
		return nil
	}
	cer, err := tls.LoadX509KeyPair(cfg.Certificate, cfg.CertificateKey)
	if err != nil {
		return err
	}
	tlsCfg := &tls.Config{
		MinVersion:   tls.VersionTLS12,
		Certificates: []tls.Certificate{cer},
	}
	for _, port := range cfg.Ports {
		addr := portToAddr(port)
		g.Go(func() error {
			return s.ListenAndServeTLS(addr, tlsCfg)
		})
	}
	return nil
}

func portToAddr(port int) string {
	return fmt.Sprintf(":%d", port)
}
