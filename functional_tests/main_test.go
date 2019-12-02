package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"testing"
	"time"

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
	s.Step(`^(\d+) network called "([^"]*)"$`, thereShouldBe1NetworkCalled)

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

	// a container can start immediately and then it can crash, this can cause a false positive for the test
	// wait a few seconds to ensure the state does not change
	time.Sleep(5 * time.Second)

	// we need to check this a number of times to make sure it is not just a slow starting container
	for i := 0; i < 10; i++ {

		args, _ := filters.ParseFlag("name="+arg2, filters.NewArgs())
		args, _ = filters.ParseFlag("status=running", args)

		opts := types.ContainerListOptions{Filters: args}

		cl, err := currentClients.Docker.ContainerList(context.Background(), opts)
		if err != nil {
			return err
		}

		if len(cl) == arg1 {
			return nil
		}

		// wait a few seconds before trying again
		time.Sleep(2 * time.Second)
	}

	return fmt.Errorf("Expected %d containers %s", arg1, arg2)
}

func thereShouldBe1NetworkCalled(arg1 string) error {
	return godog.ErrPending
}
