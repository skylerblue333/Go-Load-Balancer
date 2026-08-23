package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestNextIndex(t *testing.T) {
	pool := ServerPool{backends: []*Backend{{URL: &url.URL{Host: "a"}}, {URL: &url.URL{Host: "b"}}}}
	idx1 := pool.NextIndex()
	idx2 := pool.NextIndex()
	if idx1 == idx2 {
		t.Fatalf("expected round robin to increment index")
	}
}

func TestGetNextPeerSkipsDeadBackend(t *testing.T) {
	pool := ServerPool{backends: []*Backend{{URL: &url.URL{Host: "a"}, Alive: false}, {URL: &url.URL{Host: "b"}, Alive: true}}}
	peer := pool.GetNextPeer()
	if peer == nil || peer.URL.Host != "b" {
		t.Fatalf("expected healthy backend b, got %#v", peer)
	}
}

func TestProxyRoundRobinAndHeaders(t *testing.T) {
	backendA := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Upstream", "a")
		_, _ = io.WriteString(w, r.Header.Get("X-Trace-Id"))
	}))
	defer backendA.Close()
	backendB := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Upstream", "b")
		_, _ = io.WriteString(w, r.Header.Get("X-Trace-Id"))
	}))
	defer backendB.Close()

	uA, _ := url.Parse(backendA.URL)
	uB, _ := url.Parse(backendB.URL)
	cfg := Config{Backends: []*url.URL{uA, uB}, UpstreamTimeout: time.Second}
	app := newApp(cfg)
	for _, b := range app.pool.backends {
		b.SetAlive(true)
	}

	seen := map[string]bool{}
	for i := 0; i < 4; i++ {
		rr := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "http://edge.test/demo", nil)
		app.routes().ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rr.Code)
		}
		seen[rr.Header().Get("X-Upstream")] = true
		if strings.TrimSpace(rr.Body.String()) == "" {
			t.Fatal("expected generated trace id to reach upstream")
		}
	}
	if !seen["a"] || !seen["b"] {
		t.Fatalf("expected both backends to receive traffic, seen=%v", seen)
	}
}

func TestUnavailableWhenAllBackendsDead(t *testing.T) {
	u, _ := url.Parse("http://127.0.0.1:1")
	app := newApp(Config{Backends: []*url.URL{u}, UpstreamTimeout: time.Second})
	rr := httptest.NewRecorder()
	app.routes().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "http://edge.test/", nil))
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rr.Code)
	}
	if rr.Header().Get("Retry-After") != "1" {
		t.Fatal("expected Retry-After header")
	}
}

func TestHealthReadinessAndMetrics(t *testing.T) {
	u, _ := url.Parse("http://example.test")
	app := newApp(Config{Backends: []*url.URL{u}, UpstreamTimeout: time.Second})

	rr := httptest.NewRecorder()
	app.routes().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("health expected 200, got %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	app.routes().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("readiness expected 503 with dead backend, got %d", rr.Code)
	}

	app.pool.backends[0].SetAlive(true)
	rr = httptest.NewRecorder()
	app.routes().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("readiness expected 200, got %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	app.routes().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), "healthy_backends") {
		t.Fatalf("unexpected metrics response %d %s", rr.Code, rr.Body.String())
	}
}

func TestParseBackendURLs(t *testing.T) {
	urls, err := parseBackendURLs("https://a.example, http://b.example")
	if err != nil || len(urls) != 2 {
		t.Fatalf("expected two backends: %v %v", urls, err)
	}
	if _, err := parseBackendURLs("ftp://bad.example"); err == nil {
		t.Fatal("expected non-http scheme rejection")
	}
	if _, err := parseBackendURLs("https://a.example,https://a.example"); err == nil {
		t.Fatal("expected duplicate rejection")
	}
}

func TestGracefulShutdown(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	cfg := Config{
		ListenAddr:          "127.0.0.1:0",
		Backends:            []*url.URL{{Scheme: "http", Host: "127.0.0.1:1"}},
		HealthPath:          "/",
		HealthInterval:      time.Second,
		HealthTimeout:       10 * time.Millisecond,
		UpstreamTimeout:     time.Second,
		ReadHeaderTimeout:   time.Second,
		IdleTimeout:         time.Second,
		ShutdownGracePeriod: time.Second,
	}
	if err := run(ctx, cfg); err != nil {
		t.Fatalf("graceful shutdown failed: %v", err)
	}
}
