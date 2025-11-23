//go:build linux
// +build linux

package firewall

func ConfigureFirewall(oldInterface, newInterface, bridgeName string) error {
	conn := getConnection()
	if err := CreateStandardFilterTable(conn); err != nil {
		return err
	}
	if err := CreateStandardNATTable(conn); err != nil {
		return err
	}
	if jumpErr := AddJumpRule("FORWARD", "QEMU-FORWARD", FilterTable); jumpErr != nil {
		return jumpErr
	}

	if jumpErr := AddJumpRule("INPUT", "QEMU-INPUT", FilterTable); jumpErr != nil {
		return jumpErr
	}

	if oldInterface != "" {
		rules, rulesErr := NewRules(
			ForwardOutboundRule("QEMU-FORWARD", FilterTable, oldInterface, bridgeName),
			ForwardReturnTrafficRule("QEMU-FORWARD", FilterTable, oldInterface, bridgeName),
			MasqueradeRule(PostRoutingChain, NATTable, oldInterface),
			PortRule(53, "udp", "QEMU-INPUT", FilterTable),
			PortRule(67, "udp", "QEMU-INPUT", FilterTable),
			PortRule(68, "udp", "QEMU-INPUT", FilterTable),
			PortRule(53, "tcp", "QEMU-INPUT", FilterTable),
			PortRule(67, "tcp", "QEMU-INPUT", FilterTable),
			PortRule(68, "tcp", "QEMU-INPUT", FilterTable),
		)
		if rulesErr != nil {
			return rulesErr
		}
		if removeRules := RemoveRules(rules); removeRules != nil {
			return removeRules
		}
	}

	if newInterface != "" {
		rules, rulesErr := NewRules(
			ForwardOutboundRule("QEMU-FORWARD", FilterTable, newInterface, bridgeName),
			ForwardReturnTrafficRule("QEMU-FORWARD", FilterTable, newInterface, bridgeName),
			MasqueradeRule(PostRoutingChain, NATTable, newInterface),
			PortRule(53, "udp", "QEMU-INPUT", FilterTable),
			PortRule(67, "udp", "QEMU-INPUT", FilterTable),
			PortRule(68, "udp", "QEMU-INPUT", FilterTable),
			PortRule(53, "tcp", "QEMU-INPUT", FilterTable),
			PortRule(67, "tcp", "QEMU-INPUT", FilterTable),
			PortRule(68, "tcp", "QEMU-INPUT", FilterTable),
		)
		if rulesErr != nil {
			return rulesErr
		}
		if removeRules := AddRules(rules); removeRules != nil {
			return removeRules
		}
	}

	return nil
}
