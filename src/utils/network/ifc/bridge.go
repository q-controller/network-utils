//go:build linux
// +build linux

package ifc

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"syscall"
)

func getFirstUsableIP(ipNet *net.IPNet) net.IP {
	if ipNet == nil {
		return nil
	}
	// Start with the network address
	firstIP := ipNet.IP

	// Create a copy to avoid modifying the original
	firstUsable := make(net.IP, len(firstIP))
	copy(firstUsable, firstIP)

	for i := len(firstUsable) - 1; i >= 0; i-- {
		firstUsable[i]++
		if firstUsable[i] != 0 {
			break
		}
	}

	return firstUsable
}

func CreateBridgeWithManager(mgr LinkManager, name string, subnet string, disableTxOffloading bool) error {
	_, ipnet, ipErr := net.ParseCIDR(subnet)
	if ipErr != nil {
		return fmt.Errorf("failed to parse subnet %s: %v", subnet, ipErr)
	}

	ip := getFirstUsableIP(ipnet)
	if ip == nil {
		return fmt.Errorf("failed to get first usable IP")
	}

	if addBridgeErr := mgr.AddLink(name, LinkTypeBridge); addBridgeErr != nil {
		if errors.Is(addBridgeErr, syscall.EEXIST) {
			slog.Debug("Link already exists")
			hasIP, ipErr := mgr.HasIP(name, ip, ipnet.Mask)
			if ipErr != nil {
				return fmt.Errorf("failed to list interface addresses: %w", ipErr)
			}
			if hasIP {
				return nil
			}
		} else {
			return fmt.Errorf("failed to add bridge %s: %v", name, addBridgeErr)
		}
	}

	if addrErr := mgr.SetIP(name, ip, ipnet.Mask); addrErr != nil {
		if delErr := mgr.DeleteLink(name); delErr != nil {
			return fmt.Errorf("failed to set ip: %v, failed to delete link: %v", addrErr, delErr)
		}
		return fmt.Errorf("failed to set ip: %v", addrErr)
	}

	if upErr := mgr.BringUp(name); upErr != nil {
		if delErr := mgr.DeleteLink(name); delErr != nil {
			return fmt.Errorf("failed to bring bridge %s up: %v, failed to delete link: %v", name, upErr, delErr)
		}
		return fmt.Errorf("failed to bring bridge %s up: %v", name, upErr)
	}

	slog.Debug("successfully created bridge", "name", name)

	if disableTxOffloading {
		return mgr.DisableTxOffloading(name)
	}

	return nil
}

func CreateBridge(name string, subnet string, disableTxOffloading bool) error {
	return CreateBridgeWithManager(NetlinkBridgeManager{}, name, subnet, disableTxOffloading)
}
