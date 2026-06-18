# Go-Load-Balancer

## Overview
High-performance reverse proxy and load balancer in Go using round-robin distribution.

## Quick Start (1-Click Build)

```bash
git clone https://github.com/skylerblue333/Go-Load-Balancer.git
cd Go-Load-Balancer
go run main.go
```

## Features
- Round-robin load balancing
- Reverse proxy via `net/http/httputil`
- Atomic counter for thread-safe routing
