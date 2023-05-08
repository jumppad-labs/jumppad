package cmd

import "github.com/spf13/cobra"

var connectorCmd = &cobra.Command{
	Use:   "connector",
	Short: "Run the connector",
	Long:  `Runs the connector used by jumppad to expose remote and local applications`,
}
