package network

import (
	"fmt"
	"net"

	"github.com/q-controller/network-utils/src/utils/network/ifc"
)

type NetworkConfig struct {
	Name        string
	Subnet      *net.IPNet
	GatewayIp   net.IP
	BridgeIp    net.IP
	LinkManager ifc.LinkManager
}

type NetworkOption func(*NetworkConfig) error

func WithName(name string) NetworkOption {
	return func(n *NetworkConfig) error {
		n.Name = name
		return nil
	}
}

func WithSubnet(ipNet *net.IPNet) NetworkOption {
	return func(n *NetworkConfig) error {
		n.Subnet = ipNet
		return nil
	}
}

func WithGateway(ip net.IP) NetworkOption {
	return func(n *NetworkConfig) error {
		n.GatewayIp = ip
		return nil
	}
}

func WithBridge(ip net.IP) NetworkOption {
	return func(n *NetworkConfig) error {
		n.BridgeIp = ip
		return nil
	}
}

func WithLinkManager(manager ifc.LinkManager) NetworkOption {
	return func(n *NetworkConfig) error {
		n.LinkManager = manager
		return nil
	}
}

func (n *NetworkConfig) validate() error {
	if len(n.Name) == 0 {
		return fmt.Errorf("network name is required")
	}

	if n.Subnet == nil {
		return fmt.Errorf("network Subnet is required")
	}

	if n.GatewayIp == nil {
		return fmt.Errorf("gateway IP is required")
	}

	if n.BridgeIp == nil {
		return fmt.Errorf("bridge IP is required")
	}

	if !n.Subnet.Contains(n.GatewayIp) {
		return fmt.Errorf("gateway IP %s is not in network %s", n.GatewayIp.String(), n.Subnet.String())
	}

	if !n.Subnet.Contains(n.BridgeIp) {
		return fmt.Errorf("bridge IP %s is not in network %s", n.BridgeIp.String(), n.Subnet.String())
	}

	if n.LinkManager == nil {
		return fmt.Errorf("link manager is required")
	}

	return nil
}
