package ifc

import (
	"errors"
	"log/slog"
	"net"
	"sync"

	"github.com/vishvananda/netlink"
	"github.com/vishvananda/netlink/nl"
)

var ErrNetworkDisconnected = errors.New("network disconnected")

// GetDefaultInterface finds the default network interface
func GetDefaultInterface() (string, error) {
	routes, err := netlink.RouteList(nil, nl.FAMILY_V4)
	if err != nil {
		return "", err
	}

	for _, route := range routes {
		if route.Dst.IP.IsUnspecified() && route.Dst.Mask.String() == net.CIDRMask(0, 32).String() {
			link, err := netlink.LinkByIndex(route.LinkIndex)
			if err != nil {
				continue
			}
			return link.Attrs().Name, nil
		}
	}
	return "", ErrNetworkDisconnected
}

type InterfaceSubscription struct {
	InterfaceCh <-chan string
	stopCh      chan struct{}
	stopOnce    sync.Once
}

func (s *InterfaceSubscription) Stop() {
	s.stopOnce.Do(func() {
		close(s.stopCh)
	})
}

func SubscribeDefaultInterfaceChanges() (*InterfaceSubscription, error) {
	stopCh := make(chan struct{})
	ifcCh := make(chan string, 1) // Buffered to prevent blocking on initial send
	updates := make(chan netlink.LinkUpdate)

	if subscribeErr := netlink.LinkSubscribe(updates, stopCh); subscribeErr != nil {
		return nil, subscribeErr
	}

	subscription := &InterfaceSubscription{
		InterfaceCh: ifcCh,
		stopCh:      stopCh,
	}

	go func() {
		defer close(ifcCh)
		defer close(updates)

		currentDefaultInterface, defaultInterfaceErr := GetDefaultInterface()
		if defaultInterfaceErr != nil {
			slog.Debug("Failed to get initial default interface", "error", defaultInterfaceErr)
			return
		}

		// Send initial interface
		select {
		case ifcCh <- currentDefaultInterface:
		case <-stopCh:
			return
		}

		for {
			select {
			case <-updates:
				newDefaultInterface := ""
				if defaultInterface, defaultInterfaceErr := GetDefaultInterface(); defaultInterfaceErr == nil {
					newDefaultInterface = defaultInterface
				}

				if currentDefaultInterface != newDefaultInterface {
					if newDefaultInterface == "" {
						slog.Debug("Got disconnected from the internet")
					} else {
						slog.Debug("New default interface was configured", "interface", newDefaultInterface)
					}
					currentDefaultInterface = newDefaultInterface

					// Send updated interface only when it changes
					select {
					case ifcCh <- currentDefaultInterface:
					case <-stopCh:
						return
					}
				}
			case <-stopCh:
				return
			}
		}
	}()

	return subscription, nil
}
