package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/jumppad-labs/jumppad/pkg/clients"
	"github.com/jumppad-labs/jumppad/pkg/config/resources"
	"github.com/jumppad-labs/jumppad/pkg/jumppad"
	"github.com/jumppad-labs/jumppad/pkg/utils"
	"github.com/spf13/cobra"
	"golang.org/x/mod/sumdb/dirhash"
)

func newDevCmd(e jumppad.Engine, cc clients.Connector, l hclog.Logger) *cobra.Command {
	var variables []string
	var variablesFile string
	var interval string

	devCmd := &cobra.Command{
		Use:   "dev",
		Short: "Watches config for changes and automatically runs `up` when a change is detected",
		Long:  "Watches config for changes and automatically runs `up` when a change is detected",
		Example: `
		jumppad dev ./
`,
		Args:         cobra.ArbitraryArgs,
		RunE:         newDevCmdFunc(e, cc, l, &variables, &variablesFile, &interval),
		SilenceUsage: true,
	}

	devCmd.Flags().StringSliceVarP(&variables, "var", "", nil, "Allows setting variables from the command line, variables are specified as a key and value, e.g --var key=value. Can be specified multiple times")
	devCmd.Flags().StringVarP(&variablesFile, "vars-file", "", "", "Load variables from a location other than *.vars files in the blueprint folder. E.g --vars-file=./file.vars")
	devCmd.Flags().StringVarP(&interval, "interval", "", "5s", "Interval to check for changes. E.g. --interval=5s")

	return devCmd
}

func newDevCmdFunc(e jumppad.Engine, cc clients.Connector, l hclog.Logger, variables *[]string, variablesFile, interval *string) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		// create the shipyard and sub folders in the users home directory
		utils.CreateFolders()

		d, err := time.ParseDuration(*interval)
		if err != nil {
			return fmt.Errorf("invalid duration %s, please specify a duration using go syntax, e.g. 5s, 1m")
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

		// parse the vars into a map
		vars := map[string]string{}
		for _, v := range *variables {
			parts := strings.Split(v, "=")
			if len(parts) == 2 {
				vars[parts[0]] = parts[1]
			}
		}

		if variablesFile != nil && *variablesFile != "" {
			if _, err := os.Stat(*variablesFile); err != nil {
				return fmt.Errorf("variables file %s, does not exist", *variablesFile)
			}
		} else {
			vf := ""
			variablesFile = &vf
		}

		// create the certificates for the connector
		if cb, err := cc.GetLocalCertBundle(utils.CertsDir("")); err != nil || cb == nil {
			// generate certs
			l.Debug("Generating TLS Certificates for Ingress", "path", utils.CertsDir(""))
			_, err := cc.GenerateLocalCertBundle(utils.CertsDir(""))
			if err != nil {
				return fmt.Errorf("unable to generate connector certificates: %s", err)
			}
		}

		// start the connector
		if !cc.IsRunning() {
			cb, err := cc.GetLocalCertBundle(utils.CertsDir(""))
			if err != nil {
				return fmt.Errorf("unable to get certificates to secure ingress: %s", err)
			}

			l.Debug("Starting API server")

			err = cc.Start(cb)
			if err != nil {
				return fmt.Errorf("unable to start API server: %s", err)
			}
		}

		c := make(chan os.Signal)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		go func() {
			<-c
			os.Exit(0)
		}()

		for {
			cmd.Println("Checking for changes...")

			new, changed, removed, cfg, err := e.Diff(dst, vars, *variablesFile)
			if err != nil {
				cmd.PrintErr(err)
			}

			// check any containers that may be using build
			conts, _ := cfg.FindResourcesByType(resources.TypeContainer)
			for _, c := range conts {
				cont := c.(*resources.Container)

				if cont.Build != nil {
					// check the checksum
					hash, err := dirhash.HashDir(cont.Build.Context, "", dirhash.DefaultHash)
					if err != nil {
						cmd.PrintErr(err)
					}

					if hash != cont.Build.Checksum {
						changed = append(changed, c)
					}
				}
			}

			if len(new) > 0 || len(changed) > 0 || len(removed) > 0 {
				cmd.Printf("Changes detected, resources to add %d, resources changed %d, resources to delete %d, running up\n", len(new), len(changed), len(removed))

				// update status every 30s to let people know we are still running
				statusUpdate := time.NewTicker(15 * time.Second)
				startTime := time.Now()

				go func() {
					for range statusUpdate.C {
						elapsedTime := time.Since(startTime).Seconds()
						logger.Info(fmt.Sprintf("Please wait, still creating resources [Elapsed Time: %f]", elapsedTime))
					}
				}()

				_, err := e.ApplyWithVariables(dst, vars, *variablesFile)
				if err != nil {
					cmd.PrintErr(err)
				}

				// kill the timer
				statusUpdate.Stop()
			}

			cmd.Println("")
			time.Sleep(d)
		}

		return nil
	}
}
