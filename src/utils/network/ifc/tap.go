package ifc

import (
	"fmt"
	"log/slog"
)

func CreateTapWithManager(mgr LinkManager, name string, bridgeName string) error {
	exists, existsErr := mgr.Exists(name)
	if existsErr != nil {
		return fmt.Errorf("unexpected error checking link: %v", existsErr)
	}
	if !exists {
		if err := mgr.AddLink(name, LinkTypeTap); err != nil {
			return fmt.Errorf("failed to add tap device %s: %v", name, err)
		}
		if err := mgr.SetMaster(name, bridgeName); err != nil {
			if delErr := mgr.DeleteLink(name); delErr != nil {
				return fmt.Errorf("failed to set master for tap %s: %v, failed to delete tap: %v", name, err, delErr)
			}
			return fmt.Errorf("failed to set master for tap %s: %v", name, err)
		}
	}

	if err := mgr.BringUp(name); err != nil {
		if delErr := mgr.DeleteLink(name); delErr != nil {
			return fmt.Errorf("failed to bring tap %s up: %v, failed to delete tap: %v", name, err, delErr)
		}
		return fmt.Errorf("failed to bring tap %s up: %v", name, err)
	}

	slog.Debug("successfully added tap", "tap", name, "bridge", bridgeName)
	return nil
}

func CreateTap(name string, bridgeName string) error {
	return CreateTapWithManager(NetlinkBridgeManager{}, name, bridgeName)
}
