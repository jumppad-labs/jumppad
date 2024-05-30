package cmd

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/cucumber/godog"
	"github.com/cucumber/godog/colors"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/jumppad-labs/hclconfig/resources"
	"github.com/jumppad-labs/jumppad/pkg/clients"
	"github.com/jumppad-labs/jumppad/pkg/clients/logger"
	"github.com/jumppad-labs/jumppad/pkg/config"
	"github.com/jumppad-labs/jumppad/pkg/config/resources/container"
	"github.com/jumppad-labs/jumppad/pkg/config/resources/docs"
	"github.com/jumppad-labs/jumppad/pkg/config/resources/k8s"
	"github.com/jumppad-labs/jumppad/pkg/config/resources/network"
	"github.com/jumppad-labs/jumppad/pkg/config/resources/nomad"
	"github.com/jumppad-labs/jumppad/pkg/jumppad"
	"github.com/jumppad-labs/jumppad/pkg/utils"
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

func newTestCmd() *cobra.Command {
	var testFolder string
	var force bool
	var dontDestroy bool
	var purge bool
	var variables []string
	var variablesFile string
	var tags string

	var testCmd = &cobra.Command{
		Use:                   "test [blueprint]",
		Short:                 "Run functional tests for the blueprint",
		Long:                  `Run functional tests for the blueprint, this command will start the jumppad blueprint `,
		DisableFlagsInUseLine: true,
		Args:                  cobra.ArbitraryArgs,
		RunE:                  newTestCmdFunc(testFolder, &force, &purge, &variables, &variablesFile, &tags, &dontDestroy),
	}

	testCmd.Flags().StringVarP(&testFolder, "test-folder", "", "", "Specify the folder containing the functional tests.")
	testCmd.Flags().BoolVarP(&force, "force-update", "", false, "When set to true jumppad will ignore cached images or files and will download all resources")
	testCmd.Flags().BoolVarP(&purge, "purge", "", false, "When set to true jumppad will remove any cached images or blueprints")
	testCmd.Flags().StringSliceVarP(&variables, "var", "", nil, "Allows setting variables from the command line, variables are specified as a key and value, e.g --var key=value. Can be specified multiple times")
	testCmd.Flags().StringVarP(&variablesFile, "vars-file", "", "", "Load variables from a location other than *.vars files in the blueprint folder. E.g --vars-file=./file.vars")
	testCmd.Flags().StringVarP(&tags, "tags", "", "", "Test tags to run e.g. @wip, @wip,@new, when not set all tests are run")
	testCmd.Flags().BoolVarP(&dontDestroy, "dont-destroy", "", false, "When set to true, jumppad does not destroy the blueprint after executing the tests")

	return testCmd
}

func newTestCmdFunc(
	testFolder string,
	force *bool,
	purge *bool,
	variables *[]string,
	variablesFile *string,
	tags *string,
	dontDestroy *bool,
) func(cmd *cobra.Command, args []string) error {

	return func(cmd *cobra.Command, args []string) error {
		tr := CucumberRunner{
			cmd:           cmd,
			args:          args,
			testFolder:    testFolder,
			force:         force,
			purge:         purge,
			baseVariables: *variables,
			variablesFile: *variablesFile,
			tags:          *tags,
			dontDestroy:   dontDestroy,
		}

		tr.start()

		return nil
	}
}

// CucumberRunner is a test runner for cucumber tests
type CucumberRunner struct {
	cmd           *cobra.Command
	args          []string
	e             jumppad.Engine
	cli           *clients.Clients
	cred          map[string]string
	testFolder    string
	testPath      string
	basePath      string
	force         *bool
	purge         *bool
	l             logger.Logger
	baseVariables []string
	variables     []string
	variablesFile string
	tags          string
	dontDestroy   *bool
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
	opts.Tags = cr.tags

	status := godog.TestSuite{
		Name:                "Blueprint test",
		ScenarioInitializer: cr.initializeSuite,
		Options:             opts,
	}.Run()

	os.Exit(status)
}

func (cr *CucumberRunner) initializeSuite(ctx *godog.ScenarioContext) {
	sb := &strings.Builder{}

	ctx.BeforeScenario(func(gs *godog.Scenario) {
		// ensure the variables are not carried over from a previous scenario
		envVars = map[string]string{}
		commandOutput = bytes.NewBufferString("")
		commandExitCode = 0
		cr.variables = cr.baseVariables

		cl := logger.NewLogger(sb, logger.LogLevelDebug)

		defaultRegistry := jumppad.GetDefaultRegistry()
		registryCredentials := jumppad.GetRegistryCredentials()

		cli, _ := clients.GenerateClients(cl)
		engine, err := createEngine(cl, cli, defaultRegistry, registryCredentials)
		if err != nil {
			fmt.Printf("Unable to setup tests: %s\n", err)
			return
		}

		cr.e = engine
		cr.l = cl
		cr.cli = cli
		cr.cred = registryCredentials

		// do we need to pure the cache
		if *cr.purge {
			pc := newPurgeCmdFunc(cr.cli.Docker, cr.cli.ImageLog, cr.cli.Logger)
			pc(cr.cmd, cr.args)
		}
	})

	ctx.AfterScenario(func(gs *godog.Scenario, err error) {
		if err != nil {
			fmt.Println(sb.String())
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

		// only destroy when the dont-destroy flag is false
		if *cr.dontDestroy {
			fmt.Println("Not automatically destroying resources, run the command 'jumppad destroy' manually")
			return
		}

		sb := strings.Builder{}
		l := logger.NewLogger(&sb, logger.LogLevelDebug)
		dest := newDestroyCmd(cr.cli.Connector, l)
		dest.SetArgs([]string{"--force"})

		err = dest.Execute()
		if err != nil {
			fmt.Println(sb.String())
			os.Exit(1)
		}
	})

	ctx.Step(`^I have a running blueprint$`, cr.iRunApply)
	ctx.Step(`^I have a running blueprint at path "([^"]*)"$`, cr.iRunApplyAtPath)
	ctx.Step(`^the following environment variables are set$`, cr.theFollowingEnvironmentVariablesAreSet)
	ctx.Step(`^the environment variable "([^"]*)" has a value "([^"]*)"$`, cr.theEnvironmentVariableKHasAValueV)
	ctx.Step(`^the following jumppad variables are set$`, cr.theFollowingShipyardVariablesAreSet)
	ctx.Step(`^the jumppad variable "([^"]*)" has a value "([^"]*)"$`, cr.theShipyardVariableKHasAValueV)
	ctx.Step(`^there should be a resource running called "([^"]*)"$`, cr.thereShouldBeAResourceRunningCalled)
	ctx.Step(`^the following resources should be running$`, cr.theFollowingResourcesShouldBeRunning)
	ctx.Step(`^a HTTP call to "([^"]*)" should result in status (\d+)$`, cr.aCallToShouldResultInStatus)
	ctx.Step(`^the response body should contain "([^"]*)"$`, cr.theResponseBodyShouldContain)
	ctx.Step(`^the info "([^"]*)" for the running container "([^"]*)" should equal "([^"]*)"$`, cr.theResourceInfoShouldEqual)
	ctx.Step(`^the info "([^"]*)" for the running container "([^"]*)" should contain "([^"]*)"$`, cr.theResourceInfoShouldContain)
	ctx.Step(`^the info "([^"]*)" for the running container "([^"]*)" should exist`, cr.theResourceInfoShouldExist)
	ctx.Step(`^I run the command "([^"]*)"$`, cr.whenIRunTheCommand)
	ctx.Step(`^I run the script$`, cr.whenIRunTheScript)
	ctx.Step(`^I expect the exit code to be (\d+)$`, cr.iExpectTheExitCodeToBe)
	ctx.Step(`^I expect the response to contain "([^"]*)"$`, cr.iExpectTheResponseToContain)
	ctx.Step(`^a TCP connection to "([^"]*)" should open$`, aTCPConnectionToShouldOpen)
	ctx.Step(`^the following output variables should be set$`, cr.theFollowingOutputVaraiblesShouldBeSet)
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

	// if filepath is not absolute then it will be relative to args
	absPath := filepath.Join(cr.basePath, fp)

	args := []string{absPath}

	noOpen := true
	approve := true

	// re-use the run command
	rc := newRunCmdFunc(
		cr.e,
		cr.cli.ContainerTasks,
		cr.cli.Getter,
		cr.cli.HTTP,
		cr.cli.System,
		cr.cli.Connector,
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
	} else {
		cr.l.Debug("Running test with", "variables", cr.variables)
	}

	err := rc(cr.cmd, args)
	if err != nil {
		fmt.Println(output.String())
	}

	return err
}

// Helper function that gets the name of the resource in Docker based on
// the type.
// Returns the docker container id for the main container and in the instance
// of clusters the number of nodes
func getLookupAddress(resourceName string) (string, string, int, error) {
	c, err := config.LoadState()
	if err != nil {
		return "", "", 0, fmt.Errorf("unable to load state")
	}

	res, err := c.FindResource(resourceName)
	if err != nil {
		return "", "", 0, fmt.Errorf("unable to find resource %s %s", resourceName, err)
	}

	switch res.Metadata().Type {
	case network.TypeNetwork:
		return res.Metadata().Name, res.Metadata().Type, 1, nil
	case k8s.TypeK8sCluster:
		return res.(*k8s.Cluster).ContainerName, res.Metadata().Type, 1, nil
	case nomad.TypeNomadCluster:
		cl := res.(*nomad.NomadCluster)
		return cl.ServerContainerName, res.Metadata().Type, cl.ClientNodes + 1, nil
	case container.TypeContainer:
		return res.(*container.Container).ContainerName, res.Metadata().Type, 1, nil
	case container.TypeSidecar:
		return res.(*container.Sidecar).ContainerName, res.Metadata().Type, 1, nil
	case docs.TypeDocs:
		return res.(*docs.Docs).ContainerName, res.Metadata().Type, 1, nil
	default:
		return "", "", 0, fmt.Errorf("resource type %s is not supported", res.Metadata().Type)
	}
}

func (cr *CucumberRunner) theFollowingResourcesShouldBeRunning(arg1 *godog.Table) error {
	for i, r := range arg1.Rows {
		if i == 0 {
			if len(r.Cells) != 1 || r.Cells[0].Value != "name" {
				return fmt.Errorf("tables should be formatted with a header row containing the columns 'name'")
			}

			continue
		}

		rName := strings.TrimSpace(r.Cells[0].Value)
		addr, typ, _, err := getLookupAddress(rName)
		if err != nil {
			return fmt.Errorf("unable to find resource: %s", err)
		}

		// we need some logic here to determine how best to check the running
		// resources, for example, nomad clusters may have multiple nodes
		// kubernetes clusters the name is prefixed with server

		if typ == network.TypeNetwork {
			err := cr.thereShouldBe1NetworkCalled(addr)
			if err != nil {
				return err
			}
		} else {
			err := cr.thereShouldBeAResourceRunningCalled(addr)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (cr *CucumberRunner) thereShouldBeAResourceRunningCalled(id string) error {
	// ensure that the container starts and stays running,
	// we use checkcount to test multiple times before passing
	checkCount := 0

	// we need to check this a number of times to make sure it is not just a slow starting container
	for i := 0; i < 100; i++ {
		args := filters.NewArgs()
		args.Add("name", id)
		opts := types.ContainerListOptions{Filters: args, All: true}

		cl, err := cr.cli.Docker.ContainerList(context.Background(), opts)
		if err != nil {
			return err
		}

		for _, c := range cl {
			// check to see if the container has failed
			if c.State == "exited" {
				return fmt.Errorf("container exited prematurely")
			}

			if c.State == "running" {
				checkCount++
			}
		}

		if checkCount == 5 {
			return nil
		}

		// wait a few seconds before trying again
		time.Sleep(2 * time.Second)
	}

	return fmt.Errorf("expected %d %s", 1, id)
}

func (cr *CucumberRunner) thereShouldBe1NetworkCalled(arg1 string) error {
	args := filters.NewArgs()
	args.Add("name", arg1)
	n, err := cr.cli.Docker.NetworkList(context.Background(), types.NetworkListOptions{Filters: args})

	if err != nil {
		return err
	}

	if len(n) != 1 {
		return fmt.Errorf("expected 1 network called %s to be created", arg1)
	}

	return nil
}

var respBody = ""

// test making a HTTP call, for testing Ingress
func (cr *CucumberRunner) aCallToShouldResultInStatus(arg1 string, arg2 int) error {
	// try 5 times
	var err error
	for i := 0; i < 60; i++ {
		var resp *http.Response
		var netClient = &http.Client{
			Timeout: time.Second * 10,
			Transport: &http.Transport{
				Dial: (&net.Dialer{
					Timeout: 5 * time.Second,
				}).Dial,
				TLSHandshakeTimeout: 5 * time.Second,
				// Disable cert validation
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		}

		resp, err = netClient.Get(arg1)

		if err == nil && resp.StatusCode == arg2 {
			d, _ := ioutil.ReadAll(resp.Body)
			respBody = string(d)

			return nil
		}

		if err == nil {
			err = fmt.Errorf("expected status code %d, got %d", arg2, resp.StatusCode)
		}

		time.Sleep(2 * time.Second)
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
			return fmt.Errorf("expected value %s to be found in response %s", value, respBody)
		}
	} else {
		if !strings.Contains(respBody, value) {
			return fmt.Errorf("expected value %s to be found in response %s", value, respBody)
		}
	}

	return nil
}

func (cr *CucumberRunner) theFollowingEnvironmentVariablesAreSet(vars *godog.Table) error {
	for i, r := range vars.Rows {
		if i == 0 {
			if r.Cells[0].Value != "key" || r.Cells[1].Value != "value" {
				return fmt.Errorf("tables should be formatted with a header row containing the columns 'key' and 'value'")
			}

			continue
		}

		if len(r.Cells) != 2 {
			return fmt.Errorf("table rows should have two columns 'key' and 'value'")
		}

		// set the environment variable
		cr.theEnvironmentVariableKHasAValueV(r.Cells[0].Value, r.Cells[1].Value)
	}

	return nil
}

func (cr *CucumberRunner) theFollowingShipyardVariablesAreSet(vars *godog.Table) error {
	for i, r := range vars.Rows {
		if i == 0 {
			if r.Cells[0].Value != "key" || r.Cells[1].Value != "value" {
				return fmt.Errorf("tables should be formatted with a header row containing the columns 'key' and 'value'")
			}

			continue
		}

		if len(r.Cells) != 2 {
			return fmt.Errorf("table rows should have two columns 'key' and 'value'")
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

// ctx.Step(`^the info "([^"]*)" for the running "([^"]*)" running called "([^"]*)" should equal "([^"]*)"$`, cr.theContainerInfoShouldContainer)
func (cr *CucumberRunner) theResourceInfoShouldContain(path, resource, value string) error {
	s, err := cr.getJSONPath(path, resource)
	if err != nil {
		return err
	}

	if !strings.Contains(s, value) {
		return fmt.Errorf("string %s is not found in value %s", value, s)
	}

	return nil
}

func (cr *CucumberRunner) theResourceInfoShouldEqual(path, resource, value string) error {
	s, err := cr.getJSONPath(path, resource)
	if err != nil {
		return err
	}

	if s != value {
		return fmt.Errorf("string %s is not equal to %s", value, s)
	}

	return nil
}

func (cr *CucumberRunner) theResourceInfoShouldExist(path, resource string) error {
	_, err := cr.getJSONPath(path, resource)
	if err != nil {
		return err
	}

	return nil
}

func (cr *CucumberRunner) whenIRunTheScript(arg1 *godog.DocString) error {
	// copy the script into a temp file and try to execute it
	tmpFile, err := ioutil.TempFile(utils.JumppadTemp(), "*.sh")
	if err != nil {
		return err
	}

	// remove the file on exit
	defer func() {
		os.Remove(tmpFile.Name())
	}()

	// write the script to the temp file
	lines := strings.Split(arg1.Content, "\n")

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
		return fmt.Errorf("expected exit code to be %d, got %d\nOutput:\n%s", arg1, commandExitCode, commandOutput.String())
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

	return fmt.Errorf("expected command output to contain %s.\n Output:\n%s", arg1, commandOutput.String())
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

func (cr *CucumberRunner) theFollowingOutputVaraiblesShouldBeSet(arg1 *godog.Table) error {
	c, err := config.LoadState()
	if err != nil {
		return fmt.Errorf("unable to load state")
	}

	for i, row := range arg1.Rows {
		if i == 0 {
			if len(row.Cells) != 2 || row.Cells[0].Value != "name" || row.Cells[1].Value != "value" {
				return fmt.Errorf("tables should be formatted with a header row containing the columns 'name' and value, e.g. | name | value |")
			}

			continue
		}

		// find the output
		r, _ := c.FindResource("output." + row.Cells[0].Value)
		if r == nil {
			return fmt.Errorf("expected output variable %s to be set but was nil", row.Cells[0].Value)
		}

		o := r.(*resources.Output)

		switch v := o.Value.(type) {
		case int:
			s := strconv.Itoa(int(v))
			if s != row.Cells[1].Value {
				return fmt.Errorf("output variable %s value is %s but expected %s", row.Cells[0].Value, s, row.Cells[1].Value)
			}
		case int32:
			s := strconv.Itoa(int(v))
			if s != row.Cells[1].Value {
				return fmt.Errorf("output variable %s value is %s but expected %s", row.Cells[0].Value, s, row.Cells[1].Value)
			}
		case int64:
			s := strconv.Itoa(int(v))
			if s != row.Cells[1].Value {
				return fmt.Errorf("output variable %s value is %s but expected %s", row.Cells[0].Value, s, row.Cells[1].Value)
			}
		case float32:
			s := fmt.Sprintf("%f", v)
			if s != row.Cells[1].Value {
				return fmt.Errorf("output variable %s value is %s but expected %s", row.Cells[0].Value, s, row.Cells[1].Value)
			}
		case float64:
			s := fmt.Sprintf("%f", v)
			if s != row.Cells[1].Value {
				return fmt.Errorf("output variable %s value is %s but expected %s", row.Cells[0].Value, s, row.Cells[1].Value)
			}
		case string:
			if v != row.Cells[1].Value {
				return fmt.Errorf("output variable %s value is %s but expected %s", row.Cells[0].Value, v, row.Cells[1].Value)
			}
		default:
			return fmt.Errorf("output type is %s, unable to compare", reflect.TypeOf(o.Value).String())
		}
	}

	return nil
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

func (cr *CucumberRunner) getJSONPath(path, resource string) (string, error) {
	fqdn, err := resources.ParseFQRN(resource)
	if err != nil {
		return "", fmt.Errorf("invalid resource name: %s", err)
	}

	id := utils.FQDN(fqdn.Resource, fqdn.Module, fqdn.Type)
	ci, err := cr.cli.Docker.ContainerInspect(context.Background(), id)
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
