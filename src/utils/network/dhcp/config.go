package dhcp

import (
	"fmt"
	"net"
	"time"

	"github.com/q-controller/network-utils/src/utils/network/address"
)

type DHCPConfig struct {
	RangeStart net.IP
	RangeEnd   net.IP
	LeaseTime  time.Duration
	DNS        []net.IP
	Router     net.IP
	Subnet     *net.IPNet
	LeaseFile  string
}

type DHCPOption func(*DHCPConfig) error

func WithInterface(ifaceName string, routerIP net.IP) DHCPOption {
	return func(cfg *DHCPConfig) error {
		iface, ifaceErr := net.InterfaceByName(ifaceName)
		if ifaceErr != nil {
			return fmt.Errorf("failed to get interface %s: %v", ifaceName, ifaceErr)
		}

		addrs, addrsErr := iface.Addrs()
		if addrsErr != nil {
			return fmt.Errorf("failed to get addresses for interface %s: %v", ifaceName, addrsErr)
		}

		var foundSubnet *net.IPNet
		for _, addr := range addrs {
			if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
				if ipNet.IP.To4() != nil && ipNet.IP.Equal(routerIP) {
					cfg.Router = routerIP
					foundSubnet = ipNet
					break
				}
			}
		}

		if foundSubnet == nil {
			return fmt.Errorf("router IP %s not found as address on interface %s", routerIP.String(), ifaceName)
		}

		cfg.Subnet = foundSubnet
		return nil
	}
}

func WithRange(start, end net.IP) DHCPOption {
	return func(cfg *DHCPConfig) error {
		if start == nil || end == nil {
			return fmt.Errorf("start and end IPs must not be nil")
		}

		cfg.RangeStart = start
		cfg.RangeEnd = end
		return nil
	}
}

func WithLeaseTime(duration time.Duration) DHCPOption {
	return func(cfg *DHCPConfig) error {
		cfg.LeaseTime = duration
		return nil
	}
}

func WithDNS(ips ...net.IP) DHCPOption {
	return func(cfg *DHCPConfig) error {
		cfg.DNS = ips
		return nil
	}
}

func WithLeaseFile(filePath string) DHCPOption {
	return func(cfg *DHCPConfig) error {
		cfg.LeaseFile = filePath
		return nil
	}
}

func (c *DHCPConfig) validate() error {
	if c.RangeStart == nil || c.RangeEnd == nil {
		return fmt.Errorf("IP range must be specified")
	}

	if c.Router == nil {
		return fmt.Errorf("router IP must be specified")
	}

	if !address.IsValidRange(c.RangeStart, c.RangeEnd, c.Subnet) {
		return fmt.Errorf("invalid IP range: %s - %s", c.RangeStart, c.RangeEnd)
	}

	return nil
}
