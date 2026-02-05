package dns

import (
	"time"
)

type DNSForwarder interface {
	Stop()
}

type DNSForwarderConfig struct {
	Address        string
	Timeout        time.Duration
	ResolvconfPath string
	Zone           string
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
