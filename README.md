# Sky Edge Balancer

A compact, deployable Layer-7 reverse proxy/load-balancing product built in Go for small services, internal platforms, edge gateways, and self-hosted applications.

This repository has been productized from a small engineering component into a standalone deployable package while keeping its claims evidence-based.

## Product capabilities

- round-robin backend selection;
- active backend health checks;
- automatic unhealthy-backend avoidance;
- `503 Service Unavailable` when no healthy target exists;
- generated `X-Trace-Id` request correlation when callers do not provide one;
- JSON service metrics at `/metrics`;
- readiness-style health endpoint at `/healthz`;
- configurable listen address with `LISTEN_ADDR` or `-listen`;
- configurable comma-separated `BACKENDS` list;
- bounded upstream health-check timeout;
- safer `502 Bad Gateway` proxy error handling;
- graceful SIGINT/SIGTERM lifecycle;
- built-in `-healthcheck` probe for container orchestration;
- non-root distroless production container;
- standalone Docker Compose packaging;
- CI covering tests, vet, race detector, build, and Docker image construction.

## Quick start

```bash
BACKENDS=http://localhost:8081,http://localhost:8082 go run .
```

Then inspect health and metrics:

```bash
curl http://localhost:8080/healthz
curl http://localhost:8080/metrics
```

## Docker

```bash
docker build -t sky-edge-balancer .
docker run --rm -p 8080:8080 \
  -e BACKENDS=http://host.docker.internal:8081,http://host.docker.internal:8082 \
  sky-edge-balancer
```

Or use Compose:

```bash
BACKENDS=http://app1:8081,http://app2:8082 docker compose up --build
```

## Configuration

| Variable / flag | Default | Purpose |
| --- | --- | --- |
| `BACKENDS` | `http://localhost:8081,http://localhost:8082` | comma-separated backend URLs |
| `LISTEN_ADDR` | `:8080` | HTTP listen address |
| `-listen` | value of `LISTEN_ADDR` | command-line listen override |
| `-healthcheck` | off | probe local `/healthz` and exit |

`config.example.json` documents the intended commercial configuration surface, although runtime configuration is intentionally environment/flag based today.

## Verification

```bash
go test ./...
go vet ./...
go test -race ./...
go build ./...
docker build -t sky-edge-balancer:test .
```

GitHub Actions runs these checks for pushes and pull requests.

## Commercial packaging position

Sky Edge Balancer is suitable as a focused self-hosted gateway component or as the runtime foundation for a future managed control-plane product. A managed SaaS edition would add centralized configuration, tenant authentication, certificates, usage metering, dashboards, remote deployment, and billing without changing the data-plane contract.

## Current limitations

This code does **not** claim parity with HAProxy, Envoy, NGINX, or managed cloud load balancers. It currently does not provide TLS termination, weighted balancing, sticky sessions, distributed control-plane state, automatic certificates, retries, circuit breaking, WAF functionality, per-tenant authentication, or formal capacity/availability guarantees.

Those are the next product tiers rather than hidden or fabricated capabilities.

## License

See `LICENSE`.
