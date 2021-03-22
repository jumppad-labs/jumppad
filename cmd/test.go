package cmd

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/cucumber/godog"
	"github.com/cucumber/godog/colors"
	"github.com/cucumber/messages-go/v10"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/hashicorp/go-hclog"
	"github.com/shipyard-run/shipyard/pkg/clients"
	"github.com/shipyard-run/shipyard/pkg/shipyard"
	"github.com/shipyard-run/shipyard/pkg/utils"
	"github.com/spf13/cobra"
	"k8s.io/client-go/util/jsonpath"
)

var opts = &godog.Options{
	Format: "pretty",
	Output: colors.Colored(os.Stdout),
}

var envVars map[string]string
var output = bytes.NewBufferString("")

// used by script runner steps
var commandOutput = bytes.NewBufferString("")
var commandExitCode = 0

func newTestCmd(e shipyard.Engine, bp clients.Getter, hc clients.HTTP, bc clients.System, l hclog.Logger) *cobra.Command {
	var testFolder string
	var force bool
	var purge bool
	var variables []string
	var variablesFile string

	var testCmd = &cobra.Command{
		Use:                   "test [blueprint]",
		Short:                 "Run functional tests for the blueprint",
		Long:                  `Run functional tests for the blueprint, this command will start the shipyard blueprint `,
		DisableFlagsInUseLine: true,
		Args:                  cobra.ArbitraryArgs,
		RunE:                  newTestCmdFunc(e, bp, hc, bc, testFolder, &force, &purge, &variables, &variablesFile, l),
	}

	testCmd.Flags().StringVarP(&testFolder, "test-folder", "", "", "Specify the folder containing the functional tests.")
	testCmd.Flags().BoolVarP(&force, "force-update", "", false, "When set to true Shipyard will ignore cached images or files and will download all resources")
	testCmd.Flags().BoolVarP(&purge, "purge", "", false, "When set to true Shipyard will remove any cached images or blueprints")
	testCmd.Flags().StringSliceVarP(&variables, "var", "", nil, "Allows setting variables from the command line, variables are specified as a key and value, e.g --var key=value. Can be specified multiple times")
	testCmd.Flags().StringVarP(&variablesFile, "vars-file", "", "", "Load variables from a location other than *.vars files in the blueprint folder. E.g --vars-file=./file.vars")

	return testCmd
}

func newTestCmdFunc(e shipyard.Engine, bp clients.Getter, hc clients.HTTP, bc clients.System, testFolder string, force *bool, purge *bool, variables *[]string, variablesFile *string, l hclog.Logger) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		//

		tr := CucumberRunner{cmd, args, e, bp, hc, bc, testFolder, "", "", force, purge, l, *variables, *variablesFile}
		tr.start()

		return nil
	}
}

// CucumberRunner is a test runner for cucumber tests
type CucumberRunner struct {
	cmd           *cobra.Command
	args          []string
	e             shipyard.Engine
	bp            clients.Getter
	hc            clients.HTTP
	bc            clients.System
	testFolder    string
	testPath      string
	basePath      string
	force         *bool
	purge         *bool
	l             hclog.Logger
	variables     []string
	variablesFile string
}

// Initialize the functional tests
func (cr *CucumberRunner) start() {
	godog.BindFlags("godog.", flag.CommandLine, opts)
	flag.Parse()

	if len(cr.args) < 1 {
		cr.args = []string{"."}
	}

	// the tests will be in the blueprint_folder/test
	if cr.testFolder == "" {
		cr.testFolder = "test"
	}

	var err error
	cr.basePath, err = filepath.Abs(cr.args[0])
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	cr.testPath = filepath.Join(cr.basePath, cr.testFolder)

	opts.Paths = []string{cr.testPath}

	status := godog.TestSuite{
		Name:                "Blueprint test",
		ScenarioInitializer: cr.initializeSuite,
		Options:             opts,
	}.Run()

	os.Exit(status)
}

func (cr *CucumberRunner) initializeSuite(ctx *godog.ScenarioContext) {
	ctx.BeforeScenario(func(gs *godog.Scenario) {
		envVars = map[string]string{}
		commandOutput = bytes.NewBufferString("")
		commandExitCode = 0
	})

	ctx.AfterScenario(func(gs *godog.Scenario, err error) {
		dest := newDestroyCmd(cr.e.GetClients().Connector)
		dest.SetArgs([]string{})
		dest.Execute()

		if err != nil {
			fmt.Println(output.String())
		}

		// unset environment vars
		for k, v := range envVars {
			if v == "" {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, v)
			}
		}

		// do we need to pure the cache
		if *cr.purge {
			pc := newPurgeCmdFunc(cr.e.GetClients().Docker, cr.e.GetClients().ImageLog, cr.e.GetClients().Logger)
			pc(cr.cmd, cr.args)
		}
	})

	ctx.Step(`^I have a running blueprint$`, cr.iRunApply)
	ctx.Step(`^I have a running blueprint using version "([^"]*)"$`, cr.iRunApplyWithVersion)
	ctx.Step(`^I have a running blueprint at path "([^"]*)"$`, cr.iRunApplyAtPath)
	ctx.Step(`^I have a running blueprint at path "([^"]*)" using version "([^"]*)"$`, cr.iRunApplyAtPathWithVersion)
	ctx.Step(`^the following environment variables are set$`, cr.theFollowingEnvironmentVariablesAreSet)
	ctx.Step(`^the environment variable "([^"]*)" has a value "([^"]*)"$`, cr.theEnvironmentVariableKHasAValueV)
	ctx.Step(`^the following shipyard variables are set$`, cr.theFollowingShipyardVariablesAreSet)
	ctx.Step(`^the shipyard variable "([^"]*)" has a value "([^"]*)"$`, cr.theShipyardVariableKHasAValueV)
	ctx.Step(`^there should be a "([^"]*)" running called "([^"]*)"$`, cr.thereShouldBeAResourceRunningCalled)
	ctx.Step(`^the following resources should be running$`, cr.theFollowingResourcesShouldBeRunning)
	ctx.Step(`^a HTTP call to "([^"]*)" should result in status (\d+)$`, cr.aCallToShouldResultInStatus)
	ctx.Step(`^the response body should contain "([^"]*)"$`, cr.theResponseBodyShouldContain)
	ctx.Step(`^the info "([^"]*)" for the running "([^"]*)" called "([^"]*)" should equal "([^"]*)"$`, cr.theResourceInfoShouldEqual)
	ctx.Step(`^the info "([^"]*)" for the running "([^"]*)" called "([^"]*)" should contain "([^"]*)"$`, cr.theResourceInfoShouldContain)
	ctx.Step(`^the info "([^"]*)" for the running "([^"]*)" called "([^"]*)" should exist`, cr.theResourceInfoShouldExist)
	ctx.Step(`^I run the command "([^"]*)"$`, cr.whenIRunTheCommand)
	ctx.Step(`^I run the script$`, cr.whenIRunTheScript)
	ctx.Step(`^I expect the exit code to be (\d+)$`, cr.iExpectTheExitCodeToBe)
	ctx.Step(`^I expect the response to contain "([^"]*)"$`, cr.iExpectTheResponseToContain)
	ctx.Step(`^a TCP connection to "([^"]*)" should open$`, aTCPConnectionToShouldOpen)
}

func FeatureContext(s *godog.Suite) {
}

func (cr *CucumberRunner) iRunApply() error {
	return cr.iRunApplyWithVersion("")
}

func (cr *CucumberRunner) iRunApplyWithVersion(version string) error {
	return cr.iRunApplyAtPathWithVersion("", version)
}

func (cr *CucumberRunner) iRunApplyAtPath(path string) error {
	return cr.iRunApplyAtPathWithVersion(path, "")
}

func (cr *CucumberRunner) iRunApplyAtPathWithVersion(fp, version string) error {
	output = bytes.NewBufferString("")

	args := []string{}

	// if filepath is not absolute then it will be relative to args
	absPath := filepath.Join(cr.basePath, fp)

	args = []string{absPath}

	opts := &hclog.LoggerOptions{
		Color: hclog.AutoColor,
	}

	// set the log level
	opts.Level = hclog.Debug
	if lev := os.Getenv("LOG_LEVEL"); lev != "" {
		opts.Level = hclog.LevelFromString(lev)
	}

	// if the log level is not debug write it to a buffer
	if os.Getenv("LOG_LEVEL") != "debug" {
		opts.Output = output
		opts.Color = hclog.ColorOff
	}

	logger := hclog.New(opts)
	engine, vm := createEngine(logger)

	cr.e = engine
	cr.l = logger

	noOpen := true
	approve := true

	// re-use the run command
	rc := newRunCmdFunc(
		engine,
		engine.GetClients().Getter,
		engine.GetClients().HTTP,
		engine.GetClients().Browser,
		vm,
		engine.GetClients().Connector,
		&noOpen,
		cr.force,
		&version,
		&approve,
		&cr.variables,
		&cr.variablesFile,
		cr.l,
	)

	// if the log level is not debug write it to a buffer
	if os.Getenv("LOG_LEVEL") != "debug" {
		cr.cmd.SetOut(output)
		cr.cmd.SetErr(output)
	}

	err := rc(cr.cmd, args)
	if err != nil {
		fmt.Println(output.String())
	}
	return err
}

func (cr *CucumberRunner) theFollowingResourcesShouldBeRunning(arg1 *godog.Table) error {
	for i, r := range arg1.Rows {
		if i == 0 {
			if r.Cells[0].Value != "name" || r.Cells[1].Value != "type" {
				return fmt.Errorf("Tables should be formatted with a header row containing the columns 'name' and 'type'")
			}

			continue
		}

		if len(r.Cells) != 2 {
			return fmt.Errorf("Table rows should have two columns 'name' and 'type'")
		}

		rType := strings.TrimSpace(r.Cells[1].GetValue())
		rName := strings.TrimSpace(r.Cells[0].GetValue())

		if rType == "network" {
			err := cr.thereShouldBe1NetworkCalled(rName)
			if err != nil {
				return err
			}
		} else {
			err := cr.thereShouldBeAResourceRunningCalled(rType, rName)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (cr *CucumberRunner) thereShouldBeAResourceRunningCalled(resource string, name string) error {
	fqdn := utils.FQDN(name, resource)

	// a container can start immediately and then it can crash, this can cause a false positive for the test
	// wait a few seconds to ensure the state does not change
	time.Sleep(5 * time.Second)

	// we need to check this a number of times to make sure it is not just a slow starting container
	for i := 0; i < 100; i++ {
		args := filters.NewArgs()
		args.Add("name", fqdn)
		opts := types.ContainerListOptions{Filters: args, All: true}

		cl, err := cr.e.GetClients().Docker.ContainerList(context.Background(), opts)
		if err != nil {
			return err
		}

		runningCount := 0
		for _, c := range cl {
			// check to see if the container has failed
			if c.State == "exited" {
				return fmt.Errorf("container exited prematurely")
			}

			if c.State == "running" {
				runningCount++
			}
		}

		if runningCount == len(cl) {
			return nil
		}

		// wait a few seconds before trying again
		time.Sleep(2 * time.Second)
	}

	return fmt.Errorf("Expected %d %s %s", 1, resource, name)
}

func (cr *CucumberRunner) thereShouldBe1NetworkCalled(arg1 string) error {
	args := filters.NewArgs()
	args.Add("name", arg1)
	n, err := cr.e.GetClients().Docker.NetworkList(context.Background(), types.NetworkListOptions{Filters: args})

	if err != nil {
		return err
	}

	if len(n) != 1 {
		return fmt.Errorf("Expected 1 network called %s to be created", arg1)
	}

	return nil
}

var respBody = ""

// test making a HTTP call, for testing Ingress
func (cr *CucumberRunner) aCallToShouldResultInStatus(arg1 string, arg2 int) error {
	// try 5 times
	var err error
	for i := 0; i < 5; i++ {
		var resp *http.Response
		var netClient = &http.Client{
			Timeout: time.Second * 10,
		}
		resp, err = netClient.Get(arg1)

		if err == nil && resp.StatusCode == arg2 {
			d, _ := ioutil.ReadAll(resp.Body)
			respBody = string(d)

			return nil
		}

		if err == nil {
			err = fmt.Errorf("Expected status code %d, got %d", arg2, resp.StatusCode)
		}

		time.Sleep(10 * time.Second)
	}

	return err
}

func (cr *CucumberRunner) theResponseBodyShouldContain(value string) error {
	if strings.HasPrefix(value, "`") && strings.HasSuffix(value, "`") {
		r, err := regexp.Compile(strings.Replace(value, "`", "", -1))
		if err != nil {
			return err
		}

		s := r.FindString(respBody)
		if s == "" {
			return fmt.Errorf("Expected value %s to be found in response %s", value, respBody)
		}
	} else {
		if !strings.Contains(respBody, value) {
			return fmt.Errorf("Expected value %s to be found in response %s", value, respBody)
		}
	}

	return nil
}

func (cr *CucumberRunner) theFollowingEnvironmentVariablesAreSet(vars *godog.Table) error {
	for i, r := range vars.Rows {
		if i == 0 {
			if r.Cells[0].Value != "key" || r.Cells[1].Value != "value" {
				return fmt.Errorf("Tables should be formatted with a header row containing the columns 'key' and 'value'")
			}

			continue
		}

		if len(r.Cells) != 2 {
			return fmt.Errorf("Table rows should have two columns 'key' and 'value'")
		}

		// set the environment variable
		cr.theEnvironmentVariableKHasAValueV(r.Cells[0].GetValue(), r.Cells[1].GetValue())
	}

	return nil
}

func (cr *CucumberRunner) theFollowingShipyardVariablesAreSet(vars *godog.Table) error {
	for i, r := range vars.Rows {
		if i == 0 {
			if r.Cells[0].Value != "key" || r.Cells[1].Value != "value" {
				return fmt.Errorf("Tables should be formatted with a header row containing the columns 'key' and 'value'")
			}

			continue
		}

		if len(r.Cells) != 2 {
			return fmt.Errorf("Table rows should have two columns 'key' and 'value'")
		}

		cr.variables = append(cr.variables, fmt.Sprintf("%s=%s", r.Cells[0].Value, r.Cells[1].Value))
	}

	return nil
}

func (cr *CucumberRunner) theEnvironmentVariableKHasAValueV(key, value string) error {
	// get the existing value and set it to the map so we can undo later
	envVars[key] = os.Getenv(key)
	os.Setenv(strings.TrimSpace(key), strings.TrimSpace(value))

	return nil
}

func (cr *CucumberRunner) theShipyardVariableKHasAValueV(key, value string) error {
	cr.variables = append(cr.variables, fmt.Sprintf("%s=%s", strings.TrimSpace(key), strings.TrimSpace(value)))
	return nil
}

//ctx.Step(`^the info "([^"]*)" for the running "([^"]*)" running called "([^"]*)" should equal "([^"]*)"$`, cr.theContainerInfoShouldContainer)
func (cr *CucumberRunner) theResourceInfoShouldContain(path, resource, name, value string) error {
	s, err := cr.getJSONPath(path, resource, name)
	if err != nil {
		return err
	}

	if !strings.Contains(s, value) {
		return fmt.Errorf("String %s is not found in value %s", value, s)
	}

	return nil
}

func (cr *CucumberRunner) theResourceInfoShouldEqual(path, resource, name, value string) error {
	s, err := cr.getJSONPath(path, resource, name)
	if err != nil {
		return err
	}

	if s != value {
		return fmt.Errorf("String %s is not equal to %s", value, s)
	}

	return nil
}

func (cr *CucumberRunner) theResourceInfoShouldExist(path, resource, name string) error {
	_, err := cr.getJSONPath(path, resource, name)
	if err != nil {
		return err
	}

	return nil
}

func (cr *CucumberRunner) whenIRunTheScript(arg1 *messages.PickleStepArgument_PickleDocString) error {
	// copy the script into a temp file and try to execute it
	tmpFile, err := ioutil.TempFile(utils.ShipyardTemp(), "*.sh")
	if err != nil {
		return err
	}

	// remove the file on exit
	defer func() {
		os.Remove(tmpFile.Name())
	}()

	// write the script to the temp file
	lines := strings.Split(arg1.GetContent(), "\n")

	w := bufio.NewWriter(tmpFile)
	for _, l := range lines {
		fmt.Fprintln(w, l)
	}
	w.Flush()
	tmpFile.Close()

	// set as executable
	os.Chmod(tmpFile.Name(), 0777)

	// execute and return
	return cr.executeCommand(tmpFile.Name())
}

func (cr *CucumberRunner) whenIRunTheCommand(arg1 string) error {
	if strings.HasPrefix(arg1, ".") {
		// path is relative so make absolute using the current file path as base
		arg1 = filepath.Join(cr.testFolder, arg1)
	}

	return cr.executeCommand(arg1)
}

func (cr *CucumberRunner) iExpectTheExitCodeToBe(arg1 int) error {
	if commandExitCode != arg1 {
		return fmt.Errorf("Expected exit code to be %d, got %d\nOutput:\n%s", arg1, commandExitCode, commandOutput.String())
	}

	return nil
}

func (cr *CucumberRunner) iExpectTheResponseToContain(arg1 string) error {
	if strings.HasPrefix(arg1, "`") && strings.HasSuffix(arg1, "`") {
		r, err := regexp.Compile(strings.Replace(arg1, "`", "", -1))
		if err != nil {
			return err
		}

		s := r.FindString(commandOutput.String())
		if s != "" {
			return nil
		}
	} else {
		if strings.Contains(commandOutput.String(), arg1) {
			return nil
		}
	}

	return fmt.Errorf("Expected command output to contain %s.\n Output:\n%s", arg1, commandOutput.String())
}

func aTCPConnectionToShouldOpen(addr string) error {

	var err error
	for i := 0; i < 5; i++ {
		var c net.Conn
		c, err = net.Dial("tcp", addr)
		if err == nil {
			c.Close()
			return nil
		}

		time.Sleep(10 * time.Second)
	}

	return err
}

func (cr *CucumberRunner) executeCommand(cmd string) error {
	// split command and args
	parts := strings.Split(cmd, " ")

	commandOutput = bytes.NewBufferString("")
	commandExitCode = 0

	var c *exec.Cmd
	if len(parts) > 1 {
		c = exec.Command(parts[0], parts[1:]...)
	} else {
		c = exec.Command(parts[0])
	}

	c.Stdout = commandOutput
	c.Stderr = commandOutput

	// Ensure command does not run forever

	c.Args = parts

	errChan := make(chan error)
	doneChan := make(chan struct{})

	// Run command in background
	go func() {
		err := c.Run()
		if err != nil {
			errChan <- err
			return
		}

		doneChan <- struct{}{}
	}()

	// Block until done or error
	select {
	case err := <-errChan:
		if exiterr, ok := err.(*exec.ExitError); ok {
			if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
				commandExitCode = status.ExitStatus()
				return nil
			}
		}

		commandExitCode = -1
		return err
	case <-time.After(60 * time.Second):
		fmt.Println("timed out")
	case <-doneChan:
		return nil
	}

	return nil
}

func (cr *CucumberRunner) getJSONPath(path, resource, name string) (string, error) {
	fqdn := utils.FQDN(name, resource)
	ci, err := cr.e.GetClients().Docker.ContainerInspect(context.Background(), fqdn)
	if err != nil {
		return "", err
	}

	// flatten
	flat, _ := json.Marshal(ci)
	var flatInt interface{}
	json.Unmarshal(flat, &flatInt)

	jp := jsonpath.New("test")
	err = jp.Parse(path)
	if err != nil {
		return "", fmt.Errorf("Unable to parse JSONPath: %s", err)
	}

	buf := new(bytes.Buffer)
	err = jp.Execute(buf, flatInt)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}
