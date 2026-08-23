package main

import (
	"errors"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	ListenAddr          string
	Backends            []*url.URL
	HealthPath          string
	HealthInterval      time.Duration
	HealthTimeout       time.Duration
	UpstreamTimeout     time.Duration
	ReadHeaderTimeout   time.Duration
	IdleTimeout         time.Duration
	ShutdownGracePeriod time.Duration
}

func loadConfig() (Config, error) {
	backends, err := parseBackendURLs(envOr("BACKENDS", "http://localhost:8081,http://localhost:8082"))
	if err != nil {
		return Config{}, err
	}
	cfg := Config{
		ListenAddr:          envOr("LISTEN_ADDR", ":8080"),
		Backends:            backends,
		HealthPath:          envOr("HEALTH_PATH", "/"),
		HealthInterval:      durationEnv("HEALTH_INTERVAL", 5*time.Second),
		HealthTimeout:       durationEnv("HEALTH_TIMEOUT", 2*time.Second),
		UpstreamTimeout:     durationEnv("UPSTREAM_TIMEOUT", 30*time.Second),
		ReadHeaderTimeout:   durationEnv("READ_HEADER_TIMEOUT", 5*time.Second),
		IdleTimeout:         durationEnv("IDLE_TIMEOUT", 60*time.Second),
		ShutdownGracePeriod: durationEnv("SHUTDOWN_GRACE_PERIOD", 10*time.Second),
	}
	if cfg.HealthPath == "" || cfg.HealthPath[0] != '/' {
		return Config{}, errors.New("HEALTH_PATH must start with /")
	}
	return cfg, nil
}

func envOr(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func durationEnv(key string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	if parsed, err := time.ParseDuration(raw); err == nil && parsed > 0 {
		return parsed
	}
	if seconds, err := strconv.Atoi(raw); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}
	return fallback
}

func parseBackendURLs(raw string) ([]*url.URL, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, errors.New("BACKENDS must contain at least one URL")
	}
	items := strings.Split(raw, ",")
	backends := make([]*url.URL, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		candidate := strings.TrimSpace(item)
		parsed, err := url.Parse(candidate)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			return nil, errors.New("BACKENDS contains an invalid HTTP(S) URL")
		}
		parsed.Fragment = ""
		canonical := parsed.String()
		if _, exists := seen[canonical]; exists {
			return nil, errors.New("BACKENDS contains a duplicate URL")
		}
		seen[canonical] = struct{}{}
		backends = append(backends, parsed)
	}
	return backends, nil
}
