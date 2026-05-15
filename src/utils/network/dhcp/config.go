//go:build linux

package dhcp

import (
	"errors"
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

func WithRouter(routerIP net.IP) DHCPOption {
	return func(cfg *DHCPConfig) error {
		ifaces, ifacesErr := net.Interfaces()
		if ifacesErr != nil {
			return fmt.Errorf("failed to list interfaces: %w", ifacesErr)
		}

		for _, iface := range ifaces {
			addrs, addrsErr := iface.Addrs()
			if addrsErr != nil {
				continue
			}
			for _, addr := range addrs {
				ipNet, ok := addr.(*net.IPNet)
				if !ok || ipNet.IP.IsLoopback() || ipNet.IP.To4() == nil {
					continue
				}
				if ipNet.IP.Equal(routerIP) {
					cfg.Router = routerIP
					cfg.Subnet = ipNet
					return nil
				}
			}
		}

		return fmt.Errorf("router IP %s not found on any host interface", routerIP.String())
	}
}

func WithRange(start, end net.IP) DHCPOption {
	return func(cfg *DHCPConfig) error {
		if start == nil || end == nil {
			return errors.New("start and end IPs must not be nil")
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
		return errors.New("IP range must be specified")
	}

	if c.Router == nil {
		return errors.New("router IP must be specified")
	}

	if !address.IsValidRange(c.RangeStart, c.RangeEnd, c.Subnet) {
		return fmt.Errorf("invalid IP range: %s - %s", c.RangeStart, c.RangeEnd)
	}

	return nil
}
