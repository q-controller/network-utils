package address

import (
	"net"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetFirstUsableIP(t *testing.T) {
	tests := []struct {
		cidr     string
		expected net.IP
	}{
		{"192.168.1.0/24", net.IPv4(192, 168, 1, 1)},
		{"10.0.0.0/30", net.IPv4(10, 0, 0, 1)},
		{"0.0.0.0/32", nil},
		{"255.255.255.255/32", nil},
		{"", nil},
	}

	for _, tt := range tests {
		var ipNet *net.IPNet
		if tt.cidr != "" {
			_, ipNet, _ = net.ParseCIDR(tt.cidr)
		}
		result := GetFirstUsableIP(ipNet)
		if tt.expected == nil {
			require.Nil(t, result, "expected nil for %s", tt.cidr)
		} else {
			require.Equal(t, tt.expected.String(), result.String(), "unexpected IP for %s", tt.cidr)
		}
	}
}
