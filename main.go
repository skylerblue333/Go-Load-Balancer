package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"sync/atomic"
	"time"
)

// Simulated OpenTelemetry span context
type TraceContext struct {
	TraceID string
}

type Backend struct {
	URL          *url.URL
	Alive        bool
	mu           sync.RWMutex
	ReverseProxy *httputil.ReverseProxy
}

type ServerPool struct {
	backends []*Backend
	current  uint64
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

func (b *Backend) SetAlive(alive bool) {
	b.mu.Lock()
	b.Alive = alive
	b.mu.Unlock()
}

func (b *Backend) IsAlive() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.Alive
}

var serverPool ServerPool

func lbHandler(w http.ResponseWriter, r *http.Request) {
	peer := serverPool.GetNextPeer()
	if peer != nil {
		// Inject tracing headers
		r.Header.Set("X-Trace-Id", fmt.Sprintf("trace-%d", time.Now().UnixNano()))
		peer.ReverseProxy.ServeHTTP(w, r)
		return
	}
	http.Error(w, "Service not available", http.StatusServiceUnavailable)
}

func healthCheck() {
	t := time.NewTicker(time.Second * 5)
	for {
		select {
		case <-t.C:
			for _, b := range serverPool.backends {
				// Simulated health check logic
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				_ = ctx // Used in real HTTP calls
				b.SetAlive(true)
				cancel()
			}
		}
	}
}

func handleMetrics(w http.ResponseWriter, r *http.Request) {
	metrics := map[string]interface{}{
		"active_backends": len(serverPool.backends),
		"status":          "operational",
	}
	json.NewEncoder(w).Encode(metrics)
}

func main() {
	// Setup dummy backends for the load balancer
	urls := []string{"http://localhost:8081", "http://localhost:8082"}
	for _, u := range urls {
		parsedUrl, _ := url.Parse(u)
		serverPool.backends = append(serverPool.backends, &Backend{
			URL:          parsedUrl,
			Alive:        true,
			ReverseProxy: httputil.NewSingleHostReverseProxy(parsedUrl),
		})
	}

	go healthCheck()

	mux := http.NewServeMux()
	mux.HandleFunc("/", lbHandler)
	mux.HandleFunc("/metrics", handleMetrics)

	log.Println("L7 Load Balancer running on :8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
