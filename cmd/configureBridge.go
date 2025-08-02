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

		if jumpErr := firewall.AddJumpRule("FORWARD", fmt.Sprintf("%sFORWARD", nftPrefix), "filter"); jumpErr != nil {
			return jumpErr
		}

		if jumpErr := firewall.AddJumpRule("INPUT", fmt.Sprintf("%sINPUT", nftPrefix), "filter"); jumpErr != nil {
			return jumpErr
		}

		rules, rulesErr := firewall.NewRules(
			firewall.ForwardOutboundRule(fmt.Sprintf("%sFORWARD", nftPrefix), "filter", hostIf, name),
			firewall.ForwardReturnTrafficRule(fmt.Sprintf("%sFORWARD", nftPrefix), "filter", hostIf, name),
			firewall.MasqueradeRule("POSTROUTING", "nat", hostIf),
			firewall.PortRule(53, "udp", fmt.Sprintf("%sINPUT", nftPrefix), "filter"),
			firewall.PortRule(67, "udp", fmt.Sprintf("%sINPUT", nftPrefix), "filter"),
			firewall.PortRule(68, "udp", fmt.Sprintf("%sINPUT", nftPrefix), "filter"),
			firewall.PortRule(53, "tcp", fmt.Sprintf("%sINPUT", nftPrefix), "filter"),
			firewall.PortRule(67, "tcp", fmt.Sprintf("%sINPUT", nftPrefix), "filter"),
			firewall.PortRule(68, "tcp", fmt.Sprintf("%sINPUT", nftPrefix), "filter"),
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
	configureBridgeCmd.MarkFlagRequired("name")
	configureBridgeCmd.Flags().String("hostIf", "", "Host interface that the bridge will use")
	configureBridgeCmd.MarkFlagRequired("hostIf")
	configureBridgeCmd.Flags().String("nftPrefix", "QEMU-", "Prefix for nftables rules")
}
