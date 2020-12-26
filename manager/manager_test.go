package manager

import (
	"bufio"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"math/big"
	"net"
	"testing"

	"github.com/Frizz925/gilgamesh/server"
	"github.com/Frizz925/gilgamesh/testutils/nettest"
	"github.com/Frizz925/gilgamesh/worker"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

type ManagerTestSuite struct {
	suite.Suite

	logger *zap.Logger
	server *server.Server
}

func TestManager(t *testing.T) {
	suite.Run(t, &ManagerTestSuite{})
}

func (suite *ManagerTestSuite) SetupSuite() {
	require := suite.Require()
	logger, err := zap.NewDevelopment()
	require.NoError(err)
	suite.logger = logger
}

func (suite *ManagerTestSuite) SetupTest() {
	suite.server = server.New(server.Config{
		Logger: suite.logger,
		WorkerConfig: worker.Config{
			Logger: suite.logger,
		},
	})
}

func (suite *ManagerTestSuite) TearDownTest() {
	suite.server.Close()
}

func (suite *ManagerTestSuite) TearDownSuite() {
	_ = suite.logger.Sync()
}

func (suite *ManagerTestSuite) TestTLSReload() {
	assert := suite.Assert()
	l, c := suite.startManager()
	res, err := sendCommand(c, commandTLSReload)
	assert.NoError(err)
	assert.Equal("OK", res)
	assert.NoError(c.Close())
	assert.NoError(l.Close())
}

func (suite *ManagerTestSuite) TestInvalidCommand() {
	assert := suite.Assert()
	l, c := suite.startManager()
	res, err := sendCommand(c, "INVALID")
	assert.NoError(err)
	assert.Equal("ERROR Unknown command 'INVALID'", res)
	assert.NoError(c.Close())
	assert.NoError(l.Close())
}

func (suite *ManagerTestSuite) TestNoWrite() {
	assert := suite.Assert()
	l, c := suite.startManager()
	assert.NoError(c.Close())
	assert.NoError(l.Close())
}

func (suite *ManagerTestSuite) TestTLSReloadError() {
	assert := suite.Assert()
	expectedErr := errors.New("tls certificate load error")
	l, c := suite.startManager(func() (tls.Certificate, error) {
		return tls.Certificate{}, expectedErr
	})
	res, err := sendCommand(c, commandTLSReload)
	assert.NoError(err)
	assert.Equal("ERROR Failed updating TLS config: "+expectedErr.Error(), res)
	assert.NoError(c.Close())
	assert.NoError(l.Close())
}

func (suite *ManagerTestSuite) startManager(loader ...LoadCertificateFunc) (net.Listener, net.Conn) {
	lc := loadCertificate
	if len(loader) > 0 {
		lc = loader[0]
	}
	m := New(Config{
		Logger:          suite.logger,
		Server:          suite.server,
		LoadCertificate: lc,
	})
	l, c := nettest.NewListener()
	go func() {
		err := m.Serve(l)
		if err != nil {
			suite.logger.Error("Manager serve error", zap.Error(err))
		}
	}()
	return l, c
}

func sendCommand(c net.Conn, cmd string) (string, error) {
	bw := bufio.NewWriter(c)
	if _, err := bw.WriteString(cmd + "\r\n"); err != nil {
		return "", err
	}
	if err := bw.Flush(); err != nil {
		return "", err
	}
	sc := bufio.NewScanner(c)
	if !sc.Scan() {
		return "", sc.Err()
	}
	return sc.Text(), nil
}

func loadCertificate() (cer tls.Certificate, err error) {
	certPEM, keyPEM, err := generateCertificate()
	if err != nil {
		return cer, err
	}
	return tls.X509KeyPair(certPEM, keyPEM)
}

func generateCertificate() ([]byte, []byte, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	keyBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return nil, nil, err
	}

	snLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	sn, err := rand.Int(rand.Reader, snLimit)
	if err != nil {
		return nil, nil, err
	}

	template := &x509.Certificate{
		SerialNumber: sn,
	}
	cerBytes, err := x509.CreateCertificate(rand.Reader, template, template, pub, priv)
	if err != nil {
		return nil, nil, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cerBytes,
	})
	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: keyBytes,
	})
	return certPEM, keyPEM, nil
}
