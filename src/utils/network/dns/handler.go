package dns

import (
	"context"
	"log/slog"
	"net"
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

const systemResolverTTL = 60

// resolveViaSystem attempts to resolve A/AAAA queries using the OS's native resolver
// (getaddrinfo, /etc/hosts, mDNS, search domains, etc.).
//
// Returns a valid *dns.Msg on success (with answer records), or nil if:
// - not an A/AAAA query
// - lookup failed or returned no addresses
// - any other case where we should fall back to upstream servers
func resolveViaSystem(ctx context.Context, r *dns.Msg) *dns.Msg {
	if len(r.Question) != 1 {
		return nil // don't attempt multi-question
	}

	q := r.Question[0]
	if q.Qtype != dns.TypeA && q.Qtype != dns.TypeAAAA {
		return nil // let upstream handle MX, TXT, etc.
	}

	name := q.Name

	addrs, err := net.DefaultResolver.LookupHost(ctx, name)
	if err != nil {
		slog.Debug("system resolver LookupHost failed", "name", name, "err", err)
		return nil
	}

	if len(addrs) == 0 {
		// Name resolved but no A/AAAA → NODATA, but upstream might have different view
		return nil
	}

	resp := new(dns.Msg)
	resp.SetReply(r)
	resp.RecursionAvailable = true
	resp.Compress = true
	resp.Authoritative = false

	for _, addrStr := range addrs {
		ip := net.ParseIP(addrStr)
		if ip == nil {
			continue
		}

		if ip4 := ip.To4(); ip4 != nil && q.Qtype == dns.TypeA {
			resp.Answer = append(resp.Answer, &dns.A{
				Hdr: dns.RR_Header{
					Name:   name,
					Rrtype: dns.TypeA,
					Class:  dns.ClassINET,
					Ttl:    systemResolverTTL,
				},
				A: ip4,
			})
		} else if ip.To4() == nil && q.Qtype == dns.TypeAAAA { // IPv6 check
			resp.Answer = append(resp.Answer, &dns.AAAA{
				Hdr: dns.RR_Header{
					Name:   name,
					Rrtype: dns.TypeAAAA,
					Class:  dns.ClassINET,
					Ttl:    systemResolverTTL,
				},
				AAAA: ip,
			})
		}
	}

	if len(resp.Answer) == 0 {
		// LookupHost returned addresses but none matched the query type
		// (e.g. only IPv4 for an AAAA query). Return NODATA (NOERROR + empty answer)
		// rather than falling through to upstreams which may SERVFAIL.
		slog.Debug("Resolved via system, no matching records", "question", q)
		return resp
	}

	slog.Debug("Resolved via system", "question", q, "answers", len(resp.Answer))
	return resp
}

func (d *dnsHandler) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	ctx, cancel := context.WithTimeout(context.Background(), d.Timeout)
	defer cancel()

	// Try system resolver first for A/AAAA queries.
	// This uses getaddrinfo() which resolves /etc/hosts, mDNS, split-DNS, etc.
	if resp := resolveViaSystem(ctx, r); resp != nil {
		slog.Debug("Resolved via system resolver", "question", r.Question, "answer", resp.Answer)
		_ = w.WriteMsg(resp)
		return
	}

	// Fall back to upstream forwarding for other query types or if system resolver failed.
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

		resp, _, respErr := client.ExchangeContext(ctx, r, up)

		if respErr != nil {
			slog.Error("DNS query failed", "upstream", up, "error", respErr)
			continue
		}

		if resp == nil {
			continue
		}

		lastResp = resp

		if resp.Rcode == dns.RcodeSuccess {
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
