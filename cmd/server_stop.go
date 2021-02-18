package cmd

import "github.com/spf13/cobra"

var connectorStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stops the connector",
	Long:  `Stops the connector used by Shipyard to expose remote and local applications`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
}
