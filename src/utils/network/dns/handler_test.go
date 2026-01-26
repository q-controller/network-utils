package dns

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
)

type testResponseWriter struct {
	msg *dns.Msg
}

func (w *testResponseWriter) WriteMsg(m *dns.Msg) error {
	w.msg = m
	return nil
}

func (w *testResponseWriter) Write(b []byte) (int, error) {
	return len(b), nil
}
func (w *testResponseWriter) RemoteAddr() net.Addr {
	return &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 12345}
}
func (w *testResponseWriter) LocalAddr() net.Addr { return nil }
func (w *testResponseWriter) Close() error        { return nil }
func (w *testResponseWriter) TsigStatus() error   { return nil }
func (w *testResponseWriter) TsigTimersOnly(bool) {}
func (w *testResponseWriter) Hijack()             {}

func TestDnsHandler_EmptyUpstreams(t *testing.T) {
	h := NewDnsHandler(
		WithTimeout(time.Second),
	)
	w := &testResponseWriter{}
	r := new(dns.Msg)
	r.SetQuestion("example.com.", dns.TypeA)
	h.ServeDNS(w, r)
	assert.NotNil(t, w.msg, "expected a response message")
	if w.msg != nil {
		assert.Equal(t, dns.RcodeServerFailure, w.msg.Rcode, "expected SERVFAIL")
	}
}

func TestDnsHandler_NonEmptyFailingUpstreams(t *testing.T) {
	h := NewDnsHandler(
		WithTimeout(time.Second),
	)
	h.Upstreams.Store([]string{"127.0.0.1:53"})
	w := &testResponseWriter{}
	r := new(dns.Msg)
	r.SetQuestion("example.com.", dns.TypeA)
	h.ServeDNS(w, r)
	assert.NotNil(t, w.msg, "expected a response message")
	if w.msg != nil {
		assert.Equal(t, dns.RcodeServerFailure, w.msg.Rcode, "expected SERVFAIL")
	}
}

type testDnsClient struct{}

func (c *testDnsClient) ExchangeContext(ctx context.Context, m *dns.Msg, a string) (r *dns.Msg, rtt time.Duration, err error) {
	if a == "TEST" {
		resp := new(dns.Msg)
		resp.SetReply(m)
		resp.Answer = []dns.RR{
			&dns.A{
				Hdr: dns.RR_Header{
					Name:   m.Question[0].Name,
					Rrtype: dns.TypeA,
					Class:  dns.ClassINET,
					Ttl:    300,
				},
				A: net.ParseIP("127.0.0.1"),
			},
		}
		return resp, 0, nil
	}
	return nil, 0, fmt.Errorf("test client failed")
}

func TestDnsHandler_Responses(t *testing.T) {
	factory := func(net string, timeout time.Duration) DNSClient {
		return &testDnsClient{}
	}
	cases := []struct {
		name      string
		upstreams []string
		factory   ClientFactory
		expected  int
	}{
		{"EmptyUpstreams", nil, nil, dns.RcodeServerFailure},
		{"FailingUpstreams", []string{"127.0.0.1:53"}, nil, dns.RcodeServerFailure},
		{"MockSuccess", []string{"TEST"}, factory, dns.RcodeSuccess},
		{"MockFailing", []string{"8.8.8.8:53", "1.1.1.1:53"}, factory, dns.RcodeServerFailure},
		{"MockMixed", []string{"8.8.8.8:53", "TEST"}, factory, dns.RcodeSuccess},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			opts := []DnsOption{WithTimeout(time.Second)}
			if tc.factory != nil {
				opts = append(opts, WithClientFactory(tc.factory))
			}
			h := NewDnsHandler(opts...)
			if tc.upstreams != nil {
				h.Upstreams.Store(tc.upstreams)
			}
			w := &testResponseWriter{}
			r := new(dns.Msg)
			r.SetQuestion("example.com.", dns.TypeA)
			h.ServeDNS(w, r)
			assert.NotNil(t, w.msg, "expected a response message")
			if w.msg != nil {
				assert.Equal(t, tc.expected, w.msg.Rcode)
			}
		})
	}
}
