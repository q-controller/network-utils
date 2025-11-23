package network

import (
	"errors"
	"fmt"
	"net"

	"github.com/q-controller/network-utils/src/utils/network/firewall"
	"github.com/q-controller/network-utils/src/utils/network/ifc"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

func hostName(name string) string {
	return name + "-host"
}

func netName(name string) string {
	return name + "-net"
}

func getRulesForHostInterface(hostIf, hostLink string) (*firewall.Rules, error) {
	rules, rulesErr := firewall.NewRules(
		firewall.ForwardOutboundRule("FORWARD", "filter", hostIf, hostLink),
		firewall.ForwardReturnTrafficRule("FORWARD", "filter", hostIf, hostLink),
		firewall.MasqueradeRule("POSTROUTING", "nat", hostIf),
	)

	if rulesErr != nil {
		return nil, rulesErr
	}

	return rules, nil
}

type networkLinux struct {
	config *NetworkConfig
}

func (n *networkLinux) Destroy() error {
	var errs []error

	if delLinkErr := n.config.LinkManager.DeleteLink(hostName(n.config.Name)); delLinkErr != nil {
		errs = append(errs, delLinkErr)
	}

	if err := deleteNamespace(n.config.Name); err != nil {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

func (n *networkLinux) Execute(fn func() error) error {
	switchBack, switchErr := switchToNamespace(n.config.Name)
	if switchErr != nil {
		return switchErr
	}
	defer switchBack()

	return fn()
}

func (n *networkLinux) Connect(hostIf string) error {
	rules, rulesErr := getRulesForHostInterface(hostIf, hostName(n.config.Name))
	if rulesErr != nil {
		return rulesErr
	}
	if addedRules := firewall.AddRules(rules); addedRules != nil {
		return addedRules
	}

	return nil
}

func (n *networkLinux) Disconnect(hostIf string) error {
	rules, rulesErr := getRulesForHostInterface(hostIf, hostName(n.config.Name))
	if rulesErr != nil {
		return rulesErr
	}
	if delRules := firewall.RemoveRules(rules); delRules != nil {
		return delRules
	}

	return nil
}

func NewNetwork(opts ...NetworkOption) (Network, error) {
	config := &NetworkConfig{}
	for _, opt := range opts {
		if err := opt(config); err != nil {
			return nil, err
		}
	}

	network := &networkLinux{
		config: config,
	}

	if err := config.validate(); err != nil {
		return nil, err
	}

	// Create namespace
	nsFd, nsErr := createNamespace(config.Name)
	if nsErr != nil {
		return nil, nsErr
	}
	defer unix.Close(nsFd)

	// Create veth pair
	vethAttrs := netlink.NewLinkAttrs()
	vethAttrs.Name = hostName(config.Name)
	link := &netlink.Veth{
		LinkAttrs:     vethAttrs,
		PeerName:      netName(config.Name),
		PeerNamespace: netlink.NsFd(nsFd),
	}

	if err := netlink.LinkAdd(link); err != nil {
		// Clean up namespace on veth creation failure
		deleteNamespace(config.Name)
		return nil, err
	}

	// Configure host side of veth pair
	if err := config.LinkManager.SetIP(hostName(config.Name), config.GatewayIp, config.Subnet.Mask); err != nil {
		network.Destroy()
		return nil, err
	}

	if err := config.LinkManager.BringUp(hostName(config.Name)); err != nil {
		network.Destroy()
		return nil, err
	}

	// Configure namespace side of veth pair
	if err := network.Execute(func() error {
		cidr := &net.IPNet{
			IP:   config.BridgeIp,
			Mask: config.Subnet.Mask,
		}
		if err := ifc.CreateBridgeWithManager(config.LinkManager, config.Name, cidr.String(), true); err != nil {
			return fmt.Errorf("failed to create bridge: %w", err)
		}

		if err := config.LinkManager.BringUp(netName(config.Name)); err != nil {
			return err
		}

		if err := config.LinkManager.SetMaster(netName(config.Name), config.Name); err != nil {
			return fmt.Errorf("failed to set bridge master: %w", err)
		}

		if err := SetDefaultRoute(config.Name, config.GatewayIp); err != nil {
			return fmt.Errorf("failed to set default route: %w", err)
		}

		return nil
	}); err != nil {
		network.Destroy()
		return nil, err
	}

	return network, nil
}
