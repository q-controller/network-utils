//go:build linux

package cmd

import (
	"fmt"

	"github.com/q-controller/network-utils/src/utils/network/firewall"
	"github.com/spf13/cobra"
)

var configureBridgeCmd = &cobra.Command{
	Use:   "configure-bridge",
	Short: "configures a network bridge",
	RunE: func(cmd *cobra.Command, args []string) error {
		name, nameErr := cmd.Flags().GetString("name")
		if nameErr != nil {
			return nameErr
		}
		hostIf, hostIfErr := cmd.Flags().GetString("hostIf")
		if hostIfErr != nil {
			return hostIfErr
		}
		nftPrefix, nftPrefixErr := cmd.Flags().GetString("nftPrefix")
		if nftPrefixErr != nil {
			return nftPrefixErr
		}

		if jumpErr := firewall.AddJumpRule("FORWARD", nftPrefix+"FORWARD", "filter"); jumpErr != nil {
			return jumpErr
		}

		if jumpErr := firewall.AddJumpRule("INPUT", nftPrefix+"INPUT", "filter"); jumpErr != nil {
			return jumpErr
		}

		rules, rulesErr := firewall.NewRules(
			firewall.ForwardOutboundRule(nftPrefix+"FORWARD", "filter", hostIf, name),
			firewall.ForwardReturnTrafficRule(nftPrefix+"FORWARD", "filter", hostIf, name),
			firewall.MasqueradeRule("POSTROUTING", "nat", hostIf),
			firewall.PortRule(53, "udp", nftPrefix+"INPUT", "filter"),
			firewall.PortRule(67, "udp", nftPrefix+"INPUT", "filter"),
			firewall.PortRule(68, "udp", nftPrefix+"INPUT", "filter"),
			firewall.PortRule(53, "tcp", nftPrefix+"INPUT", "filter"),
			firewall.PortRule(67, "tcp", nftPrefix+"INPUT", "filter"),
			firewall.PortRule(68, "tcp", nftPrefix+"INPUT", "filter"),
		)

		if rulesErr != nil {
			return rulesErr
		}

		return firewall.AddRules(rules)
	},
}

func init() {
	rootCmd.AddCommand(configureBridgeCmd)

	configureBridgeCmd.Flags().StringP("name", "n", "", "Name of the bridge to configure")
	if err := configureBridgeCmd.MarkFlagRequired("name"); err != nil {
		panic(fmt.Errorf("failed to mark flag `name` as required: %w", err))
	}
	configureBridgeCmd.Flags().String("hostIf", "", "Host interface that the bridge will use")
	if err := configureBridgeCmd.MarkFlagRequired("hostIf"); err != nil {
		panic(fmt.Errorf("failed to mark flag `hostIf` as required: %w", err))
	}
	configureBridgeCmd.Flags().String("nftPrefix", "QEMU-", "Prefix for nftables rules")
}
