package cmd

import (
	"github.com/q-controller/network-utils/src/utils/network/ifc"
	"github.com/spf13/cobra"
)

// createTapCmd represents the create-tap command
var createTapCmd = &cobra.Command{
	Use:   "create-tap",
	Short: "Creates a tap device",
	RunE: func(cmd *cobra.Command, args []string) error {
		name, nameErr := cmd.Flags().GetString("name")
		if nameErr != nil {
			return nameErr
		}
		bridgeName, bridgeErr := cmd.Flags().GetString("bridge")
		if bridgeErr != nil {
			return bridgeErr
		}

		return ifc.CreateTap(name, bridgeName)
	},
}

func init() {
	rootCmd.AddCommand(createTapCmd)

	createTapCmd.Flags().StringP("name", "n", "", "Name of the tap device to create")
	createTapCmd.MarkFlagRequired("name")
	createTapCmd.Flags().String("bridge", "", "Name of the bridge to attach the tap device to")
	createTapCmd.MarkFlagRequired("bridge")
}
