package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jumppad-labs/jumppad/cmd/view"
	"github.com/jumppad-labs/jumppad/pkg/clients"
	"github.com/jumppad-labs/jumppad/pkg/config"
	"github.com/jumppad-labs/jumppad/pkg/jumppad"
	"github.com/jumppad-labs/jumppad/pkg/utils"
	"github.com/spf13/cobra"
)

func newDevCmd() *cobra.Command {
	var variables []string
	var variablesFile string
	var interval string
	var ttyFlag bool

	devCmd := &cobra.Command{
		Use:   "dev",
		Short: "Watches config for changes and automatically runs `up` when a change is detected",
		Long:  "Watches config for changes and automatically runs `up` when a change is detected",
		Example: `
		jumppad dev ./
`,
		Args:         cobra.ArbitraryArgs,
		RunE:         newDevCmdFunc(&variables, &variablesFile, &interval, &ttyFlag),
		SilenceUsage: true,
	}

	devCmd.Flags().StringSliceVarP(&variables, "var", "", nil, "Allows setting variables from the command line, variables are specified as a key and value, e.g --var key=value. Can be specified multiple times")
	devCmd.Flags().StringVarP(&variablesFile, "vars-file", "", "", "Load variables from a location other than *.vars files in the blueprint folder. E.g --vars-file=./file.vars")
	devCmd.Flags().StringVarP(&interval, "interval", "", "5s", "Interval to check for changes. E.g. --interval=5s")
	devCmd.Flags().BoolVarP(&ttyFlag, "disable-tty", "", false, "Enable/disable output to TTY")

	return devCmd
}

func newDevCmdFunc(variables *[]string, variablesFile, interval *string, ttyFlag *bool) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		// create the output view
		var v view.View
		var err error
		if *ttyFlag {
			v, err = view.NewLogView()
			if err != nil {
				return fmt.Errorf("unable to create output view: %s", err)
			}
		} else {
			v, err = view.NewTTYView()
			if err != nil {
				return fmt.Errorf("unable to create output view: %s", err)
			}
		}

		engineClients, _ := clients.GenerateClients(v.Logger())
		engine, _, err := createEngine(v.Logger(), engineClients)
		if err != nil {
			return fmt.Errorf("unable to create engine: %s", err)
		}

		// create the shipyard and sub folders in the users home directory
		utils.CreateFolders()

		d, err := time.ParseDuration(*interval)
		if err != nil {
			return fmt.Errorf("invalid duration %s, please specify a duration using go syntax, e.g. 5s, 1m", *interval)
		}

		// set the source
		src := ""
		if len(args) == 1 {
			src = args[0]
		} else {
			src = "./"
		}

		if src == "." {
			src = "./"
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
		if cb, err := engineClients.Connector.GetLocalCertBundle(utils.CertsDir("")); err != nil || cb == nil {
			// generate certs
			v.Logger().Debug("Generating TLS Certificates for Ingress", "path", utils.CertsDir(""))

			_, err := engineClients.Connector.GenerateLocalCertBundle(utils.CertsDir(""))
			if err != nil {
				return fmt.Errorf("unable to generate connector certificates: %s", err)
			}
		}

		// start the connector
		if !engineClients.Connector.IsRunning() {
			cb, err := engineClients.Connector.GetLocalCertBundle(utils.CertsDir(""))
			if err != nil {
				return fmt.Errorf("unable to get certificates to secure ingress: %s", err)
			}

			v.Logger().Debug("Starting API server")

			err = engineClients.Connector.Start(cb)
			if err != nil {
				return fmt.Errorf("unable to start API server: %s", err)
			}
		}

		// start the
		go doUpdates(v, engine, src, vars, *variablesFile, d)

		// Show the view
		err = v.Display()
		if err != nil {
			return err
		}

		return nil
	}
}

func doUpdates(v view.View, e jumppad.Engine, source string, variables map[string]string, variableFile string, interval time.Duration) {
	v.Logger().Debug("P_Init: Checking cmd-line parameters....................")
	v.Logger().Debug("V_Init: Allocate screens................................")
	v.Logger().Debug("M_LoadDefaults: Load system defaults....................")
	v.Logger().Debug("Z_Init: Init zone memory allocation daemon..............")
	v.Logger().Debug("DPMI Memory: 8xa11000, 8x800000 allocated...............")
	v.Logger().Debug("W_Init: Init WADfiles...................................")
	v.Logger().Debug("W_Init: adding doom1.wad................................")
	v.Logger().Debug("W_Init: shareware version...............................")
	v.Logger().Debug("startskill: 2 deathmatch: 0 startepisode: 1")

	// first check if the state exists
	// if not we need to do an apply then we can go into the check loop
	_, err := config.LoadState()
	if err != nil {
		v.UpdateStatus("Applying initial configuration...", false)
		_, err := e.ApplyWithVariables(context.Background(), source, variables, variableFile)
		if err != nil {
			v.Logger().Error(err.Error())
		}
	}

	v.UpdateStatus("Checking for changes...", false)
	for {
		time.Sleep(interval)

		new, changed, removed, _, err := e.Diff(source, variables, variableFile)
		if err != nil {
			v.Logger().Error(err.Error())
		}

		if len(new) > 0 || len(changed) > 0 || len(removed) > 0 {
			v.UpdateStatus(
				fmt.Sprintf(
					"Applying changes, %d resources to add, %d resources changed, %d resources to delete, running up",
					len(new),
					len(changed),
					len(removed),
				), false)

			for _, n := range changed {
				v.Logger().Debug("Changed", "resource", n.Metadata().ID)
			}

			_, err := e.ApplyWithVariables(context.Background(), source, variables, variableFile)
			if err != nil {
				v.Logger().Error(err.Error())
			}

			v.UpdateStatus("Checking for changes...", false)
		}
	}
}
