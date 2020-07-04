package cmd

import (
	"os"

	"github.com/hashicorp/go-hclog"
)

func createLogger() hclog.Logger {

	opts := &hclog.LoggerOptions{Color: hclog.AutoColor}

	// set the log level
	if lev := os.Getenv("LOG_LEVEL"); lev != "" {
		opts.Level = hclog.LevelFromString(lev)
	}

	return hclog.New(opts)
}
