package worker

import (
	"bufio"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"sync"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

const (
	MinBufferSize  = 512
	DefaultTimeout = 30 * time.Second
)

type Worker struct {
	Config
	reader *bufio.Reader
	writer *bufio.Writer

	peerBuf   []byte
	tunnelBuf []byte

	mu sync.Mutex
}

type Config struct {
	ReadBufferSize  int
	WriteBufferSize int
	Dialer          *net.Dialer
	Logger          *zap.Logger
}

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
	return &Worker{
		Config:    cfg,
		peerBuf:   make([]byte, cfg.ReadBufferSize),
		tunnelBuf: make([]byte, cfg.ReadBufferSize),
	}
}

func (w *Worker) ServeConn(c net.Conn) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	rb := w.acquireReader(c)
	wb := w.acquireWriter(c)

	req, err := readRequest(rb)
	if err != nil {
		return err
	}
	defer req.Body.Close()

	if req.URL.Host == "" {
		return writeResponse(badResponse(req), wb)
	}
	w.Logger.Info(
		"Opening proxy connection",
		zap.String("src", c.RemoteAddr().String()),
		zap.String("dest", req.URL.Host),
	)

	t, err := w.establishTunnel(req.URL.Host)
	if err != nil {
		return err
	}
	defer t.Close()

	if req.Method == http.MethodConnect {
		if err := handleTunneling(req, wb); err != nil {
			return err
		}
	} else {
		b, err := httputil.DumpRequestOut(req, true)
		if err != nil {
			return err
		}
		if err := writeFull(t, b); err != nil {
			return err
		}
	}

	g := &errgroup.Group{}
	g.Go(func() error {
		for {
			n, err := c.Read(w.peerBuf)
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
			if err := writeFull(c, w.tunnelBuf[:n]); err != nil {
				return err
			}
		}
	})
	return g.Wait()
}

func (w *Worker) acquireReader(c net.Conn) *bufio.Reader {
	if w.reader == nil {
		w.reader = bufio.NewReaderSize(c, w.ReadBufferSize)
	} else {
		w.reader.Reset(c)
	}
	return w.reader
}

func (w *Worker) acquireWriter(c net.Conn) *bufio.Writer {
	if w.writer == nil {
		w.writer = bufio.NewWriterSize(c, w.WriteBufferSize)
	} else {
		w.writer.Reset(c)
	}
	return w.writer
}

func (w *Worker) establishTunnel(hostport string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(hostport)
	if err != nil {
		host = hostport
		port = "80"
	}
	return w.Dialer.Dial("tcp", net.JoinHostPort(host, port))
}

func readRequest(rb *bufio.Reader) (*http.Request, error) {
	return http.ReadRequest(rb)
}

func handleTunneling(req *http.Request, wb *bufio.Writer) error {
	return writeResponse(connectResponse(req), wb)
}

func badResponse(req *http.Request) *http.Response {
	return &http.Response{
		Status:     "400 Bad Request",
		StatusCode: http.StatusBadRequest,
		Proto:      req.Proto,
		ProtoMajor: req.ProtoMajor,
		ProtoMinor: req.ProtoMinor,
	}
}

func connectResponse(req *http.Request) *http.Response {
	return &http.Response{
		Status:     "200 OK",
		StatusCode: http.StatusOK,
		Proto:      req.Proto,
		ProtoMajor: req.ProtoMajor,
		ProtoMinor: req.ProtoMinor,
	}
}

func writeResponse(res *http.Response, wb *bufio.Writer) error {
	b, err := httputil.DumpResponse(res, false)
	if err != nil {
		return err
	}
	if _, err := wb.Write(b); err != nil {
		return err
	}
	return wb.Flush()
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
