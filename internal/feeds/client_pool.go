package feeds

import (
	"net/http"
	"sync"
)

// clientPool caches HTTP clients keyed by proxy URL, building lazily.
type clientPool struct {
	mu      sync.Mutex
	clients map[string]*http.Client
}

func newClientPool() *clientPool {
	return &clientPool{clients: map[string]*http.Client{}}
}

func (p *clientPool) get(key string, build func() (*http.Client, error)) (*http.Client, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if c, ok := p.clients[key]; ok {
		return c, nil
	}
	c, err := build()
	if err != nil {
		return nil, err
	}
	p.clients[key] = c
	return c, nil
}

func (p *clientPool) closeIdle() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for _, c := range p.clients {
		c.CloseIdleConnections()
	}
}
