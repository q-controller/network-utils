package network

import (
	"fmt"
	"net"

	"github.com/vishvananda/netlink"
)

func SetDefaultRoute(iface string, gatewayIp net.IP) error {
	link, err := netlink.LinkByName(iface)
	if err != nil {
		return fmt.Errorf("failed to get link: %w", err)
	}
	route := &netlink.Route{
		Dst:       nil,
		Gw:        gatewayIp,
		LinkIndex: link.Attrs().Index,
	}

	return netlink.RouteAdd(route)
}
