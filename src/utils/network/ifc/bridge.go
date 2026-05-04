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

func CreateBridgeWithManager(mgr LinkManager, name string, gatewayCidr string, disableTxOffloading bool) error {
	ip, ipnet, ipErr := net.ParseCIDR(gatewayCidr)
	if ipErr != nil {
		return fmt.Errorf("invalid CIDR format: %w", ipErr)
	}

	if ip == nil {
		return fmt.Errorf("wrong IP address in CIDR: %s", gatewayCidr)
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
			return fmt.Errorf("failed to add bridge %s: %w", name, addBridgeErr)
		}
	}

	if addrErr := mgr.SetIP(name, ip, ipnet.Mask); addrErr != nil {
		if delErr := mgr.DeleteLink(name); delErr != nil {
			return fmt.Errorf("failed to set ip: %w, failed to delete link: %w", addrErr, delErr)
		}
		return fmt.Errorf("failed to set ip: %w", addrErr)
	}

	if upErr := mgr.BringUp(name); upErr != nil {
		if delErr := mgr.DeleteLink(name); delErr != nil {
			return fmt.Errorf("failed to bring bridge %s up: %w, failed to delete link: %w", name, upErr, delErr)
		}
		return fmt.Errorf("failed to bring bridge %s up: %w", name, upErr)
	}

	slog.Debug("successfully created bridge", "name", name)

	if disableTxOffloading {
		return mgr.DisableTxOffloading(name)
	}

	return nil
}

func CreateBridge(name string, gatewayCidr string, disableTxOffloading bool) error {
	return CreateBridgeWithManager(NetlinkBridgeManager{}, name, gatewayCidr, disableTxOffloading)
}
