package cmd

import (
	"os"
	"syscall"
	"testing"
	"time"
	
	"github.com/hashicorp/go-hclog"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	
	"github.com/shipyard-run/shipyard/pkg/config"
	"github.com/shipyard-run/shipyard/pkg/shipyard"
	"github.com/shipyard-run/shipyard/pkg/utils"
)

// These tests only run successfully when a blueprint is running,
// It tests whether the streamed stack/cluster logs are greater than the specified size.
// not sure how to add this to `test_feature`

const bluePrintDockerLogSize int64 = 10000 // Kb
const bluePrintClusterLogSize int64 = 10000 // Kb
const bluePrintListSize = 100 // Kb

// signal cli exit
const UserInterruptTime = 3 * time.Second

// setupFile sets up a tmp *os.File to redirect cli logs
func mockStdOut(t *testing.T) *os.File {
	cwd, _ := os.Getwd()
	tmpFile, err := os.CreateTemp(cwd, ".tmp.logs.")
	assert.NoError(t, err)
	return tmpFile
}

// runCmdWithInterrupt Tests whether output from cli utility is greater than the
// expectedSize
func runCmdWithInterrupt(t *testing.T, logs *cobra.Command, tmpFile *os.File, expectedSize int64){
	assert.New(t)
	// user interrupt, to stop tailing logs
	go func() {
		<- time.After(UserInterruptTime)
		err := syscall.Kill(syscall.Getpid(), syscall.SIGINT)
		assert.NoError(t, err)
	}()
	// execute the cli log command, which runs until interrupt
	err := logs.Execute()
	assert.NoError(t, err)

	// not sure how else to verify whether logs worked on not
	stats, _ := os.Stat(tmpFile.Name())
	assert.NotNil(t, stats)
	err = os.Remove(tmpFile.Name())
	assert.NoError(t, err)
	
	// fmt.Println(stats.Size())
	assert.Greater(t, stats.Size(), expectedSize)
}
// `shipyard log kubernetes`
func testKubernetesLogs(t *testing.T, engine shipyard.Engine, expectedSize int64) {
	t.Parallel()
	
	// setup new stdio -> os.File + cli command
	kubernetesOutFile := mockStdOut(t)

	logsKubernetes := logCmd(kubernetesOutFile, engine)
	// add cli args
	logsKubernetes.SetArgs([]string{"kubernetes"})
	
	// run cli
	runCmdWithInterrupt(t, logsKubernetes, kubernetesOutFile, expectedSize)
}

func testDockerLogs(t *testing.T, engine shipyard.Engine, expectedSize int64) {
	t.Parallel()
	
	// setup new stdio -> os.File + cli command
	dockerOutFile := mockStdOut(t)
	
	// `shipyard log containers`
	logDocker := logCmd(dockerOutFile, engine)
	logDocker.SetArgs([]string{"containers"})
	
	// run cli
	runCmdWithInterrupt(t, logDocker, dockerOutFile, expectedSize)
	
}

func testLogList(t *testing.T, engine shipyard.Engine, expectedSize int64){
	t.Parallel()
	
	listFile := mockStdOut(t)
	
	// `shipyard log`
	listCmd := logCmd(listFile, engine)
	
	// run cli
	runCmdWithInterrupt(t, listCmd, listFile, expectedSize)
}

func checkBlueprintRunning(t *testing.T) (bool,bool) {
	if !assert.FileExists(t, utils.StatePath(),"No docker+k8 blueprint is running," +
		"`shipyard run github.com/shipyard-run/shipyard/examples/single_k3s_cluster`") {
		os.Exit(1)
	}
	c := config.New()
	err := c.FromJSON(utils.StatePath())
	assert.NoError(t, err)
	for _, r := range c.Resources {
		if r.Info().Type == "k8s_cluster"{
			assert.NotNil(t, os.Getenv("KUBECONFIG"), "KUBECONFIG not set")
			return true, false // k8,nomad
		}
	}
	return false, false
}
// make test_unit will fail here if k8s blueprint isn't running
// `shipyard run github.com/shipyard-run/shipyard/examples/single_k3s_cluster`
// `export $kubecofig..`
// `go test log_test.go log.go -v -cover`
// func TestLogCmd(t *testing.T) {
func testLogCmd(t *testing.T) {
	checkBlueprintRunning(t)
	// t.Parallel()
	engine, err := shipyard.New(hclog.NewNullLogger())
	assert.NoError(t, err)
	t.Run("Test `shipyard log`", func(t *testing.T) {
		testLogList(t, engine, bluePrintListSize)
	})
	t.Run("Test `shipyard log containers`", func(t *testing.T) {
		testKubernetesLogs(t, engine, bluePrintClusterLogSize)
	})
	t.Run("Test `shipyard log kubernetes`", func(t *testing.T) {
		testDockerLogs(t, engine, bluePrintDockerLogSize)
	})
}
