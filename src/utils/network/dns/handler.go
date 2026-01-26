package dns

import (
	"context"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/miekg/dns"
)

type DNSClient interface {
	ExchangeContext(ctx context.Context, m *dns.Msg, address string) (r *dns.Msg, rtt time.Duration, err error)
}

type ClientFactory func(net string, timeout time.Duration) DNSClient

type dnsHandler struct {
	Upstreams     atomic.Value // will store []string
	Timeout       time.Duration
	ClientFactory ClientFactory
}

func (d *dnsHandler) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	upstreams := []string{}
	if val := d.Upstreams.Load(); val != nil {
		upstreams = val.([]string)
	}

	var client DNSClient
	if d.ClientFactory != nil {
		client = d.ClientFactory(w.RemoteAddr().Network(), d.Timeout)
	} else {
		client = &dns.Client{
			Net:     w.RemoteAddr().Network(),
			Timeout: d.Timeout,
		}
	}

	var lastResp *dns.Msg // track the "best" negative
	for _, up := range upstreams {
		slog.Debug("Forwarding DNS query", "upstream", up, "question", r.Question)

		resp, _, respErr := client.ExchangeContext(context.Background(), r, up)

		if respErr != nil {
			slog.Error("DNS query failed", "upstream", up, "error", respErr)
			continue
		}

		if resp == nil {
			continue
		}

		lastResp = resp

		if resp.Rcode == dns.RcodeSuccess && len(resp.Answer) > 0 {
			slog.Debug("Received successful DNS response", "upstream", up, "answer", resp.Answer)
			_ = w.WriteMsg(resp)
			return
		}
	}

	if lastResp != nil {
		slog.Debug("Returning last DNS response", "rcode", lastResp.Rcode)
		_ = w.WriteMsg(lastResp)
		return
	}
	slog.Error("All upstreams failed completely")
	m := new(dns.Msg)
	m.SetReply(r)
	m.Rcode = dns.RcodeServerFailure
	_ = w.WriteMsg(m)
}

type DnsConfig struct {
	Timeout       time.Duration
	ClientFactory ClientFactory
}

type DnsOption func(*DnsConfig)

func WithTimeout(timeout time.Duration) DnsOption {
	return func(c *DnsConfig) {
		c.Timeout = timeout
	}
}

func WithClientFactory(factory ClientFactory) DnsOption {
	return func(c *DnsConfig) {
		c.ClientFactory = factory
	}
}

func NewDnsHandler(options ...DnsOption) *dnsHandler {
	cfg := &DnsConfig{}
	for _, opt := range options {
		opt(cfg)
	}
	return &dnsHandler{
		Timeout:       cfg.Timeout,
		ClientFactory: cfg.ClientFactory,
	}
}
