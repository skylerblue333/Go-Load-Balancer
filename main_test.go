package main

import (
	"net/url"
	"testing"
)

func TestNextIndex(t *testing.T) {
	pool := &ServerPool{}
	u1, _ := url.Parse("http://localhost:8081")
	u2, _ := url.Parse("http://localhost:8082")
	
	pool.AddBackend(&Backend{URL: u1, Alive: true})
	pool.AddBackend(&Backend{URL: u2, Alive: true})

	idx1 := pool.NextIndex()
	idx2 := pool.NextIndex()
	
	if idx1 == idx2 {
		t.Errorf("Expected different indexes for round-robin, got %d and %d", idx1, idx2)
	}
}
