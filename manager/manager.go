package manager

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"net"
	"strings"

	"github.com/Frizz925/gilgamesh/server"
	"go.uber.org/zap"
)

const commandTLSReload = "TLS_RELOAD"

type LoadCertificateFunc func() (tls.Certificate, error)

type Manager struct {
	logger          *zap.Logger
	server          *server.Server
	loadCertificate LoadCertificateFunc
}

type Config struct {
	Logger          *zap.Logger
	Server          *server.Server
	LoadCertificate LoadCertificateFunc
}

func New(cfg Config) *Manager {
	return &Manager{
		logger:          cfg.Logger,
		server:          cfg.Server,
		loadCertificate: cfg.LoadCertificate,
	}
}

func (m *Manager) Serve(l net.Listener) error {
	log := m.logger.With(
		zap.String("domain", "manager"),
		zap.String("listener", l.Addr().String()),
	)
	log.Info("Gilgamesh manager started")
	defer log.Info("Gilgamesh manager stopped")
	for {
		c, err := l.Accept()
		if err != nil {
			return err
		}
		m.serveConn(log, c)
	}
}

func (m *Manager) serveConn(log *zap.Logger, c net.Conn) {
	var (
		cmdOk  bool
		errMsg string
	)
	defer func() {
		defer c.Close()
		var res string
		if cmdOk {
			res = "OK\r\n"
		} else if errMsg != "" {
			log.Error(errMsg)
			res = fmt.Sprintf("ERROR %s\r\n", errMsg)
		} else {
			return
		}
		bw := bufio.NewWriter(c)
		if _, err := bw.WriteString(res); err != nil {
			log.Error("Failed to write to buffer", zap.Error(err))
			return
		}
		if err := bw.Flush(); err != nil {
			log.Error("Failed to flush response", zap.Error(err))
			return
		}
	}()

	log = log.With(zap.String("src", c.RemoteAddr().String()))
	sc := bufio.NewScanner(c)
	if !sc.Scan() {
		log.Error("Read error", zap.Error(sc.Err()))
		return
	}
	parts := strings.Split(sc.Text(), " ")
	cmd, args := parts[0], parts[1:]
	log = log.With(
		zap.String("cmd", cmd),
		zap.Strings("args", args),
	)
	// For now we just handle TLS_RELOAD command
	if cmd != commandTLSReload {
		errMsg = fmt.Sprintf("Unknown command '%s'", cmd)
		return
	}
	if err := m.updateTLSConfig(); err != nil {
		errMsg = fmt.Sprintf("Failed updating TLS config: %+v", err)
		return
	}
	cmdOk = true
}

func (m *Manager) updateTLSConfig() error {
	cer, err := m.loadCertificate()
	if err != nil {
		return err
	}
	m.server.UpdateTLSConfig(&tls.Config{
		Certificates: []tls.Certificate{cer},
	})
	return nil
}
