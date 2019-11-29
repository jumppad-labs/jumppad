package main

import (
	"flag"
	"os"
	"testing"

	"github.com/DATA-DOG/godog"
	"github.com/DATA-DOG/godog/colors"
	"github.com/shipyard-run/cli/config"
)

var currentConfig *config.Config

var opt = godog.Options{
	Output: colors.Colored(os.Stdout),
	Format: "progress", // can define default values
}

func init() {
	godog.BindFlags("godog.", flag.CommandLine, &opt)
}

func TestMain(m *testing.M) {
	format := "progress"
	for _, arg := range os.Args[1:] {
		if arg == "-test.v=true" { // go test transforms -v option
			format = "pretty"
			break
		}
	}
	status := godog.RunWithOptions("godog", func(s *godog.Suite) {
		FeatureContext(s)
	}, godog.Options{
		Format: format,
		Paths:  []string{"features"},
	})

	if st := m.Run(); st > status {
		status = st
	}
	os.Exit(status)
}

func FeatureContext(s *godog.Suite) {
	s.Step(`^the config "([^"]*)"$`, theConfig)
	s.Step(`^I run apply$`, iRunApply)
	s.Step(`^there should be (\d+) container running$`, thereShouldBeContainerRunning)

	s.BeforeScenario(func(interface{}) {
	})
}

func theConfig(arg1 string) error {
	var err error
	currentConfig = &config.Config{}
	err = config.ParseFolder(arg1, currentConfig)

	return err
}

func iRunApply() error {
	return godog.ErrPending
}

func thereShouldBeContainerRunning(arg1 int) error {
	return godog.ErrPending
}
