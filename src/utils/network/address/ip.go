package address

import (
	"net"
	"slices"
)

// getFirstUsableIP returns the first usable IP address in the given subnet.
// Returns nil if the input is nil or if there is no usable address in the subnet.
func GetFirstUsableIP(ipNet *net.IPNet) net.IP {
	if ipNet == nil {
		return nil
	}

	firstUsable := incrementIP(ipNet.IP)
	if !ipNet.Contains(firstUsable) {
		return nil
	}

	return firstUsable
}

func incrementIP(ip net.IP) net.IP {
	ip = slices.Clone(ip) // make a copy
	for i := len(ip) - 1; i >= 0; i-- {
		ip[i]++
		if ip[i] != 0 {
			break
		}
	}
	return ip
}
