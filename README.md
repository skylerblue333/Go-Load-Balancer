# Go-Load-Balancer

![CI](https://github.com/skylerblue333/Go-Load-Balancer/workflows/CI/badge.svg)
![Go](https://img.shields.io/badge/Go-1.21+-00ADD8.svg?style=flat&logo=go)
![gRPC](https://img.shields.io/badge/gRPC-Ready-244c5a.svg)

A production-ready Layer 7 reverse proxy load balancer written in Go, featuring active health checking, atomic round-robin routing, and distributed tracing hooks.

## System Architecture


```mermaid
graph TD
    Client -->|gRPC/HTTP2| LB[Go Load Balancer]
    LB -->|Round Robin| Node1[Service Node 1]
    LB -->|Round Robin| Node2[Service Node 2]
    Node1 -.->|OpenTelemetry| Jaeger[Jaeger Tracing]
    Node2 -.->|OpenTelemetry| Jaeger
    Node1 <-->|Consul| Discovery[Service Registry]
```


## Elite Features
- **Lock-Free Routing**: `sync/atomic` operations for zero-contention round robin.
- **Active Health Checks**: Background goroutine polling upstream health.
- **Tracing Ready**: Request header injection for OpenTelemetry spans.

## Quick Start
```bash
go mod tidy
go test ./...
go run main.go
```
