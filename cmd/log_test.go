package cmd

import (
	"fmt"
	"os"
	"syscall"
	"testing"
	"time"
	
	"github.com/hashicorp/go-hclog"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	
	"github.com/shipyard-run/shipyard/pkg/shipyard"
)

const bluePrintDockerLogSize int64 = 170000 // Kb
const bluePrintClusterLogSize int64 = 19000 // Kb
const bluePrintListSize = 100

const UserInterruptTime = 3 * time.Second

// setupFile sets up a tmp *os.File to redirect cli logs
func mockShipyardLogCmd(t *testing.T) *os.File {
	cwd, _ := os.Getwd()
	tmpFile, err := os.CreateTemp(cwd, ".tmp.logs.")
	assert.NoError(t, err)
	assert.NotNil(t, tmpFile)
	return tmpFile
}

// runCmdWithInterrupt Tests whether output from cli utility is greater than the
// expected size
func runCmdWithInterrupt(t *testing.T, logs *cobra.Command, tmpFile *os.File, expectedSize int64){
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
	fmt.Println(stats.Size())
	assert.Greater(t, stats.Size(), expectedSize)
}
// `shipyard log kubernetes`
func testKubernetesLogs(t *testing.T, engine shipyard.Engine) {
	t.Parallel()
	
	// setup new stdio -> os.File + cli command
	kubernetesOutFile := mockShipyardLogCmd(t)
	defer func() {
		err := os.Remove(kubernetesOutFile.Name())
		assert.NoError(t, err)
	}()
	logsKubernetes := logCmd(kubernetesOutFile, engine)
	// add cli args
	logsKubernetes.SetArgs([]string{"kubernetes"})
	
	// run cli
	runCmdWithInterrupt(t, logsKubernetes, kubernetesOutFile, bluePrintClusterLogSize)
}

func testDockerLogs(t *testing.T, engine shipyard.Engine) {
	t.Parallel()
	
	// setup new stdio -> os.File + cli command
	dockerOutFile := mockShipyardLogCmd(t)
	defer func() {
		err := os.Remove(dockerOutFile.Name())
		assert.NoError(t, err)
	}()
	
	// `shipyard log containers`
	logDocker := logCmd(dockerOutFile, engine)
	logDocker.SetArgs([]string{"containers"})
	
	// run cli
	runCmdWithInterrupt(t, logDocker, dockerOutFile, bluePrintDockerLogSize)
	
}

func testLogList(t *testing.T, engine shipyard.Engine){
	t.Parallel()
	
	listFile := mockShipyardLogCmd(t)
	defer func() {
		err := os.Remove(listFile.Name())
		assert.NoError(t, err)
	}()
	
	// `shipyard log`
	listCmd := logCmd(listFile, engine)
	
	// run cli
	runCmdWithInterrupt(t, listCmd, listFile, bluePrintListSize)
}


// Requires a currently running shipyard blueprint of a Kubernetes cluster
// `shipyard run github.com/shipyard-run/shipyard/examples/single_k3s_cluster`
// `go test log_test.go log.go -v -cover`
//  not sure how to add this to mock
func TestLogCmd(t *testing.T) {
	engine, err := shipyard.New(hclog.NewNullLogger())
	assert.NoError(t, err)
	t.Run("Test `shipyard log`", func(t *testing.T) {
		testLogList(t, engine)
	})
	t.Run("Test `shipyard log containers`", func(t *testing.T) {
		testKubernetesLogs(t, engine)
	})
	t.Run("Test `shipyard log kubernetes`", func(t *testing.T) {
		testDockerLogs(t, engine)
	})
}