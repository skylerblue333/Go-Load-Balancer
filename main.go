package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

type Backend struct {
	URL          *url.URL
	Alive        bool
	mu           sync.RWMutex
	ReverseProxy *httputil.ReverseProxy
	failures     atomic.Uint64
	requests     atomic.Uint64
}

type ServerPool struct {
	backends []*Backend
	current  atomic.Uint64
}

type App struct {
	pool   *ServerPool
	config Config
}

func (s *ServerPool) NextIndex() int {
	if len(s.backends) == 0 {
		return -1
	}
	return int(s.current.Add(1)-1) % len(s.backends)
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

func (b *Backend) Check(client *http.Client, healthPath string) bool {
	target := *b.URL
	target.Path = healthPath
	target.RawQuery = ""
	request, err := http.NewRequest(http.MethodGet, target.String(), nil)
	if err != nil {
		b.SetAlive(false)
		return false
	}
	request.Header.Set("User-Agent", "sky-edge-balancer-health/1")
	response, err := client.Do(request)
	if err != nil {
		b.SetAlive(false)
		return false
	}
	defer response.Body.Close()
	alive := response.StatusCode >= 200 && response.StatusCode < 500
	b.SetAlive(alive)
	return alive
}

func newBackend(target *url.URL, transport http.RoundTripper) *Backend {
	backend := &Backend{URL: target, Alive: false}
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.Transport = transport
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		req.Header.Set("X-Forwarded-Host", req.Host)
	}
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		backend.failures.Add(1)
		backend.SetAlive(false)
		log.Printf("upstream_error backend=%s trace_id=%s error=%q", backend.URL, r.Header.Get("X-Trace-Id"), err)
		http.Error(w, "upstream unavailable", http.StatusBadGateway)
	}
	backend.ReverseProxy = proxy
	return backend
}

func newApp(cfg Config) *App {
	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           (&net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          256,
		MaxIdleConnsPerHost:   64,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ExpectContinueTimeout: time.Second,
		ResponseHeaderTimeout: cfg.UpstreamTimeout,
	}
	pool := &ServerPool{backends: make([]*Backend, 0, len(cfg.Backends))}
	for _, target := range cfg.Backends {
		pool.backends = append(pool.backends, newBackend(target, transport))
	}
	return &App{pool: pool, config: cfg}
}

func (a *App) lbHandler(w http.ResponseWriter, r *http.Request) {
	peer := a.pool.GetNextPeer()
	if peer == nil {
		w.Header().Set("Retry-After", "1")
		http.Error(w, "no healthy backend is available", http.StatusServiceUnavailable)
		return
	}
	if r.Header.Get("X-Trace-Id") == "" {
		r.Header.Set("X-Trace-Id", newTraceID())
	}
	peer.requests.Add(1)
	peer.ReverseProxy.ServeHTTP(w, r)
}

func newTraceID() string {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return hex.EncodeToString([]byte(time.Now().UTC().Format(time.RFC3339Nano)))
	}
	return hex.EncodeToString(bytes[:])
}

func (a *App) healthCheck(ctx context.Context) {
	ticker := time.NewTicker(a.config.HealthInterval)
	defer ticker.Stop()
	client := &http.Client{Timeout: a.config.HealthTimeout}
	check := func() {
		var wg sync.WaitGroup
		for _, backend := range a.pool.backends {
			wg.Add(1)
			go func(b *Backend) {
				defer wg.Done()
				b.Check(client, a.config.HealthPath)
			}(backend)
		}
		wg.Wait()
	}
	check()
	for {
		select {
		case <-ticker.C:
			check()
		case <-ctx.Done():
			return
		}
	}
}

func (a *App) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (a *App) handleReadyz(w http.ResponseWriter, _ *http.Request) {
	for _, backend := range a.pool.backends {
		if backend.IsAlive() {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
			return
		}
	}
	http.Error(w, "no healthy backend", http.StatusServiceUnavailable)
}

func (a *App) handleMetrics(w http.ResponseWriter, _ *http.Request) {
	healthy := 0
	requests := uint64(0)
	failures := uint64(0)
	for _, backend := range a.pool.backends {
		if backend.IsAlive() {
			healthy++
		}
		requests += backend.requests.Load()
		failures += backend.failures.Load()
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"configured_backends": len(a.pool.backends),
		"healthy_backends":    healthy,
		"proxied_requests":    requests,
		"upstream_failures":   failures,
	})
}

func (a *App) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", a.handleHealthz)
	mux.HandleFunc("GET /readyz", a.handleReadyz)
	mux.HandleFunc("GET /metrics", a.handleMetrics)
	mux.HandleFunc("/", a.lbHandler)
	return securityHeaders(mux)
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

func run(ctx context.Context, cfg Config) error {
	app := newApp(cfg)
	healthCtx, cancelHealth := context.WithCancel(ctx)
	defer cancelHealth()
	go app.healthCheck(healthCtx)

	server := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           app.routes(),
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		IdleTimeout:       cfg.IdleTimeout,
		MaxHeaderBytes:    1 << 20,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("sky-edge-balancer listening=%s backends=%d", cfg.ListenAddr, len(cfg.Backends))
		errCh <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownGracePeriod)
		defer cancel()
		return server.Shutdown(shutdownCtx)
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

func main() {
	cfg, err := loadConfig()
	if err != nil {
		log.Fatal(err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, cfg); err != nil {
		log.Fatal(err)
	}
}
