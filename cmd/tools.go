package cmd

import (
	"github.com/spf13/cobra"
)

var cluster string

var toolsCmd = &cobra.Command{
	Use:   "tools",
	Short: "Starts the tools container and attaches it to the current stack",
	Long:  `Starts the tools container and attaches it to the current stack`,
	Args: cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		// Do Stuff Here
	},
}

func init() {
	toolsCmd.PersistentFlags().StringVarP(&cluster, "cluster", "c", "default", "the cluster to attach to")
}