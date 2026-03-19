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
	once        sync.Once
	udp         *dns.Server
	tcp         *dns.Server
	dnsHandler  *dnsHandler
	upstreamsCh <-chan UpstreamDNS
}

func (d *DNSFailoverForwarder) Serve() (func(), error) {
	if d.upstreamsCh != nil {
		go func() {
			for upstreams := range d.upstreamsCh {
				slog.Info("Updating DNS upstreams", "upstreams", upstreams.Endpoints, "error", upstreams.Error)
				d.dnsHandler.Upstreams.Store(upstreams.Endpoints)
			}
		}()
	}

	go func() {
		if err := d.udp.ListenAndServe(); err != nil {
			slog.Error("UDP DNS server failed", "error", err)
		}
	}()

	go func() {
		if err := d.tcp.ListenAndServe(); err != nil {
			slog.Error("TCP DNS server failed", "error", err)
		}
	}()

	stop := func() {
		d.once.Do(func() {
			if err := d.udp.Shutdown(); err != nil {
				slog.Error("Failed to shutdown UDP DNS server", "error", err)
			}
			if err := d.tcp.Shutdown(); err != nil {
				slog.Error("Failed to shutdown TCP DNS server", "error", err)
			}
		})
	}
	return stop, nil
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
	tcp := &dns.Server{Net: "tcp", Addr: address, Handler: handler, ReusePort: cfg.ReusePort}
	udp := &dns.Server{Net: "udp", Addr: address, Handler: handler, ReusePort: cfg.ReusePort}

	var upstreamsCh <-chan UpstreamDNS
	if len(cfg.Upstreams) > 0 {
		slog.Info("Using static DNS upstreams", "upstreams", cfg.Upstreams)
		dnsHandler.Upstreams.Store(cfg.Upstreams)
	} else {
		ch, upstreamsErr := GetUpstreamDNSFromFile(ctx, cfg.ResolvconfPath)
		if upstreamsErr != nil {
			return nil, upstreamsErr
		}
		upstreamsCh = ch
	}

	return &DNSFailoverForwarder{
		udp:         udp,
		tcp:         tcp,
		dnsHandler:  dnsHandler,
		upstreamsCh: upstreamsCh,
	}, nil
}
