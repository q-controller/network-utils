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

func TestDNSForwarder_Integration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Pick a random free port
	ln, listErr := net.Listen("tcp", fmt.Sprintf("%s:0", "127.0.0.1"))
	assert.NoError(t, listErr)
	addr := ln.Addr().(*net.TCPAddr)
	_ = ln.Close()
	forwarderAddr := fmt.Sprintf("%s:%d", "127.0.0.1", addr.Port)
	forwarder, err := NewDNSForwarder(ctx,
		WithForwarderAddress(forwarderAddr),
		WithForwarderTimeout(2*time.Second),
		WithResolvconfPath("/etc/resolv.conf"),
	)
	require.NoError(t, err)
	defer forwarder.Stop()

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
