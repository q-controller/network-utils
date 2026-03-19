package dns

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDNSForwarder_StaticUpstreams(t *testing.T) {
	// Start a fake upstream DNS server
	upstreamAddr := startFakeUpstream(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ln, listErr := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, listErr)
	addr := ln.Addr().(*net.TCPAddr)
	_ = ln.Close()
	forwarderAddr := fmt.Sprintf("127.0.0.1:%d", addr.Port)

	forwarder, err := NewDNSFailoverForwarder(ctx,
		WithForwarderAddress(forwarderAddr),
		WithForwarderTimeout(2*time.Second),
		WithUpstreams([]string{upstreamAddr}),
	)
	require.NoError(t, err)
	stop, serveErr := forwarder.Serve()
	require.NoError(t, serveErr)
	defer stop()

	time.Sleep(200 * time.Millisecond)

	c := new(dns.Client)
	m := new(dns.Msg)
	m.SetQuestion("test.invalid.", dns.TypeA)
	resp, _, err := c.Exchange(m, forwarderAddr)
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, dns.RcodeSuccess, resp.Rcode)
	require.Len(t, resp.Answer, 1)
	a, ok := resp.Answer[0].(*dns.A)
	require.True(t, ok)
	assert.Equal(t, net.ParseIP("10.0.0.1").To4(), a.A.To4())
}

func startFakeUpstream(t *testing.T) string {
	t.Helper()
	ln, err := net.ListenPacket("udp", "127.0.0.1:0")
	require.NoError(t, err)

	srv := &dns.Server{
		PacketConn: ln,
		Handler: dns.HandlerFunc(func(w dns.ResponseWriter, r *dns.Msg) {
			resp := new(dns.Msg)
			resp.SetReply(r)
			resp.Answer = []dns.RR{
				&dns.A{
					Hdr: dns.RR_Header{
						Name:   r.Question[0].Name,
						Rrtype: dns.TypeA,
						Class:  dns.ClassINET,
						Ttl:    60,
					},
					A: net.ParseIP("10.0.0.1"),
				},
			}
			_ = w.WriteMsg(resp)
		}),
	}
	go func() { _ = srv.ActivateAndServe() }()
	t.Cleanup(func() { _ = srv.Shutdown() })
	time.Sleep(100 * time.Millisecond)
	return ln.LocalAddr().String()
}

func TestDNSForwarder_Integration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Pick a random free port
	ln, listErr := net.Listen("tcp", fmt.Sprintf("%s:0", "127.0.0.1"))
	assert.NoError(t, listErr)
	addr := ln.Addr().(*net.TCPAddr)
	_ = ln.Close()
	forwarderAddr := fmt.Sprintf("%s:%d", "127.0.0.1", addr.Port)
	forwarder, err := NewDNSFailoverForwarder(ctx,
		WithForwarderAddress(forwarderAddr),
		WithForwarderTimeout(2*time.Second),
		WithResolvconfPath("/etc/resolv.conf"),
	)
	require.NoError(t, err)
	stop, serveErr := forwarder.Serve()
	require.NoError(t, serveErr)
	defer stop()

	// Wait a moment for the server to start
	time.Sleep(200 * time.Millisecond)

	// Prepare a DNS client and query
	c := new(dns.Client)
	m := new(dns.Msg)
	m.SetQuestion("example.com.", dns.TypeA)
	resp, _, err := c.Exchange(m, forwarderAddr)
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, dns.RcodeSuccess, resp.Rcode)
}
