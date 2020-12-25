package worker

import (
	"bufio"
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"testing"

	"github.com/Frizz925/gilgamesh/auth"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

type mockHandler struct{}

func (mockHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusOK)
}

type WorkerTestSuite struct {
	suite.Suite

	logger *zap.Logger

	listener  net.Listener
	server    *http.Server
	client    *http.Client
	transport *http.Transport
	url       *url.URL

	worker   *Worker
	username string
	password []byte

	pipe struct {
		client net.Conn
		server net.Conn
	}
}

func TestWorker(t *testing.T) {
	suite.Run(t, &WorkerTestSuite{
		username: "user",
		password: []byte("password"),
	})
}

func (suite *WorkerTestSuite) SetupSuite() {
	require := suite.Require()

	// Setup logger
	logger, err := zap.NewDevelopment()
	require.NoError(err)
	suite.logger = logger

	// Setup http listener
	l, err := net.Listen("tcp", ":0")
	require.NoError(err)
	suite.listener = l
	suite.url = &url.URL{
		Scheme: "http",
		Host:   l.Addr().String(),
		Path:   "/",
	}

	// Setup http server
	server := &http.Server{Handler: mockHandler{}}
	go func() {
		if err := server.Serve(l); err != nil {
			logger.Error("HTTP server error", zap.Error(err))
		}
	}()
	suite.server = server

	// Setup worker dependencies
	pw, err := auth.CreatePassword(suite.password)
	require.NoError(err)
	creds := auth.Credentials{
		suite.username: pw,
	}

	// Setup worker
	suite.worker = New(Config{
		Logger:      logger,
		Credentials: creds,
	})
}

func (suite *WorkerTestSuite) SetupTest() {
	// Serve the connection
	cp, sp := net.Pipe()
	go suite.worker.ServeConn(sp)
	suite.pipe.client = cp
	suite.pipe.server = sp

	// Setup http transport
	transport := &http.Transport{
		DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
			return cp, nil
		},
	}
	suite.transport = transport

	// Setup http client
	client := &http.Client{Transport: transport}
	suite.client = client
}

func (suite *WorkerTestSuite) TearDownTest() {
	assert := suite.Assert()
	assert.NoError(suite.pipe.client.Close())
	assert.NoError(suite.pipe.server.Close())
}

func (suite *WorkerTestSuite) TearDownSuite() {
	_ = suite.server.Close()
	_ = suite.listener.Close()
	_ = suite.logger.Sync()
}

func (suite *WorkerTestSuite) TestMalformedRequest() {
	require := suite.Require()
	c := suite.pipe.client
	br, bw := bufio.NewReader(c), bufio.NewWriter(c)

	_, err := bw.Write([]byte("INVALID\r\n\r\n"))
	require.NoError(err)
	require.NoError(bw.Flush())

	res, err := http.ReadResponse(br, nil)
	require.NoError(err)
	require.Equal(http.StatusBadRequest, res.StatusCode)
}

func (suite *WorkerTestSuite) TestAuthRequired() {
	require := suite.Require()
	res, err := suite.client.Get(suite.url.String())
	require.NoError(err)
	require.Equal(http.StatusProxyAuthRequired, res.StatusCode)
}

func (suite *WorkerTestSuite) TestInvalidURL() {
	require := suite.Require()

	auth := fmt.Sprintf("%s:%s", suite.username, suite.password)
	authEnc := base64.URLEncoding.EncodeToString([]byte(auth))
	header := make(http.Header)
	header.Set("Proxy-Authorization", fmt.Sprintf("Basic %s", authEnc))

	res, err := suite.client.Do(&http.Request{
		Method: http.MethodGet,
		URL:    suite.url,
		Header: header,
	})
	require.NoError(err)
	require.Equal(http.StatusBadRequest, res.StatusCode)
}
