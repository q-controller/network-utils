package dns

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"

	// Import CoreDNS plugins
	_ "github.com/coredns/coredns/plugin/bind"
	_ "github.com/coredns/coredns/plugin/errors"
	_ "github.com/coredns/coredns/plugin/forward"
	_ "github.com/coredns/coredns/plugin/log"
)

func init() {
	// Register the DNS server type with Caddy
	dnsserver.Directives = []string{
		"bind",
		"forward",
		"log",
		"errors",
	}
}

type CoreDNSServer struct {
	instance *caddy.Instance
	once     sync.Once
}

func NewCoreDNSServer(ctx context.Context, options ...DNSForwarderOption) (DNSForwarder, error) {
	filename := ResolvConfPath
	if _, statErr := os.Stat(SystemdResolvConfPath); statErr == nil {
		filename = SystemdResolvConfPath
	}
	cfg := &DNSForwarderConfig{
		Timeout:        2 * time.Minute,
		ResolvconfPath: filename,
	}

	for _, opt := range options {
		opt(cfg)
	}
	if cfg.Address == "" {
		return nil, fmt.Errorf("DNS forwarder address not specified")
	}

	// Build Corefile content
	corefile := buildCorefile(cfg)
	slog.Debug("Starting CoreDNS", "corefile", corefile)

	// Create input for Caddy
	input := caddy.CaddyfileInput{
		Contents:       []byte(corefile),
		Filepath:       "Corefile",
		ServerTypeName: "dns",
	}

	instance, err := caddy.Start(input)
	if err != nil {
		return nil, fmt.Errorf("failed to start CoreDNS: %w", err)
	}

	return &CoreDNSServer{
		instance: instance,
	}, nil
}

func buildCorefile(cfg *DNSForwarderConfig) string {
	zone := "."
	if cfg.Zone != "" {
		zone = cfg.Zone
	}

	// Extract IP and port from address
	ip := cfg.Address
	port := "53"
	if idx := strings.LastIndex(cfg.Address, ":"); idx != -1 {
		ip = cfg.Address[:idx]
		port = cfg.Address[idx+1:]
	}

	return fmt.Sprintf(`%s:%s {
    bind %s
    forward . %s
    log
    errors
}
`, zone, port, ip, cfg.ResolvconfPath)
}

func (s *CoreDNSServer) Stop() {
	s.once.Do(func() {
		if s.instance != nil {
			s.instance.Stop()
			s.instance.Wait()
		}
	})
}
