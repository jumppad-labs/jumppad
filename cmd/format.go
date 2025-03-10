package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/jumppad-labs/jumppad/pkg/utils"
	"github.com/spf13/cobra"
)

func newFormatCmd() *cobra.Command {
	formatCmd := &cobra.Command{
		Use:   "fmt [file] | [directory]",
		Short: "fmt the configuration at the given path",
		Long:  `fmt the configuration at the given path`,
		Example: `
  # fmt configuration in .hcl files in the current folder
  jumppad fmt

  # format configuration in a specific file
  jumppad fmt my-stack/network.hcl

	# format configuration in a specific directory
  jumppad fmt ./my-stack
	`,
		Args:         cobra.ArbitraryArgs,
		RunE:         newFormatCmdFunc(),
		SilenceUsage: true,
	}

	return formatCmd
}

func newFormatCmdFunc() func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
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
			if utils.IsHCLFile(dst) {
				err := format(dst)
				if err != nil {
					return err
				}
			} else if utils.IsLocalFolder(dst) {
				err := filepath.Walk(dst, func(path string, info os.FileInfo, err error) error {
					if err != nil {
						return err
					}

					if !info.IsDir() && strings.HasSuffix(path, ".hcl") {
						err := format(path)
						if err != nil {
							return err
						}
					}

					return nil
				})
				if err != nil {
					return err
				}
			} else {
				return fmt.Errorf("error: can only format local files and directories")
			}
		}

		return nil
	}
}

func format(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	file, diags := hclwrite.ParseConfig(data, path, hcl.InitialPos)
	if diags.HasErrors() {
		return fmt.Errorf("errors: %v", diags)
	}

	err = os.WriteFile(path, file.Bytes(), 0644)
	if err != nil {
		return err
	}

	return nil
}
