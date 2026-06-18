package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync/atomic"
)

// Backend represents a backend server
type Backend struct {
	URL          *url.URL
	ReverseProxy *httputil.ReverseProxy
}

// LoadBalancer distributes requests across backends
type LoadBalancer struct {
	backends []*Backend
	current  uint64
}

// NextBackend returns the next backend using round-robin
func (lb *LoadBalancer) NextBackend() *Backend {
	n := atomic.AddUint64(&lb.current, 1)
	return lb.backends[n%uint64(len(lb.backends))]
}

func (lb *LoadBalancer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	backend := lb.NextBackend()
	log.Printf("Routing request to: %s", backend.URL)
	backend.ReverseProxy.ServeHTTP(w, r)
}

func NewLoadBalancer(backendURLs []string) (*LoadBalancer, error) {
	lb := &LoadBalancer{}
	for _, rawURL := range backendURLs {
		u, err := url.Parse(rawURL)
		if err != nil {
			return nil, fmt.Errorf("invalid backend URL %s: %w", rawURL, err)
		}
		lb.backends = append(lb.backends, &Backend{
			URL:          u,
			ReverseProxy: httputil.NewSingleHostReverseProxy(u),
		})
	}
	return lb, nil
}

func main() {
	backends := []string{
		"http://localhost:8081",
		"http://localhost:8082",
	}

	lb, err := NewLoadBalancer(backends)
	if err != nil {
		log.Fatalf("Failed to create load balancer: %v", err)
	}

	log.Println("Load balancer listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", lb))
}
