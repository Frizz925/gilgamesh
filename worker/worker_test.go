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
	"github.com/stretchr/testify/require"
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

	username string
	password string

	pipe struct {
		client net.Conn
		server net.Conn
	}
}

func TestWorker(t *testing.T) {
	t.Run("empty logger", func(t *testing.T) {
		require.Panics(t, func() {
			New(Config{})
		})
	})

	suite.Run(t, &WorkerTestSuite{
		username: "user",
		password: "password",
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
}

func (suite *WorkerTestSuite) SetupTest() {
	// Serve the connection
	cp, sp := net.Pipe()
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

func (suite *WorkerTestSuite) TestNoAuth() {
	suite.setupWorker(false)
	require := suite.Require()
	res, err := suite.client.Get(suite.url.String())
	require.NoError(err)
	require.Equal(http.StatusOK, res.StatusCode)
	require.NoError(res.Body.Close())
}

func (suite *WorkerTestSuite) TestTunneling() {
	suite.setupWorker(false)
	require := suite.Require()
	{
		res, err := suite.client.Do(&http.Request{
			Method: http.MethodConnect,
			URL:    suite.url,
		})
		require.NoError(err)
		require.Equal(http.StatusOK, res.StatusCode)
	}
	{
		res, err := suite.client.Get(suite.url.String())
		require.NoError(err)
		require.Equal(http.StatusOK, res.StatusCode)
	}
}

func (suite *WorkerTestSuite) TestMalformedRequest() {
	suite.setupWorker(false)
	require := suite.Require()
	c := suite.pipe.client
	br, bw := bufio.NewReader(c), bufio.NewWriter(c)

	_, err := bw.Write([]byte("INVALID\r\n\r\n"))
	require.NoError(err)
	require.NoError(bw.Flush())

	res, err := http.ReadResponse(br, nil)
	require.NoError(err)
	require.Equal(0, res.StatusCode)
	require.NoError(res.Body.Close())
}

func (suite *WorkerTestSuite) TestUnknownHost() {
	suite.setupWorker(false)
	require := suite.Require()
	c := suite.pipe.client
	br, bw := bufio.NewReader(c), bufio.NewWriter(c)

	_, err := bw.Write([]byte("GET / HTTP/1.1\r\n\r\n"))
	require.NoError(err)
	require.NoError(bw.Flush())

	res, err := http.ReadResponse(br, nil)
	require.NoError(err)
	require.Equal(http.StatusBadRequest, res.StatusCode)
	require.NoError(res.Body.Close())
}

func (suite *WorkerTestSuite) TestFailedTunnel() {
	suite.setupWorker(false)
	require := suite.Require()
	res, err := suite.client.Get("http://0.0.0.0/")
	require.NoError(err)
	require.Equal(http.StatusBadGateway, res.StatusCode)
	require.NoError(res.Body.Close())
}

func (suite *WorkerTestSuite) TestAuthRequired() {
	suite.setupWorker(true)
	require := suite.Require()
	res, err := suite.client.Get(suite.url.String())
	require.NoError(err)
	require.Equal(http.StatusProxyAuthRequired, res.StatusCode)
	require.NoError(res.Body.Close())
}

func (suite *WorkerTestSuite) TestAuthSuccess() {
	suite.setupWorker(true)
	require := suite.Require()
	res, err := suite.client.Do(&http.Request{
		URL:    suite.url,
		Header: createAuthHeader(suite.username, suite.password),
	})
	require.NoError(err)
	require.Equal(http.StatusOK, res.StatusCode)
	require.NoError(res.Body.Close())
}

func (suite *WorkerTestSuite) TestAuthMalformedHeader() {
	suite.setupWorker(true)
	require := suite.Require()
	header := make(http.Header)
	header.Set("Proxy-Authorization", "Basic username:password")
	res, err := suite.client.Do(&http.Request{
		URL:    suite.url,
		Header: header,
	})
	require.NoError(err)
	require.Equal(http.StatusBadRequest, res.StatusCode)
}

func (suite *WorkerTestSuite) TestAuthUserNotFound() {
	suite.setupWorker(true)
	require := suite.Require()
	res, err := suite.client.Do(&http.Request{
		URL:    suite.url,
		Header: createAuthHeader("notfound", suite.password),
	})
	require.NoError(err)
	require.Equal(http.StatusForbidden, res.StatusCode)
}

func (suite *WorkerTestSuite) TestAuthPasswordMismatch() {
	suite.setupWorker(true)
	require := suite.Require()
	res, err := suite.client.Do(&http.Request{
		URL:    suite.url,
		Header: createAuthHeader(suite.username, "invalid"),
	})
	require.NoError(err)
	require.Equal(http.StatusForbidden, res.StatusCode)
}

func (suite *WorkerTestSuite) setupWorker(withAuth bool) {
	creds := make(auth.Credentials)
	if withAuth {
		require := suite.Require()
		pw, err := auth.CreatePassword([]byte(suite.password))
		require.NoError(err)
		creds[suite.username] = pw
	}
	w := New(Config{
		Logger:      suite.logger,
		Credentials: creds,
	})
	go w.ServeConn(suite.pipe.server)
}

func createAuthHeader(username, password string) http.Header {
	auth := fmt.Sprintf("%s:%s", username, password)
	authEnc := base64.URLEncoding.EncodeToString([]byte(auth))
	header := make(http.Header)
	header.Set("Proxy-Authorization", fmt.Sprintf("Basic %s", authEnc))
	return header
}
