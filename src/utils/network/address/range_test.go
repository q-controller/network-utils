package address

import (
	"net"
	"testing"
)

func TestIsValidRange(t *testing.T) {
	tests := []struct {
		name    string
		start   string
		end     string
		network string
		valid   bool
	}{
		{
			name:    "Valid range",
			start:   "192.168.1.10",
			end:     "192.168.1.20",
			network: "192.168.1.0/24",
			valid:   true,
		},
		{
			name:    "Start not in network",
			start:   "192.168.2.10",
			end:     "192.168.1.20",
			network: "192.168.1.0/24",
			valid:   false,
		},
		{
			name:    "End not in network",
			start:   "192.168.1.10",
			end:     "192.168.2.20",
			network: "192.168.1.0/24",
			valid:   false,
		},
		{
			name:    "End less than start",
			start:   "192.168.1.20",
			end:     "192.168.1.10",
			network: "192.168.1.0/24",
			valid:   false,
		},
		{
			name:    "Invalid start IP",
			start:   "invalid",
			end:     "192.168.1.20",
			network: "192.168.1.0/24",
			valid:   false,
		},
		{
			name:    "Invalid end IP",
			start:   "192.168.1.10",
			end:     "invalid",
			network: "192.168.1.0/24",
			valid:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, network, _ := net.ParseCIDR(tt.network)
			start := net.ParseIP(tt.start)
			end := net.ParseIP(tt.end)
			if got := IsValidRange(start, end, network); got != tt.valid {
				t.Errorf("IsValidRange() = %v, want %v", got, tt.valid)
			}
		})
	}
}
