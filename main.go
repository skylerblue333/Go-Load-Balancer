package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const healthTimeout = 2 * time.Second

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
	if len(s.backends) == 0 {
		return -1
	}
	return int(atomic.AddUint64(&s.current, 1)-1) % len(s.backends)
}

func (s *ServerPool) GetNextPeer() *Backend {
	if len(s.backends) == 0 {
		return nil
	}
	next := s.NextIndex()
	for offset := 0; offset < len(s.backends); offset++ {
		idx := (next + offset) % len(s.backends)
		if s.backends[idx].IsAlive() {
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

func (b *Backend) Check(client *http.Client) bool {
	request, err := http.NewRequest(http.MethodGet, b.URL.String(), nil)
	if err != nil {
		b.SetAlive(false)
		return false
	}
	response, err := client.Do(request)
	if err != nil {
		b.SetAlive(false)
		return false
	}
	defer response.Body.Close()
	alive := response.StatusCode < http.StatusInternalServerError
	b.SetAlive(alive)
	return alive
}

var serverPool ServerPool

func lbHandler(w http.ResponseWriter, r *http.Request) {
	peer := serverPool.GetNextPeer()
	if peer == nil {
		http.Error(w, "no healthy backend is available", http.StatusServiceUnavailable)
		return
	}
	if traceID := r.Header.Get("X-Trace-Id"); traceID == "" {
		r.Header.Set("X-Trace-Id", newTraceID())
	}
	peer.ReverseProxy.ServeHTTP(w, r)
}

func newTraceID() string { return time.Now().UTC().Format("20060102T150405.000000000Z") }

func healthCheck(stop <-chan struct{}, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	client := &http.Client{Timeout: healthTimeout}
	check := func() {
		for _, backend := range serverPool.backends {
			backend.Check(client)
		}
	}
	check()
	for {
		select {
		case <-ticker.C:
			check()
		case <-stop:
			return
		}
	}
}

func handleMetrics(w http.ResponseWriter, _ *http.Request) {
	healthy := 0
	for _, backend := range serverPool.backends {
		if backend.IsAlive() {
			healthy++
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]int{"configured_backends": len(serverPool.backends), "healthy_backends": healthy})
}

func configuredBackends(raw string) ([]*Backend, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, errors.New("BACKENDS must contain at least one URL")
	}
	var backends []*Backend
	for _, item := range strings.Split(raw, ",") {
		parsed, err := url.Parse(strings.TrimSpace(item))
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			return nil, errors.New("BACKENDS contains an invalid URL")
		}
		backends = append(backends, &Backend{URL: parsed, Alive: false, ReverseProxy: httputil.NewSingleHostReverseProxy(parsed)})
	}
	return backends, nil
}

func main() {
	raw := os.Getenv("BACKENDS")
	if raw == "" {
		raw = "http://localhost:8081,http://localhost:8082"
	}
	backends, err := configuredBackends(raw)
	if err != nil {
		log.Fatal(err)
	}
	serverPool.backends = backends
	stop := make(chan struct{})
	go healthCheck(stop, 5*time.Second)
	defer close(stop)
	mux := http.NewServeMux()
	mux.HandleFunc("/metrics", handleMetrics)
	mux.HandleFunc("/", lbHandler)
	server := &http.Server{Addr: ":8080", Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	log.Println("L7 load balancer listening on :8080")
	log.Fatal(server.ListenAndServe())
}
