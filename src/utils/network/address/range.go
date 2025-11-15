package address

import "net"

func IsValidRange(start, end net.IP, network *net.IPNet) bool {
	start = start.To4()
	end = end.To4()
	if start == nil || end == nil {
		return false
	}

	// Check if both IPs are in the same network
	if !network.Contains(start) || !network.Contains(end) {
		return false
	}

	// Convert to uint32 for comparison
	startInt := uint32(start[0])<<24 | uint32(start[1])<<16 | uint32(start[2])<<8 | uint32(start[3])
	endInt := uint32(end[0])<<24 | uint32(end[1])<<16 | uint32(end[2])<<8 | uint32(end[3])

	return endInt > startInt
}
