package server

import (
	"crypto/tls"
	"fmt"
	"net"
	"os"
	"strconv"

	"github.com/Frizz925/gilgamesh/app"
	"github.com/Frizz925/gilgamesh/auth"
	"github.com/Frizz925/gilgamesh/server"
	"github.com/Frizz925/gilgamesh/utils"
	"github.com/Frizz925/gilgamesh/worker"
	"golang.org/x/sync/errgroup"

	"go.uber.org/zap"
)

type Dependencies struct {
	Logger    *zap.Logger
	TLSConfig *tls.Config
}

func Start() error {
	cfg, err := app.LoadConfig()
	if err != nil {
		return fmt.Errorf("config load: %+v", err)
	}

	deps := &Dependencies{}
	if utils.IsProduction() {
		deps.Logger, err = zap.NewProduction()
	} else {
		deps.Logger, err = zap.NewDevelopment()
	}
	if err != nil {
		return fmt.Errorf("logger init: %+v", err)
	}
	if len(cfg.Proxy.Server.TLSPorts) > 0 {
		cer, err := tls.LoadX509KeyPair(
			cfg.Proxy.TLS.Certificate,
			cfg.Proxy.TLS.CertificateKey,
		)
		if err != nil {
			return fmt.Errorf("certificate load: %+v", err)
		}
		deps.TLSConfig = &tls.Config{
			Certificates: []tls.Certificate{cer},
		}
	}

	s, err := New(cfg, deps)
	if err != nil {
		return fmt.Errorf("server init: %+v", err)
	}

	g := &errgroup.Group{}
	if err := listenAndServe(g, cfg.Proxy.Server.Ports, s.Serve); err != nil {
		return err
	}
	if err := listenAndServe(g, cfg.Proxy.Server.TLSPorts, s.ServeTLS); err != nil {
		return err
	}
	return g.Wait()
}

func New(cfg *app.Config, deps *Dependencies) (*server.Server, error) {
	var credentials auth.Credentials
	if cfg.Proxy.PasswordsFile != "" {
		f, err := os.Open(cfg.Proxy.PasswordsFile)
		if err != nil {
			return nil, fmt.Errorf("passwords file read: %+v", err)
		}
		v, err := auth.ReadCredentials(f)
		if err != nil {
			return nil, fmt.Errorf("passwords file parsing: %+v", err)
		}
		credentials = v
	}

	return server.New(server.Config{
		Logger:    deps.Logger,
		TLSConfig: deps.TLSConfig,
		PoolSize:  cfg.Proxy.Worker.PoolCount,
		WorkerConfig: worker.Config{
			Logger:          deps.Logger,
			ReadBufferSize:  cfg.Proxy.Worker.ReadBuffer,
			WriteBufferSize: cfg.Proxy.Worker.WriteBuffer,
			Credentials:     credentials,
		},
	}), nil
}

func listenAndServe(g *errgroup.Group, ports []int, serve func(l net.Listener) error) error {
	for _, port := range ports {
		l, err := net.Listen("tcp", portToAddr(port))
		if err != nil {
			return fmt.Errorf("listener init: %+v", err)
		}
		g.Go(func() error {
			return serve(l)
		})
	}
	return nil
}

func portToAddr(port int) string {
	return net.JoinHostPort("", strconv.Itoa(port))
}
