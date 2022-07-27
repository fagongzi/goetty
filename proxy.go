package goetty

import (
	"errors"
	"io"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Proxy simple reverse proxy
type Proxy interface {
	// Start start the proxy
	Start() error
	// Stop stop the proxy
	Stop() error
	// AddUpStream add upstream
	AddUpStream(address string, connectTimeout time.Duration)
}

// NewProxy returns a simple tcp proxy
func NewProxy(address string, logger *zap.Logger) Proxy {
	return &proxy{
		address: address,
		logger:  adjustLogger(logger),
	}
}

type proxy struct {
	logger  *zap.Logger
	address string
	server  NetApplication
	mu      struct {
		sync.Mutex
		seq       uint64
		upstreams []*upstream
	}
}

func (p *proxy) Start() error {
	server, err := NewApplication(p.address, nil, WithAppHandleSessionFunc(p.handleSession))
	if err != nil {
		return err
	}
	p.server = server
	return p.server.Start()
}

func (p *proxy) Stop() error {
	return p.server.Stop()
}

func (p *proxy) AddUpStream(address string, connectTimeout time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.mu.upstreams = append(p.mu.upstreams, &upstream{
		address:        address,
		connectTimeout: connectTimeout,
	})
}

func (p *proxy) getUpStream() *upstream {
	p.mu.Lock()
	defer p.mu.Unlock()

	n := uint64(len(p.mu.upstreams))
	if n == 0 {
		return nil
	}
	up := p.mu.upstreams[p.mu.seq%n]
	p.mu.seq++
	return up
}

func (p *proxy) handleSession(conn IOSession) error {
	upstream := p.getUpStream()
	if upstream == nil {
		return errors.New("no upstream")
	}
	upstreamConn := NewIOSession()
	err := upstreamConn.Connect(upstream.address, upstream.connectTimeout)
	if err != nil {
		return err
	}

	defer func() {
		if err := upstreamConn.Close(); err != nil {
			p.logger.Error("close upstream failed",
				zap.String("upstream", upstream.address),
				zap.Error(err))
		}
	}()

	srcConn := conn.RawConn()
	dstConn := upstreamConn.RawConn()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, err := io.Copy(srcConn, dstConn)
		if err != nil {
			p.logger.Error("copy data from upstream to client failed",
				zap.String("upstream", upstream.address),
				zap.Error(err))
		}
	}()
	_, err = io.Copy(dstConn, srcConn)
	if err != nil {
		p.logger.Error("copy data from client to upstream failed",
			zap.String("upstream", upstream.address),
			zap.Error(err))
	}
	wg.Wait()
	return err
}

type upstream struct {
	address        string
	connectTimeout time.Duration
}
