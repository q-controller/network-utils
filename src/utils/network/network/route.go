//go:build linux

package network

import (
	"fmt"
	"net"

	"github.com/vishvananda/netlink"
)

func SetDefaultRoute(iface string, gatewayIP net.IP) error {
	link, err := netlink.LinkByName(iface)
	if err != nil {
		return fmt.Errorf("failed to get link: %w", err)
	}
	route := &netlink.Route{
		Dst:       nil,
		Gw:        gatewayIP,
		LinkIndex: link.Attrs().Index,
	}

	return netlink.RouteReplace(route)
}
