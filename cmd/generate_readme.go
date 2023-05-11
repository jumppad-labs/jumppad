package cmd

import (
	"github.com/jumppad-labs/jumppad/pkg/config/resources"
	"github.com/jumppad-labs/jumppad/pkg/shipyard"
	"github.com/shipyard-run/hclconfig/types"
	"github.com/spf13/cobra"
)

func newGenerateReadmeCommand(e shipyard.Engine) *cobra.Command {
	connectorRunCmd := &cobra.Command{
		Use:   "readme",
		Short: "Generate a markdown readme for the blueprints",
		Long:  `Generate a markdown readme for the blueprints`,
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {

			dst := ""
			if len(args) == 1 {
				dst = args[0]
			} else {
				dst = "./"
			}

			if dst == "." {
				dst = "./"
			}

			_, err := e.ParseConfig(dst)
			if err != nil {
				return err
			}

			// find a blueprint
			var blueprint *resources.Blueprint
			bps, _ := e.Config().FindResourcesByType(resources.TypeBlueprint)
			for _, bp := range bps {
				// pick the first blueprint in the root
				if bp.Metadata().Module == "" {
					blueprint = bp.(*resources.Blueprint)
					break
				}
			}

			if blueprint == nil {
				return nil
			}

			// print the title
			cmd.Printf("# %s\n", blueprint.Title)
			cmd.Println("")

			// print the authors
			cmd.Println("| <!-- -->    | <!-- -->    |")
			cmd.Println("| Author | %s |", blueprint.Author)
			cmd.Println("| Slug | %s |", blueprint.Slug)
			cmd.Println("")

			cmd.Println("## Description")
			cmd.Println(blueprint.Description)

			variables := []*types.Variable{}
			os, _ := e.Config().FindResourcesByType(types.TypeVariable)
			for _, o := range os {
				// only grab the root outputs
				if o.Metadata().Module == "" {
					variables = append(variables, o.(*types.Variable))
				}
			}

			if len(variables) > 0 {
				cmd.Println("## Variables")

				cmd.Println("These variables can be set to configure this blueprint")
				cmd.Println("")

				cmd.Println("| Name | Default | Description |")
				cmd.Println("| ---- | ------- | ----------- |")

				for _, v := range variables {
					cmd.Printf("| %s | %s | %s |\n", v.Name, v.Default, v.Description)
				}
				cmd.Println("")
			}

			outputs := []*types.Output{}
			os, _ = e.Config().FindResourcesByType(types.TypeOutput)
			for _, o := range os {
				// only grab the root outputs
				if o.Metadata().Module == "" {
					outputs = append(outputs, o.(*types.Output))
				}
			}

			if len(outputs) > 0 {
				cmd.Println("## Outputs")

				cmd.Println("These blueprint sets the following outputs")
				cmd.Println("")

				cmd.Println("| Name |  Description |")
				cmd.Println("| ---- |  ----------- |")

				for _, v := range outputs {
					cmd.Printf("| %s | %s |\n", v.Name, "")
				}
				cmd.Println("")
			}

			return nil
		},
	}

	return connectorRunCmd
}
