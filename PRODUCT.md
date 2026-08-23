# Sky Edge Balancer — Product Definition

## Customer problem

Small engineering teams and self-hosted platform operators often need a tiny reverse-proxy/load-balancing layer without adopting a full service mesh or a large control plane. Sky Edge Balancer packages the existing Go data plane into a single container with health-aware routing, trace correlation, metrics, and orchestration-friendly probes.

## Ideal users

- internal application platforms;
- homelab/self-hosted stacks;
- small SaaS backends;
- development/staging clusters;
- SKYCOIN4444 service ingress tiers.

## Editions

### Community

Current open-source repository:

- round-robin balancing;
- health-aware routing;
- trace IDs;
- JSON metrics;
- health probes;
- Docker/Compose packaging;
- CI verification.

### Pro — roadmap

- weighted backends;
- retries and circuit breaking;
- TLS termination and automatic certificates;
- configuration reload without restart;
- Prometheus metrics;
- request/connection limits;
- structured JSON logs.

### Managed — roadmap

- hosted control plane;
- organization/tenant accounts;
- fleet configuration and remote rollout;
- usage analytics and alerts;
- managed certificates/domains;
- audit history;
- billing and support plans.

## Non-goals for the current release

This package is not marketed as a drop-in replacement for Envoy, HAProxy, NGINX, or cloud-provider L7 load balancers. Enterprise features remain roadmap items until implemented and verified.

## Release gate

A standalone product release requires all CI gates to pass on the exact tagged commit. Real traffic/capacity claims require benchmark evidence generated from documented test hardware and workload definitions.
