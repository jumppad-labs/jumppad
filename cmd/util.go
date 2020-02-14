package cmd

import "github.com/hashicorp/go-hclog"

func createLogger() hclog.Logger {
	return hclog.New(&hclog.LoggerOptions{Level: hclog.Debug, Color: hclog.AutoColor})
}
