package dns

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
)

type DNSFailoverForwarder struct {
	once sync.Once
	udp  *dns.Server
	tcp  *dns.Server
}

func (d *DNSFailoverForwarder) Stop() {
	d.once.Do(func() {
		if err := d.udp.Shutdown(); err != nil {
			slog.Error("Failed to shutdown UDP DNS server", "error", err)
		}
		if err := d.tcp.Shutdown(); err != nil {
			slog.Error("Failed to shutdown TCP DNS server", "error", err)
		}
	})
}

func NewDNSFailoverForwarder(ctx context.Context, options ...DNSForwarderOption) (DNSForwarder, error) {
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
	address := cfg.Address
	if !strings.Contains(address, ":") {
		address = fmt.Sprintf("%s:53", address)
	}

	dnsHandler := NewDnsHandler(
		WithTimeout(cfg.Timeout),
	)

	handler := dns.NewServeMux()
	handler.Handle(".", dnsHandler)
	tcp := &dns.Server{Net: "tcp", Addr: address, Handler: handler}
	udp := &dns.Server{Net: "udp", Addr: address, Handler: handler}

	upstreamsCh, upstreamsErr := GetUpstreamDNSFromFile(ctx, cfg.ResolvconfPath)
	if upstreamsErr != nil {
		return nil, upstreamsErr
	}

	go func() {
		for upstreams := range upstreamsCh {
			slog.Info("Updating DNS upstreams", "upstreams", upstreams.Endpoints, "error", upstreams.Error)
			dnsHandler.Upstreams.Store(upstreams.Endpoints)
		}
	}()

	go func() {
		if err := udp.ListenAndServe(); err != nil {
			slog.Error("UDP DNS server failed", "error", err)
		}
	}()

	go func() {
		if err := tcp.ListenAndServe(); err != nil {
			slog.Error("TCP DNS server failed", "error", err)
		}
	}()

	return &DNSFailoverForwarder{
		udp: udp,
		tcp: tcp,
	}, nil
}
