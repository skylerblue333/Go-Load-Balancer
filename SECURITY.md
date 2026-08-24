# Security Policy

## Supported scope

The current product is a small self-hosted HTTP reverse proxy/load balancer. Security reports should focus on request routing, backend validation, proxy behavior, container packaging, and denial-of-service weaknesses.

## Reporting

Please report vulnerabilities privately through GitHub's security reporting mechanisms when available. Do not include credentials, private keys, tokens, or production customer data in public issues.

## Current security boundaries

- backend URLs must include a scheme and host;
- health checks use a bounded timeout;
- the public server uses a request-header timeout and idle timeout;
- the container runs as a non-root distroless user;
- upstream proxy failures return a generic 502 rather than exposing the internal error;
- CI runs unit tests, `go vet`, the race detector, binary compilation, and Docker image construction.

## Known limitations

TLS termination, WAF rules, authentication, authorization, distributed rate limiting, retries, circuit breaking, and formal DDoS protections are not yet implemented. Deploy behind appropriate network/TLS controls when exposed to untrusted traffic.
