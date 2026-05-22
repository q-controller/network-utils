package network

import (
	"errors"
	"fmt"
	"net"

	"github.com/google/nftables"
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

func getRulesForInterface(iface, hostLink string, vmSubnet *net.IPNet, masquerade bool) (*firewall.Rules, error) {
	newRules := []firewall.NewRule{
		firewall.ForwardOutboundRule("FORWARD", "filter", iface, hostLink),
		firewall.ForwardReturnTrafficRule("FORWARD", "filter", iface, hostLink),
	}
	if masquerade {
		newRules = append(newRules,
			firewall.MasqueradeRule("POSTROUTING", "nat", iface),
			firewall.ReverseMasqueradeRule("POSTROUTING", "nat", hostLink, vmSubnet),
		)
	}
	return firewall.NewRules(newRules...)
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

func (n *networkLinux) Connect(iface string, masquerade bool) error {
	// On a pristine host (e.g. fresh cloud-init VM) the nftables ruleset is
	// empty — no filter/nat tables, no FORWARD/POSTROUTING chains — and our
	// rule installation would fail with "table filter does not exist".
	// EnsureStandardFirewallInfrastructure is idempotent, so a hot host
	// where Docker/firewalld/iptables-nft has already populated the tables
	// is unaffected.
	if err := firewall.EnsureStandardFirewallInfrastructure(&nftables.Conn{}); err != nil {
		return fmt.Errorf("failed to ensure standard firewall tables: %w", err)
	}

	rules, rulesErr := getRulesForInterface(iface, hostName(n.config.Name), n.config.Subnet, masquerade)
	if rulesErr != nil {
		return rulesErr
	}
	if addedRules := firewall.AddRules(rules); addedRules != nil {
		return addedRules
	}

	return nil
}

func (n *networkLinux) Disconnect(iface string, masquerade bool) error {
	rules, rulesErr := getRulesForInterface(iface, hostName(n.config.Name), n.config.Subnet, masquerade)
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
	defer func() { _ = unix.Close(nsFd) }()

	// Create veth pair
	vethAttrs := netlink.NewLinkAttrs()
	vethAttrs.Name = hostName(config.Name)
	link := &netlink.Veth{
		LinkAttrs:     vethAttrs,
		PeerName:      netName(config.Name),
		PeerNamespace: netlink.NsFd(nsFd),
	}

	if addErr := netlink.LinkAdd(link); addErr != nil {
		if !errors.Is(addErr, unix.EEXIST) {
			// Clean up namespace on veth creation failure
			_ = deleteNamespace(config.Name)
			return nil, addErr
		}
	}

	// Configure host side of veth pair
	if err := config.LinkManager.SetIP(hostName(config.Name), config.GatewayIP, config.Subnet.Mask); err != nil {
		_ = network.Destroy()
		return nil, err
	}

	if err := config.LinkManager.BringUp(hostName(config.Name)); err != nil {
		_ = network.Destroy()
		return nil, err
	}

	// Configure namespace side of veth pair
	if err := network.Execute(func() error {
		cidr := &net.IPNet{
			IP:   config.BridgeIP,
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

		if err := SetDefaultRoute(config.Name, config.GatewayIP); err != nil {
			return fmt.Errorf("failed to set default route: %w", err)
		}

		return nil
	}); err != nil {
		_ = network.Destroy()
		return nil, err
	}

	return network, nil
}
