package main

import (
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"testing"
)

func TestNextIndex(t *testing.T) {
	pool := ServerPool{backends: []*Backend{{URL: &url.URL{Host: "a"}}, {URL: &url.URL{Host: "b"}}}}
	idx1 := pool.NextIndex()
	idx2 := pool.NextIndex()
	if idx1 == idx2 {
		t.Errorf("expected round robin to increment index")
	}
}

func TestGetNextPeerSkipsDeadBackend(t *testing.T) {
	pool := ServerPool{backends: []*Backend{{URL: &url.URL{Host: "a"}, Alive: false}, {URL: &url.URL{Host: "b"}, Alive: true}}}
	peer := pool.GetNextPeer()
	if peer == nil || peer.URL.Host != "b" {
		t.Fatalf("expected healthy backend b, got %#v", peer)
	}
}

func TestConfiguredBackendsRejectsInvalidValues(t *testing.T) {
	for _, raw := range []string{"", "not-a-url", "localhost:8080"} {
		if _, err := configuredBackends(raw); err == nil {
			t.Fatalf("expected %q to be rejected", raw)
		}
	}
}

func TestConfiguredBackendsCreatesProxy(t *testing.T) {
	backends, err := configuredBackends("http://a:8081,https://b.example")
	if err != nil {
		t.Fatal(err)
	}
	if len(backends) != 2 || backends[0].ReverseProxy == nil || backends[1].ReverseProxy == nil {
		t.Fatalf("expected two initialized backends")
	}
}

func TestMetricsEndpoint(t *testing.T) {
	original := serverPool
	defer func() { serverPool = original }()
	serverPool = ServerPool{backends: []*Backend{{Alive: true}, {Alive: false}}}

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rr := httptest.NewRecorder()
	handleMetrics(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rr.Code)
	}
	if got := rr.Body.String(); got != "{\"configured_backends\":2,\"healthy_backends\":1}\n" {
		t.Fatalf("unexpected metrics body: %s", got)
	}
}

func TestHealthzReflectsBackendAvailability(t *testing.T) {
	original := serverPool
	defer func() { serverPool = original }()

	serverPool = ServerPool{backends: []*Backend{{Alive: false}}}
	rr := httptest.NewRecorder()
	handleHealthz(rr, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 without healthy backend, got %d", rr.Code)
	}

	serverPool = ServerPool{backends: []*Backend{{Alive: true}}}
	rr = httptest.NewRecorder()
	handleHealthz(rr, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 with healthy backend, got %d", rr.Code)
	}
}

func TestProxyAddsTraceID(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Trace-Id") == "" {
			t.Fatalf("expected trace id")
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer backend.Close()

	parsed, _ := url.Parse(backend.URL)
	proxy := httputil.NewSingleHostReverseProxy(parsed)
	original := serverPool
	defer func() { serverPool = original }()
	serverPool = ServerPool{backends: []*Backend{{URL: parsed, Alive: true, ReverseProxy: proxy}}}

	rr := httptest.NewRecorder()
	lbHandler(rr, httptest.NewRequest(http.MethodGet, "/hello", nil))
	if rr.Code != http.StatusNoContent {
		t.Fatalf("expected proxied 204, got %d", rr.Code)
	}
}
