package dns

import (
	"time"
)

type DNSForwarder interface {
	Serve() (stop func(), err error)
}

type DNSForwarderConfig struct {
	Address        string
	Timeout        time.Duration
	ResolvconfPath string
	Zone           string
	Upstreams      []string
	ReusePort      bool
}

type DNSForwarderOption func(*DNSForwarderConfig)

func WithForwarderTimeout(timeout time.Duration) DNSForwarderOption {
	return func(cfg *DNSForwarderConfig) {
		cfg.Timeout = timeout
	}
}

func WithForwarderAddress(address string) DNSForwarderOption {
	return func(cfg *DNSForwarderConfig) {
		cfg.Address = address
	}
}

func WithResolvconfPath(path string) DNSForwarderOption {
	return func(cfg *DNSForwarderConfig) {
		cfg.ResolvconfPath = path
	}
}

func WithForwarderZone(zone string) DNSForwarderOption {
	return func(cfg *DNSForwarderConfig) {
		cfg.Zone = zone
	}
}

func WithUpstreams(upstreams []string) DNSForwarderOption {
	return func(cfg *DNSForwarderConfig) {
		cfg.Upstreams = upstreams
	}
}

func WithReusePort() DNSForwarderOption {
	return func(cfg *DNSForwarderConfig) {
		cfg.ReusePort = true
	}
}
