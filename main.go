package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"sync/atomic"
)

type Backend struct {
	URL    *url.URL
	Alive  bool
	mux    sync.RWMutex
	ReverseProxy *httputil.ReverseProxy
}

func (b *Backend) SetAlive(alive bool) {
	b.mux.Lock()
	b.Alive = alive
	b.mux.Unlock()
}

func (b *Backend) IsAlive() (alive bool) {
	b.mux.RLock()
	alive = b.Alive
	b.mux.RUnlock()
	return
}

type ServerPool struct {
	backends []*Backend
	current  uint64
}

func (s *ServerPool) AddBackend(b *Backend) {
	s.backends = append(s.backends, b)
}

func (s *ServerPool) NextIndex() int {
	return int(atomic.AddUint64(&s.current, uint64(1)) % uint64(len(s.backends)))
}

func (s *ServerPool) GetNextPeer() *Backend {
	next := s.NextIndex()
	l := len(s.backends) + next
	for i := next; i < l; i++ {
		idx := i % len(s.backends)
		if s.backends[idx].IsAlive() {
			if i != next {
				atomic.StoreUint64(&s.current, uint64(idx))
			}
			return s.backends[idx]
		}
	}
	return nil
}

func lbHandler(serverPool *ServerPool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		peer := serverPool.GetNextPeer()
		if peer != nil {
			peer.ReverseProxy.ServeHTTP(w, r)
			return
		}
		http.Error(w, "Service not available", http.StatusServiceUnavailable)
	}
}

func main() {
	pool := &ServerPool{}
	
	urls := []string{"http://localhost:8081", "http://localhost:8082"}
	for _, u := range urls {
		serverUrl, _ := url.Parse(u)
		proxy := httputil.NewSingleHostReverseProxy(serverUrl)
		pool.AddBackend(&Backend{
			URL:          serverUrl,
			Alive:        true,
			ReverseProxy: proxy,
		})
	}

	server := http.Server{
		Addr:    ":8080",
		Handler: lbHandler(pool),
	}

	log.Println("Load Balancer started on port 8080")
	log.Fatal(server.ListenAndServe())
}
