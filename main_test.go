package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestNextIndex(t *testing.T) {
	pool := ServerPool{
		backends: []*Backend{
			{URL: &url.URL{Host: "a"}},
			{URL: &url.URL{Host: "b"}},
		},
	}

	idx1 := pool.NextIndex()
	idx2 := pool.NextIndex()

	if idx1 == idx2 {
		t.Errorf("Expected round robin to increment index")
	}
}

func TestGetNextPeer(t *testing.T) {
	pool := ServerPool{
		backends: []*Backend{
			{URL: &url.URL{Host: "a"}, Alive: false},
			{URL: &url.URL{Host: "b"}, Alive: true},
		},
	}

	peer := pool.GetNextPeer()
	if peer == nil || peer.URL.Host != "b" {
		t.Errorf("Expected to skip dead backend and return 'b'")
	}
}

func TestMetricsEndpoint(t *testing.T) {
	req, _ := http.NewRequest("GET", "/metrics", nil)
	rr := httptest.NewRecorder()
	handleMetrics(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected 200 OK, got %d", rr.Code)
	}
}
