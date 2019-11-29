package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"testing"

	"github.com/DATA-DOG/godog"
	"github.com/DATA-DOG/godog/colors"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/shipyard-run/cli/config"
	"github.com/shipyard-run/cli/shipyard"
)

var currentClients *shipyard.Clients
var currentConfig *config.Config
var currentEngine *shipyard.Engine

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
	s.Step(`^there should be (\d+) container running called "([^"]*)"$`, thereShouldBeContainerRunningCalled)

	s.BeforeScenario(func(interface{}) {
	})

	s.AfterScenario(func(interface{}, error) {
		err := currentEngine.Destroy()
		if err != nil {
			panic(err)
		}
	})
}

func theConfig(arg1 string) error {
	var err error
	currentConfig = &config.Config{}
	err = config.ParseFolder(arg1, currentConfig)
	if err != nil {
		return err
	}

	err = config.ParseReferences(currentConfig)
	if err != nil {
		return err
	}

	// create providers
	cc, err := shipyard.GenerateClients()
	if err != nil {
		return err
	}

	currentClients = cc
	currentEngine = shipyard.New(currentConfig, cc)

	return nil
}

func iRunApply() error {
	return currentEngine.Apply()
}

func thereShouldBeContainerRunningCalled(arg1 int, arg2 string) error {
	args, _ := filters.ParseFlag("name="+arg2, filters.NewArgs())
	args, _ = filters.ParseFlag("status=running", args)

	opts := types.ContainerListOptions{Filters: args}

	cl, err := currentClients.Docker.ContainerList(context.Background(), opts)
	if err != nil {
		return err
	}

	if len(cl) != arg1 {
		return fmt.Errorf("Expected %d containers %s found %d", arg1, arg2, len(cl))
	}

	return nil
}
