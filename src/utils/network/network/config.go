//go:build linux

package network

import (
	"errors"
	"fmt"
	"net"

	"github.com/q-controller/network-utils/src/utils/network/ifc"
)

type NetworkConfig struct {
	Name        string
	Subnet      *net.IPNet
	GatewayIP   net.IP
	BridgeIP    net.IP
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
		n.GatewayIP = ip
		return nil
	}
}

func WithBridge(ip net.IP) NetworkOption {
	return func(n *NetworkConfig) error {
		n.BridgeIP = ip
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
		return errors.New("network name is required")
	}

	if n.Subnet == nil {
		return errors.New("network Subnet is required")
	}

	if n.GatewayIP == nil {
		return errors.New("gateway IP is required")
	}

	if n.BridgeIP == nil {
		return errors.New("bridge IP is required")
	}

	if !n.Subnet.Contains(n.GatewayIP) {
		return fmt.Errorf("gateway IP %s is not in network %s", n.GatewayIP.String(), n.Subnet.String())
	}

	if !n.Subnet.Contains(n.BridgeIP) {
		return fmt.Errorf("bridge IP %s is not in network %s", n.BridgeIP.String(), n.Subnet.String())
	}

	if n.LinkManager == nil {
		return errors.New("link manager is required")
	}

	return nil
}
