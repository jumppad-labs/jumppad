package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/instruqt/jumppad/pkg/clients/getter"
	"github.com/instruqt/jumppad/pkg/jumppad"
	"github.com/instruqt/jumppad/pkg/utils"
	"github.com/spf13/cobra"
)

func newValidateCmd(e jumppad.Engine, bp getter.Getter) *cobra.Command {
	var variables []string
	var variablesFile string

	validateCmd := &cobra.Command{
		Use:   "validate [file] | [directory]",
		Short: "Validate the configuration at the given path",
		Long:  `Validate the configuration at the given path`,
		Example: `
  # Validate configuration from .hcl files in the current folder
  jumppad validate

  # Validate configuration from a specific file
  jumppad validate my-stack/network.hcl

  # Validate configuration from a blueprint in GitHub
  jumppad validate github.com/jumppad-labs/blueprints/kubernetes-vault
	`,
		Args:         cobra.ArbitraryArgs,
		RunE:         newValidateCmdFunc(e, bp, &variables, &variablesFile),
		SilenceUsage: true,
	}

	validateCmd.Flags().StringSliceVarP(&variables, "var", "", nil, "Allows setting variables from the command line, variables are specified as a key and value, e.g --var key=value. Can be specified multiple times")
	validateCmd.Flags().StringVarP(&variablesFile, "vars-file", "", "", "Load variables from a location other than *.vars files in the blueprint folder. E.g --vars-file=./file.vars")

	return validateCmd
}

func newValidateCmdFunc(e jumppad.Engine, bp getter.Getter, variables *[]string, variablesFile *string) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		// create the jumppad and sub folders in the users home directory
		utils.CreateFolders()

		// parse the vars into a map
		vars := map[string]string{}
		for _, v := range *variables {
			// if the variable is wrapped in single quotes remove them
			v = strings.TrimPrefix(v, "'")
			v = strings.TrimSuffix(v, "'")

			parts := strings.Split(v, "=")
			if len(parts) >= 2 {
				vars[parts[0]] = strings.Join(parts[1:], "=")
			}
		}

		// check the variables file exists
		if variablesFile != nil && *variablesFile != "" {
			if _, err := os.Stat(*variablesFile); err != nil {
				return fmt.Errorf("variables file %s, does not exist", *variablesFile)
			}
		} else {
			vf := ""
			variablesFile = &vf
		}

		dst := ""
		if len(args) == 1 {
			dst = args[0]
		} else {
			dst = "./"
		}

		if dst == "." {
			dst = "./"
		}

		if dst != "" {
			cmd.Printf("Validating configuration from '%s':\n", dst)
			if !utils.IsLocalFolder(dst) && !utils.IsHCLFile(dst) {
				// fetch the remote server from github
				bp.SetForce(true)
				err := bp.Get(dst, utils.BlueprintLocalFolder(dst))
				if err != nil {
					return fmt.Errorf("unable to retrieve blueprint: %s", err)
				}

				dst = utils.BlueprintLocalFolder(dst)
			}
		}

		_, err := e.ParseConfigWithVariables(dst, vars, *variablesFile)
		if err != nil {
			return err
		}

		cmd.Println()
		cmd.Println("Success! The configuration is valid")

		return nil
	}
}
