package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/cucumber/godog"
	"github.com/cucumber/godog/colors"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/hashicorp/go-hclog"
	gvm "github.com/nicholasjackson/version-manager"
	"github.com/shipyard-run/shipyard/pkg/clients"
	"github.com/shipyard-run/shipyard/pkg/shipyard"
	"github.com/shipyard-run/shipyard/pkg/utils"
	"github.com/spf13/cobra"
	"k8s.io/client-go/util/jsonpath"
)

func newTestCmd(e shipyard.Engine, bp clients.Getter, hc clients.HTTP, bc clients.System, l hclog.Logger) *cobra.Command {
	var testFolder string
	var force bool
	var purge bool
	var testCmd = &cobra.Command{
		Use:                   "test [blueprint]",
		Short:                 "Run functional tests for the blueprint",
		Long:                  `Run functional tests for the blueprint, this command will start the shipyard blueprint `,
		DisableFlagsInUseLine: true,
		Args:                  cobra.ArbitraryArgs,
		RunE:                  newTestCmdFunc(e, bp, hc, bc, testFolder, &force, &purge, l),
	}

	testCmd.Flags().StringVarP(&testFolder, "test-folder", "", "", "Specify the folder containing the functional tests.")
	testCmd.Flags().BoolVarP(&force, "force-update", "", false, "When set to true Shipyard will ignore cached images or files and will download all resources")
	testCmd.Flags().BoolVarP(&purge, "purge", "", false, "When set to true Shipyard will remove any cached images or blueprints")

	return testCmd
}

func newTestCmdFunc(e shipyard.Engine, bp clients.Getter, hc clients.HTTP, bc clients.System, testFolder string, force *bool, purge *bool, l hclog.Logger) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		//

		tr := CucumberRunner{cmd, args, e, bp, hc, bc, testFolder, force, purge, l}
		tr.start()

		return nil
	}
}

// CucumberRunner is a test runner for cucumber tests
type CucumberRunner struct {
	cmd        *cobra.Command
	args       []string
	e          shipyard.Engine
	bp         clients.Getter
	hc         clients.HTTP
	bc         clients.System
	testFolder string
	force      *bool
	purge      *bool
	l          hclog.Logger
}

var opts = &godog.Options{
	Format: "pretty",
	Output: colors.Colored(os.Stdout),
}

var envVars map[string]string

// Initialize the functional tests
func (cr *CucumberRunner) start() {
	godog.BindFlags("godog.", flag.CommandLine, opts)
	flag.Parse()

	// the tests will be in the blueprint_folder/test
	testFolder := fmt.Sprintf("%s/test", cr.args[0])

	// unless overidden by a flag
	if cr.testFolder != "" {
		testFolder = cr.testFolder
	}

	opts.Paths = []string{testFolder}

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
	})

	ctx.AfterScenario(func(gs *godog.Scenario, err error) {
		fmt.Println("")
		cr.e.Destroy("", true)

		// unset environment vars
		for k, v := range envVars {
			if v == "" {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, v)
			}
		}

		if err != nil {
			fmt.Println(writer.String())
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
	ctx.Step(`^there should be a "([^"]*)" running called "([^"]*)"$`, cr.thereShouldBeAResourceRunningCalled)
	ctx.Step(`^the following resources should be running$`, cr.theFollowingResourcesShouldBeRunning)
	ctx.Step(`^the following environment variables are set$`, cr.theFollowingEnvironmentVariablesAreSet)
	ctx.Step(`^the environment variable "([^"]*)" has a value "([^"]*)"$`, cr.theEnvironmentVariableKHasAValueV)
	ctx.Step(`^a HTTP call to "([^"]*)" should result in status (\d+)$`, cr.aCallToShouldResultInStatus)
	ctx.Step(`^the response body should contain "([^"]*)"$`, cr.theResponseBodyShouldContain)
	ctx.Step(`^the info "([^"]*)" for the running "([^"]*)" running called "([^"]*)" should equal "([^"]*)"$`, cr.theResourceInfoShouldEqual)
	ctx.Step(`^the info "([^"]*)" for the running "([^"]*)" running called "([^"]*)" should contain "([^"]*)"$`, cr.theResourceInfoShouldContain)
	ctx.Step(`^the info "([^"]*)" for the running "([^"]*)" running called "([^"]*)" should exist`, cr.theResourceInfoShouldExist)
}

var writer = bytes.NewBufferString("")

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
	writer = bytes.NewBufferString("")
	args := []string{}

	// if filepath is not absolute then it will be relative to args
	if path.IsAbs(fp) {
		args = []string{fp}
	} else {
		// is relative to args
		args = []string{path.Join(cr.args[0], fp)}
	}

	// convert the args to absolute
	args[0], _ = filepath.Abs(args[0])

	opts := &hclog.LoggerOptions{}

	// set the log level
	opts.Level = hclog.Debug
	if lev := os.Getenv("LOG_LEVEL"); lev != "" {
		opts.Level = hclog.LevelFromString(lev)
	}

	// if the log level is not debug write it to a buffer
	if os.Getenv("LOG_LEVEL") != "debug" {
		opts.Output = writer
	}

	cr.l = hclog.New(opts)
	engine, err := shipyard.New(cr.l)
	if err != nil {
		panic(err)
	}

	cr.e = engine

	o := gvm.Options{
		Organization: "shipyard-run",
		Repo:         "shipyard",
		ReleasesPath: path.Join(utils.ShipyardHome(), "releases"),
	}

	o.AssetNameFunc = func(version, goos, goarch string) string {
		// No idea why we set the release architecture for the binary like this
		if goarch == "amd64" {
			goarch = "x86_64"
		}

		// zip is used on windows as tar is not available by default
		switch goos {
		case "linux":
			return fmt.Sprintf("shipyard_%s_%s_%s.tar.gz", version, goos, goarch)
		case "darwin":
			return fmt.Sprintf("shipyard_%s_%s_%s.tar.gz", version, goos, goarch)
		case "windows":
			return fmt.Sprintf("shipyard_%s_%s_%s.zip", version, goos, goarch)
		}

		return ""
	}

	o.ExeNameFunc = func(version, goos, goarch string) string {
		if goos == "windows" {
			return "shipyard.exe"
		}

		return "shipyard"
	}

	noOpen := true
	approve := true

	vm := gvm.New(o)

	// re-use the run command
	rc := newRunCmdFunc(
		engine,
		engine.GetClients().Getter,
		engine.GetClients().HTTP,
		engine.GetClients().Browser,
		vm,
		&noOpen,
		cr.force,
		&version,
		&approve,
		cr.l,
	)

	// if the log level is not debug write it to a buffer
	if os.Getenv("LOG_LEVEL") != "debug" {
		cr.cmd.SetOut(writer)
		cr.cmd.SetErr(writer)
	}

	err = rc(cr.cmd, args)
	if err != nil {
		fmt.Println(writer.String())
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

		if len(cl) == 1 {
			// check to see if the container has failed
			if cl[0].State == "exited" {
				return fmt.Errorf("container exited prematurely")
			}

			if cl[0].State == "running" {
				return nil
			}
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
	// try 100 times
	var err error
	for i := 0; i < 200; i++ {
		var resp *http.Response
		resp, err = http.Get(arg1)

		if err == nil && resp.StatusCode == arg2 {
			d, _ := ioutil.ReadAll(resp.Body)
			respBody = string(d)

			return nil
		}

		if err == nil {
			err = fmt.Errorf("Expected status code %d, got %d", arg2, resp.StatusCode)
		}

		time.Sleep(2 * time.Second)
	}

	return err
}

func (cr *CucumberRunner) theResponseBodyShouldContain(value string) error {
	if !strings.Contains(respBody, value) {
		return fmt.Errorf("Expected value %s to be found in response %s", value, respBody)
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

func (cr *CucumberRunner) theEnvironmentVariableKHasAValueV(key, value string) error {
	// get the existing value and set it to the map so we can undo later
	envVars[key] = os.Getenv(key)
	os.Setenv(strings.TrimSpace(key), strings.TrimSpace(value))

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

//{
// "Id": "14547512994ad61edf8cdb5f7f889bbbf03ccbce8e0e339e99c24c6083354eba",
// "Created": "2020-07-10T10:35:36.5563185Z",
// "Path": "docker-entrypoint.sh",
// "Args": [
//  "consul",
//  "agent",
//  "-config-file=/config/consul.hcl"
// ],
// "State": {
//  "Status": "running",
//  "Running": true,
//  "Paused": false,
//  "Restarting": false,
//  "OOMKilled": false,
//  "Dead": false,
//  "Pid": 18697,
//  "ExitCode": 0,
//  "Error": "",
//  "StartedAt": "2020-07-10T10:35:37.0349303Z",
//  "FinishedAt": "0001-01-01T00:00:00Z"
// },
// "Image": "sha256:941109e2896d418d13924ff4c9119ba67dc00ca9e9de0e081b255cce9eeecd77",
// "ResolvConfPath": "/var/lib/docker/containers/14547512994ad61edf8cdb5f7f889bbbf03ccbce8e0e339e99c24c6083354eba/resolv.conf",
// "HostnamePath": "/var/lib/docker/containers/14547512994ad61edf8cdb5f7f889bbbf03ccbce8e0e339e99c24c6083354eba/hostname",
// "HostsPath": "/var/lib/docker/containers/14547512994ad61edf8cdb5f7f889bbbf03ccbce8e0e339e99c24c6083354eba/hosts",
// "LogPath": "/var/lib/docker/containers/14547512994ad61edf8cdb5f7f889bbbf03ccbce8e0e339e99c24c6083354eba/14547512994ad61edf8cdb5f7f889bbbf03ccbce8e0e339e99c24c6083354eba-json.log",
// "Name": "/consul.container.shipyard.run",
// "RestartCount": 0,
// "Driver": "overlay2",
// "Platform": "linux",
// "MountLabel": "",
// "ProcessLabel": "",
// "AppArmorProfile": "",
// "ExecIDs": null,
// "HostConfig": {
//  "Binds": null,
//  "ContainerIDFile": "",
//  "LogConfig": {
//   "Type": "json-file",
//   "Config": {}
//  },
//  "NetworkMode": "default",
//  "PortBindings": {
//   "8500/": [
//    {
//     "HostIp": "0.0.0.0",
//     "HostPort": "8500"
//    }
//   ],
//   "8501/": [
//    {
//     "HostIp": "0.0.0.0",
//     "HostPort": "8501"
//    }
//   ],
//   "8502/": [
//    {
//     "HostIp": "0.0.0.0",
//     "HostPort": "8502"
//    }
//   ]
//  },
//  "RestartPolicy": {
//   "Name": "",
//   "MaximumRetryCount": 0
//  },
//  "AutoRemove": false,
//  "VolumeDriver": "",
//  "VolumesFrom": null,
//  "CapAdd": null,
//  "CapDrop": null,
//  "Dns": null,
//  "DnsOptions": null,
//  "DnsSearch": null,
//  "ExtraHosts": null,
//  "GroupAdd": null,
//  "IpcMode": "shareable",
//  "Cgroup": "",
//  "Links": null,
//  "OomScoreAdj": 0,
//  "PidMode": "",
//  "Privileged": false,
//  "PublishAllPorts": false,
//  "ReadonlyRootfs": false,
//  "SecurityOpt": null,
//  "UTSMode": "",
//  "UsernsMode": "",
//  "ShmSize": 67108864,
//  "Runtime": "runc",
//  "ConsoleSize": [
//   0,
//   0
//  ],
//  "Isolation": "",
//  "CpuShares": 0,
//  "Memory": 0,
//  "NanoCpus": 0,
//  "CgroupParent": "",
//  "BlkioWeight": 0,
//  "BlkioWeightDevice": null,
//  "BlkioDeviceReadBps": null,
//  "BlkioDeviceWriteBps": null,
//  "BlkioDeviceReadIOps": null,
//  "BlkioDeviceWriteIOps": null,
//  "CpuPeriod": 0,
//  "CpuQuota": 0,
//  "CpuRealtimePeriod": 0,
//  "CpuRealtimeRuntime": 0,
//  "CpusetCpus": "",
//  "CpusetMems": "",
//  "Devices": null,
//  "DeviceCgroupRules": null,
//  "DiskQuota": 0,
//  "KernelMemory": 0,
//  "MemoryReservation": 0,
//  "MemorySwap": 0,
//  "MemorySwappiness": null,
//  "OomKillDisable": false,
//  "PidsLimit": 0,
//  "Ulimits": null,
//  "CpuCount": 0,
//  "CpuPercent": 0,
//  "IOMaximumIOps": 0,
//  "IOMaximumBandwidth": 0,
//  "Mounts": [
//   {
//    "Type": "bind",
//    "Source": "/home/nicj/go/src/github.com/shipyard-run/shipyard/examples/container/consul_config",
//    "Target": "/config"
//   }
//  ],
//  "MaskedPaths": [
//   "/proc/asound",
//   "/proc/acpi",
//   "/proc/kcore",
//   "/proc/keys",
//   "/proc/latency_stats",
//   "/proc/timer_list",
//   "/proc/timer_stats",
//   "/proc/sched_debug",
//   "/proc/scsi",
//   "/sys/firmware"
//  ],
//  "ReadonlyPaths": [
//   "/proc/bus",
//   "/proc/fs",
//   "/proc/irq",
//   "/proc/sys",
//   "/proc/sysrq-trigger"
//  ]
// },
// "GraphDriver": {
//  "Data": {
//   "LowerDir": "/var/lib/docker/overlay2/c818f66a3b4d310607e88ecbf13e5d267331f67e402e406cb66f78cecd3e283b-init/diff:/var/lib/docker/overlay2/efc131adfe8d0a04432d86397764b46c2d7f4d5ea93aebe23a22b8559607be61/diff:/var/lib/docker/overlay2/cc6bec59eb052302fb36973c43e5594bf52b5de17e827fdcbb4bc66b8d89a8e7/diff:/var/lib/docker/overlay2/c43d1f7b095067003cfd8a3c64d2cd11515b1aea56a4090d01a9a5322ebdf2d3/diff:/var/lib/docker/overlay2/a63a949cee02936319929ce35d9ab48401ad5752ffce060eca2f63fe404e2b5a/diff:/var/lib/docker/overlay2/e28b7d43b3d1a1af962616e29a62560aad2825225338598ad456d955299f4ba7/diff:/var/lib/docker/overlay2/8837bfee52fac6a26eb42154bcc0d2924227d0f3605d9bfb0a569a7c2dff8efb/diff",
//   "MergedDir": "/var/lib/docker/overlay2/c818f66a3b4d310607e88ecbf13e5d267331f67e402e406cb66f78cecd3e283b/merged",
//   "UpperDir": "/var/lib/docker/overlay2/c818f66a3b4d310607e88ecbf13e5d267331f67e402e406cb66f78cecd3e283b/diff",
//   "WorkDir": "/var/lib/docker/overlay2/c818f66a3b4d310607e88ecbf13e5d267331f67e402e406cb66f78cecd3e283b/work"
//  },
//  "Name": "overlay2"
// },
// "Mounts": [
//  {
//   "Type": "bind",
//   "Source": "/home/nicj/go/src/github.com/shipyard-run/shipyard/examples/container/consul_config",
//   "Destination": "/config",
//   "Mode": "",
//   "RW": true,
//   "Propagation": "rprivate"
//  },
//  {
//   "Type": "volume",
//   "Name": "7d5366a70596fc62107e30d40a77f8018c4ebbec7f0bff6d21e63ace1333af2d",
//   "Source": "/var/lib/docker/volumes/7d5366a70596fc62107e30d40a77f8018c4ebbec7f0bff6d21e63ace1333af2d/_data",
//   "Destination": "/consul/data",
//   "Driver": "local",
//   "Mode": "",
//   "RW": true,
//   "Propagation": ""
//  }
// ],
// "Config": {
//  "Hostname": "consul",
//  "Domainname": "",
//  "User": "",
//  "AttachStdin": true,
//  "AttachStdout": true,
//  "AttachStderr": true,
//  "ExposedPorts": {
//   "8300/tcp": {},
//   "8301/tcp": {},
//   "8301/udp": {},
//   "8302/tcp": {},
//   "8302/udp": {},
//   "8500/": {},
//   "8500/tcp": {},
//   "8501/": {},
//   "8502/": {},
//   "8600/tcp": {},
//   "8600/udp": {}
//  },
//  "Tty": false,
//  "OpenStdin": false,
//  "StdinOnce": false,
//  "Env": [
//   "abc=123",
//   "SHIPYARD_FOLDER=/home/nicj/.shipyard",
//   "HOME_FOLDER=/home/nicj",
//   "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
//   "CONSUL_VERSION=1.8.0",
//   "HASHICORP_RELEASES=https://releases.hashicorp.com"
//  ],
//  "Cmd": [
//   "consul",
//   "agent",
//   "-config-file=/config/consul.hcl"
//  ],
//  "Image": "consul:1.8.0",
//  "Volumes": {
//   "/consul/data": {}
//  },
//  "WorkingDir": "",
//  "Entrypoint": [
//   "docker-entrypoint.sh"
//  ],
//  "OnBuild": null,
//  "Labels": {}
// },
// "NetworkSettings": {
//  "Bridge": "",
//  "SandboxID": "6cf5b8c7cd560be757332eddac4b8f7bf487148fc6c47bddbe75d6f87bb6a426",
//  "HairpinMode": false,
//  "LinkLocalIPv6Address": "",
//  "LinkLocalIPv6PrefixLen": 0,
//  "Ports": {
//   "8300/tcp": null,
//   "8301/tcp": null,
//   "8301/udp": null,
//   "8302/tcp": null,
//   "8302/udp": null,
//   "8500/tcp": [
//    {
//     "HostIp": "0.0.0.0",
//     "HostPort": "8500"
//    }
//   ],
//   "8501/tcp": [
//    {
//     "HostIp": "0.0.0.0",
//     "HostPort": "8501"
//    }
//   ],
//   "8502/tcp": [
//    {
//     "HostIp": "0.0.0.0",
//     "HostPort": "8502"
//    }
//   ],
//   "8600/tcp": null,
//   "8600/udp": null
//  },
//  "SandboxKey": "/var/run/docker/netns/6cf5b8c7cd56",
//  "SecondaryIPAddresses": null,
//  "SecondaryIPv6Addresses": null,
//  "EndpointID": "",
//  "Gateway": "",
//  "GlobalIPv6Address": "",
//  "GlobalIPv6PrefixLen": 0,
//  "IPAddress": "",
//  "IPPrefixLen": 0,
//  "IPv6Gateway": "",
//  "MacAddress": "",
//  "Networks": {
//   "onprem": {
//    "IPAMConfig": {
//     "IPv4Address": "10.6.0.200"
//    },
//    "Links": null,
//    "Aliases": [
//     "14547512994a"
//    ],
//    "NetworkID": "cdd3ac998d62c3b98b4cb80eb9c1bc4b93f290bafa8ba6904a9513291d6e6670",
//    "EndpointID": "8438f72eaf0368cad4e7bb8febdfc3b56218c6c2e58eaedae888281fd68cadc4",
//    "Gateway": "10.6.0.1",
//    "IPAddress": "10.6.0.200",
//    "IPPrefixLen": 16,
//    "IPv6Gateway": "",
//    "GlobalIPv6Address": "",
//    "GlobalIPv6PrefixLen": 0,
//    "MacAddress": "02:42:0a:06:00:c8",
//    "DriverOpts": null
//   }
//  }
// }
//}
