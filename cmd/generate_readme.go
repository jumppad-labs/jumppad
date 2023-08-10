package cmd

import (
	"github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/config/resources/blueprint"
	"github.com/jumppad-labs/jumppad/pkg/jumppad"
	"github.com/spf13/cobra"
)

func newGenerateReadmeCommand(e jumppad.Engine) *cobra.Command {
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
			var br *blueprint.Blueprint
			bps, _ := e.Config().FindResourcesByType(blueprint.TypeBlueprint)
			for _, bp := range bps {
				// pick the first blueprint in the root
				if bp.Metadata().Module == "" {
					br = bp.(*blueprint.Blueprint)
					break
				}
			}

			if br == nil {
				return nil
			}

			// print the title
			cmd.Printf("# %s\n", br.Title)
			cmd.Println("")

			// print the authors
			cmd.Println("| <!-- -->    | <!-- -->    |")
			cmd.Println("| ---- |  ----------- |")
			cmd.Printf("| Author | %s |\n", br.Author)
			cmd.Printf("| Slug | %s |\n", br.Slug)
			cmd.Println("")

			cmd.Println("## Description")
			cmd.Println(br.Description)

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

				cmd.Println("| Name |  Description |")
				cmd.Println("| ---- |  ----------- |")

				for _, v := range variables {
					cmd.Printf("| %s | %s |\n", v.Name, v.Description)
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
					cmd.Printf("| %s | %s |\n", v.Name, v.Description)
				}
				cmd.Println("")
			}

			return nil
		},
	}

	return connectorRunCmd
}
