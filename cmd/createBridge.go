//go:build linux

package cmd

import (
	"fmt"

	"github.com/q-controller/network-utils/src/utils/network/ifc"
	"github.com/spf13/cobra"
)

var createBridgeCmd = &cobra.Command{
	Use:   "create-bridge",
	Short: "Creates a network bridge",
	RunE: func(cmd *cobra.Command, args []string) error {
		name, nameErr := cmd.Flags().GetString("name")
		if nameErr != nil {
			return nameErr
		}
		cidr, cidrErr := cmd.Flags().GetString("cidr")
		if cidrErr != nil {
			return cidrErr
		}
		disableTxOffload, txErr := cmd.Flags().GetBool("disable-tx-offload")
		if txErr != nil {
			return txErr
		}

		return ifc.CreateBridge(name, cidr, disableTxOffload)
	},
}

func init() {
	rootCmd.AddCommand(createBridgeCmd)

	createBridgeCmd.Flags().StringP("name", "n", "", "Name of the bridge to create")
	if err := createBridgeCmd.MarkFlagRequired("name"); err != nil {
		panic(fmt.Errorf("failed to mark flag `name` as required: %w", err))
	}
	createBridgeCmd.Flags().String("cidr", "", "CIDR for the bridge network")
	if err := createBridgeCmd.MarkFlagRequired("cidr"); err != nil {
		panic(fmt.Errorf("failed to mark flag `cidr` as required: %w", err))
	}
	createBridgeCmd.Flags().Bool("disable-tx-offload", false, "Disable TX offload for the bridge interface")
}
