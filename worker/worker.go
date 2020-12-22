package worker

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Frizz925/gilgamesh/auth"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

const (
	MinBufferSize  = 512
	DefaultTimeout = 30 * time.Second
)

const (
	authRealm        = "Gilgamesh Web Proxy"
	authHeaderName   = "Proxy-Authorization"
	authHeaderPrefix = "Basic "
)

type Worker struct {
	id     uint64
	b64enc *base64.Encoding
	reader *bufio.Reader
	writer *bufio.Writer

	logger          *zap.Logger
	dialer          *net.Dialer
	credentials     auth.Credentials
	readBufferSize  int
	writeBufferSize int

	peerBuf       []byte
	tunnelBuf     []byte
	authorization bool

	mu sync.Mutex
}

type Config struct {
	ReadBufferSize  int
	WriteBufferSize int
	Dialer          *net.Dialer
	Logger          *zap.Logger
	Credentials     auth.Credentials
}

var (
	nextID         = uint64(1)
	defaultRequest = &http.Request{
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
	}
)

func New(config ...Config) *Worker {
	var cfg Config
	if len(config) > 0 {
		cfg = config[0]
	}
	if cfg.ReadBufferSize < MinBufferSize {
		cfg.ReadBufferSize = MinBufferSize
	}
	if cfg.WriteBufferSize < MinBufferSize {
		cfg.WriteBufferSize = MinBufferSize
	}
	if cfg.Logger == nil {
		panic("Logger is required")
	}
	if cfg.Dialer == nil {
		cfg.Dialer = new(net.Dialer)
	}
	id := atomic.AddUint64(&nextID, 1)
	return &Worker{
		id:     id,
		b64enc: base64.URLEncoding,

		logger:          cfg.Logger.With(zap.Uint64("worker_id", id)),
		dialer:          cfg.Dialer,
		credentials:     cfg.Credentials,
		readBufferSize:  cfg.ReadBufferSize,
		writeBufferSize: cfg.WriteBufferSize,

		peerBuf:       make([]byte, cfg.ReadBufferSize),
		tunnelBuf:     make([]byte, cfg.ReadBufferSize),
		authorization: len(cfg.Credentials) > 0,
	}
}

func (w *Worker) ServeConn(c net.Conn) {
	w.mu.Lock()
	defer w.mu.Unlock()
	rb := w.acquireReader(c)
	wb := w.acquireWriter(c)
	log := w.logger.With(zap.String("src", c.RemoteAddr().String()))

	var responseCode int
	req, err := readRequest(rb)
	if err != nil {
		log.Error("Malformed HTTP request")
		writeResponse(log, respond(nil, responseCode), wb)
		return
	}
	defer func() {
		if responseCode == http.StatusProxyAuthRequired {
			hdr := make(http.Header)
			hdr.Set("Proxy-Authenticate", fmt.Sprintf("Basic realm=\"%s\"", authRealm))
			writeResponse(log, respond(req, responseCode, hdr), wb)
		} else if responseCode > 0 {
			writeResponse(log, respond(req, responseCode), wb)
		}
		if req.Body != nil {
			_ = req.Body.Close()
		}
		log.Info("Closing connection")
		_ = c.Close()
	}()

	if w.authorization {
		responseCode = http.StatusProxyAuthRequired
		auth := req.Header.Get(authHeaderName)
		if !strings.HasPrefix(auth, authHeaderPrefix) {
			return
		}

		responseCode = http.StatusBadRequest
		dec, err := w.b64enc.DecodeString(auth[len(authHeaderPrefix):])
		if err != nil {
			log.Error("Malformed authorization header", zap.Error(err))
			return
		}

		responseCode = http.StatusForbidden
		parts := strings.Split(string(dec), ":")
		username, password := parts[0], parts[1]
		log = log.With(zap.String("user", username))
		pw, ok := w.credentials[username]
		if !ok {
			log.Error("Username not found")
			return
		}
		if pw.Compare([]byte(password)) != nil {
			log.Error("Password mismatch")
			return
		}
	}

	responseCode = http.StatusBadRequest
	if req.URL.Host == "" {
		log.Error("Malformed request URI", zap.String("url", req.URL.String()))
		return
	}
	host, port, err := net.SplitHostPort(req.URL.Host)
	if err != nil {
		host = req.URL.Host
		port = "80"
	}
	hostport := net.JoinHostPort(host, port)
	log = log.With(zap.String("dst", hostport))
	log.Info("Opening proxy connection")

	responseCode = http.StatusBadGateway
	t, err := w.establishTunnel(hostport)
	if err != nil {
		writeResponse(log, respond(req, http.StatusBadGateway), wb)
		return
	}
	defer t.Close()

	responseCode = 0
	if req.Method == http.MethodConnect {
		if !handleTunneling(log, req, wb) {
			return
		}
	} else {
		for key := range req.Header {
			if strings.HasPrefix(key, "Proxy") {
				req.Header.Del(key)
			}
		}
		b, err := httputil.DumpRequestOut(req, true)
		if err != nil {
			log.Error("Failed to dump request", zap.Error(err))
			return
		}
		if err := writeFull(t, b); err != nil {
			log.Error("Failed to write request to tunnel", zap.Error(err))
			return
		}
	}

	g := &errgroup.Group{}
	g.Go(func() error {
		for {
			n, err := rb.Read(w.peerBuf)
			if err != nil {
				return err
			}
			if err := writeFull(t, w.peerBuf[:n]); err != nil {
				return err
			}
		}
	})
	g.Go(func() error {
		for {
			n, err := t.Read(w.tunnelBuf)
			if err != nil {
				return err
			}
			if _, err := wb.Write(w.tunnelBuf[:n]); err != nil {
				return err
			}
			if err := wb.Flush(); err != nil {
				return err
			}
		}
	})
	if err := g.Wait(); err != nil && err != io.EOF {
		log.Error("Tunnel error", zap.Error(err))
	}
}

func (w *Worker) acquireReader(c net.Conn) *bufio.Reader {
	if w.reader == nil {
		w.reader = bufio.NewReaderSize(c, w.readBufferSize)
	} else {
		w.reader.Reset(c)
	}
	return w.reader
}

func (w *Worker) acquireWriter(c net.Conn) *bufio.Writer {
	if w.writer == nil {
		w.writer = bufio.NewWriterSize(c, w.writeBufferSize)
	} else {
		w.writer.Reset(c)
	}
	return w.writer
}

func (w *Worker) establishTunnel(hostport string) (net.Conn, error) {
	return w.dialer.Dial("tcp", hostport)
}

func readRequest(rb *bufio.Reader) (*http.Request, error) {
	return http.ReadRequest(rb)
}

func handleTunneling(log *zap.Logger, req *http.Request, wb *bufio.Writer) bool {
	raw := fmt.Sprintf("HTTP/%d.%d 200 OK\r\n\r\n", req.ProtoMajor, req.ProtoMinor)
	if _, err := wb.WriteString(raw); err != nil {
		log.Error("Failed to write response to buffer", zap.Error(err))
		return false
	}
	if err := wb.Flush(); err != nil {
		log.Error("Failed to flush buffer", zap.Error(err))
		return false
	}
	return true
}

func respond(req *http.Request, code int, header ...http.Header) *http.Response {
	if req == nil {
		req = defaultRequest
	}
	var hdr http.Header
	if len(header) > 0 {
		hdr = header[0]
	}
	status := fmt.Sprintf("%d %s", code, http.StatusText(code))
	return &http.Response{
		Status:     status,
		StatusCode: code,
		Proto:      req.Proto,
		ProtoMajor: req.ProtoMajor,
		ProtoMinor: req.ProtoMinor,
		Header:     hdr,
	}
}

func writeResponse(log *zap.Logger, res *http.Response, wb *bufio.Writer) bool {
	b, err := httputil.DumpResponse(res, false)
	if err != nil {
		log.Error("Failed to dump response", zap.Error(err))
		return false
	}
	if _, err := wb.Write(b); err != nil {
		log.Error("Failed to write response to buffer", zap.Error(err))
		return false
	}
	if err := wb.Flush(); err != nil {
		log.Error("Failed to flush buffer", zap.Error(err))
		return false
	}
	return true
}

func writeFull(w io.Writer, p []byte) error {
	for n := 0; n < len(p); {
		nn, err := w.Write(p[n:])
		if err != nil {
			return err
		}
		n += nn
	}
	return nil
}
