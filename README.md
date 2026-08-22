# Go Load Balancer

A small HTTP reverse proxy with round-robin selection, real backend health checks, and JSON metrics. It is a focused component for local or controlled deployments, not a certified high-performance or production load-balancing platform.

## Implemented behavior

The service reads a comma-separated `BACKENDS` environment variable, validates backend URLs, routes requests only to healthy backends, returns `503 Service Unavailable` when no healthy backend exists, performs bounded HTTP health checks, preserves or adds an `X-Trace-Id` header, and exposes configured and healthy backend counts at `/metrics`.

```bash
BACKENDS=http://localhost:8081,http://localhost:8082 go run .
curl http://localhost:8080/metrics
```

The server uses a request-header timeout and a bounded health-check client. The pool handles an empty backend set without panicking. No telemetry, retry policy, circuit breaker, TLS termination, graceful shutdown orchestration, or distributed control plane is claimed.

## Validation

```bash
go test ./...
go vet ./...
```

The current test suite passes, and `go vet ./...` passes in the audit environment. Additional integration tests should be added before deployment behind real traffic.

## Scope and limitations

This repository does not establish capacity, latency, availability, or security guarantees. Operators must provide real backend services, TLS and network policy, structured production logging, rate limiting, access control, and an operational deployment strategy. The previous “high-performance,” “scalable,” and “cloud-native” language was removed because the current code does not substantiate those claims.
