package ifc

import (
	"fmt"
	"net"
	"os/exec"

	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netlink/nl"
)

type NetlinkBridgeManager struct{}

func (NetlinkBridgeManager) AddLink(name string, linkType LinkType) error {
	var link netlink.Link
	switch linkType {
	case LinkTypeBridge:
		bridgeAttrs := netlink.NewLinkAttrs()
		bridgeAttrs.Name = name
		link = &netlink.Bridge{LinkAttrs: bridgeAttrs}
	case LinkTypeTap:
		link = &netlink.Tuntap{
			Mode: netlink.TUNTAP_MODE_TAP,
			LinkAttrs: netlink.LinkAttrs{
				Name: name,
			},
		}
	default:
		return fmt.Errorf("unsupported link type: %s", linkType)
	}

	return netlink.LinkAdd(link)
}

func (NetlinkBridgeManager) SetIP(name string, ip net.IP, mask net.IPMask) error {
	link, linkErr := netlink.LinkByName(name)
	if linkErr != nil {
		return linkErr
	}

	addr := &netlink.Addr{IPNet: &net.IPNet{IP: ip, Mask: mask}}
	return netlink.AddrReplace(link, addr)
}

func (NetlinkBridgeManager) Exists(name string) (bool, error) {
	if _, linkErr := netlink.LinkByName(name); linkErr != nil {
		if _, ok := linkErr.(netlink.LinkNotFoundError); ok {
			return false, nil
		}
		return false, fmt.Errorf("error checking if link exists: %v", linkErr)
	}

	return true, nil
}

func (NetlinkBridgeManager) SetMaster(name string, masterName string) error {
	link, linkErr := netlink.LinkByName(name)
	if linkErr != nil {
		return linkErr
	}
	masterLink, masterErr := netlink.LinkByName(masterName)
	if masterErr != nil {
		return masterErr
	}
	return netlink.LinkSetMaster(link, masterLink)
}

func (NetlinkBridgeManager) BringUp(name string) error {
	link, linkErr := netlink.LinkByName(name)
	if linkErr != nil {
		return linkErr
	}
	return netlink.LinkSetUp(link)
}

func (NetlinkBridgeManager) HasIP(name string, ip net.IP, mask net.IPMask) (bool, error) {
	link, linkErr := netlink.LinkByName(name)
	if linkErr != nil {
		return false, linkErr
	}
	addresses, addrErr := netlink.AddrList(link, nl.FAMILY_V4)
	if addrErr != nil {
		return false, addrErr
	}
	for _, addr := range addresses {
		if addr.Equal(netlink.Addr{IPNet: &net.IPNet{IP: ip, Mask: mask}}) {
			return true, nil
		}
	}
	return false, nil
}

func (NetlinkBridgeManager) DeleteLink(name string) error {
	link, linkErr := netlink.LinkByName(name)
	if linkErr != nil {
		if _, ok := linkErr.(netlink.LinkNotFoundError); ok {
			return nil // Link does not exist, nothing to delete
		}
		return linkErr
	}
	return netlink.LinkDel(link)
}

func (NetlinkBridgeManager) DisableTxOffloading(name string) error {
	cmd := exec.Command("ethtool", "-K", name, "tx", "off")
	return cmd.Run()
}

func DeleteLink(name string) error {
	return NetlinkBridgeManager{}.DeleteLink(name)
}
