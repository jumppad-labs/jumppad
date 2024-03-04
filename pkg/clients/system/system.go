package system

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/jumppad-labs/jumppad/pkg/clients/container"
	"github.com/jumppad-labs/jumppad/pkg/clients/container/types"
	"github.com/jumppad-labs/jumppad/pkg/clients/logger"
)

const (
	Red    = "\033[1;31m%s\033[0m"
	Green  = "\033[1;32m%s\033[0m"
	Yellow = "\033[1;33m%s\033[0m"
)

// System handles interactions between Shipyard and the OS
type System interface {
	OpenBrowser(string) error
	Preflight() DependencyStatus
	CheckVersion(string) (string, bool)
	PromptInput(in io.Reader, out io.Writer, message string) string
}

// SystemImpl is a concrete implementation of the System interface
type SystemImpl struct {
	logger logger.Logger
}

// OpenBrowser opens a URI in a new browser window
func (b *SystemImpl) OpenBrowser(uri string) error {
	openCommand := ""
	args := []string{}

	switch runtime.GOOS {
	case "linux":
		openCommand = "xdg-open"
	case "darwin":
		openCommand = "open"
	case "windows":
		openCommand = "rundll32"
		args = append(args, "url.dll,FileProtocolHandler")
	}

	args = append(args, uri)

	cmd := exec.Command(openCommand, args...)

	// we need to enable a timeout for this command as it can hang on WSL2
	doneChan := make(chan struct{})
	timerChan := time.After(10 * time.Second)

	var err error
	go func() {
		err = cmd.Run()
	}()

	select {
	case <-timerChan:
	case <-doneChan:

	}

	return err
}

type DependencyStatus struct {
	Docker bool
	Podman bool
	Git    bool
	XDG    bool

	Errors []error
}

// Preflight checks that the required software is installed and is
// working correctly
func (b *SystemImpl) Preflight() DependencyStatus {
	status := DependencyStatus{
		Docker: true,
		Podman: true,
		Git:    true,
		XDG:    true,
		Errors: []error{},
	}

	if b.checkDocker() != nil {
		status.Docker = false
	}

	if b.checkPodman() != nil {
		status.Podman = false
	}

	if !status.Docker && !status.Podman {
		status.Errors = append(status.Errors, fmt.Errorf("unable to connect to Docker or Podman, ensure Docker or Podman is installed and running"))
	}

	if b.checkGit() != nil {
		status.Git = false
		status.Errors = append(status.Errors, fmt.Errorf("unable to find 'git' command, ensure 'git' is installed"))
	}

	if runtime.GOOS == "linux" {
		if b.checkXdgOpen() != nil {
			status.XDG = false
			status.Errors = append(status.Errors, fmt.Errorf("unable to find 'xdg-open' command, ensure 'xdg-open' is installed. jumppad uses the 'xdg-open' to open browser windows"))
		}
	}

	return status
}

// CheckVersion checks the current version against the latest online version
// if an update is required the function returns a string with the upgrade text
// and a boolean value set to false.
// If no upgrade is reuquired then the boolean will be set to true and the string
// will be empty.
func (b *SystemImpl) CheckVersion(current string) (string, bool) {
	// try and get the latest version
	resp, err := http.DefaultClient.Get("https://shipyard.run/latest")
	if err != nil || resp.StatusCode != http.StatusOK {
		// if we fail just return
		return "", true
	}
	defer resp.Body.Close()

	// get the version
	d, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", true
	}

	ver := strings.TrimSpace(string(d))

	// check the version
	if current != ver {
		return fmt.Sprintf(
			fmt.Sprintf("\033[1;31m%s\033[0m", updateText),
			ver, current,
		), false
	}

	return "", true
}

// PromptInput prompts the user for input in the CLI and returns the
// entered value
func (b *SystemImpl) PromptInput(in io.Reader, out io.Writer, message string) string {
	out.Write([]byte(message))

	scanner := bufio.NewScanner(in)

	scanner.Scan()
	return scanner.Text()
}

var updateText = `
########################################################
                   JUMPPAD UPDATE
########################################################

The current version of jumppad is "%s", you have "%s".

To upgrade jumppad please use your package manager or, 
see the documentation at:
https://jumppad.dev/docs/introduction/installation for other options.
`

func (b *SystemImpl) checkDocker() error {
	d, err := container.NewDocker()
	if err != nil {
		return err
	}

	dt, err := container.NewDockerTasks(d, nil, nil, b.logger)

	if err != nil {
		return fmt.Errorf("unable to determine docker engine, please check that Docker or Podman is installed and the DOCKER_HOST is set")
	}

	// check that the server is a docker engine not podman
	// if Docker there will be a component cEngine"
	if dt.EngineInfo().EngineType != types.EngineTypeDocker {
		return fmt.Errorf("platform is not Docker")
	}

	return nil
}

func (b *SystemImpl) checkPodman() error {
	d, err := container.NewDocker()
	if err != nil {
		return err
	}

	dt, _ := container.NewDockerTasks(d, nil, nil, b.logger)

	if dt == nil {
		return fmt.Errorf("unable to determine docker engine, please check that Docker or Podman is installed and the DOCKER_HOST is set")
	}

	// check that the server is a docker engine not podman
	// if Docker there will be a component called "Engine"
	if dt.EngineInfo().EngineType != types.EngineTypePodman {
		return fmt.Errorf("platform is not Podman")
	}

	return nil
}

func (b *SystemImpl) checkGit() error {
	_, err := exec.LookPath("git")
	return err
}

func (b *SystemImpl) checkXdgOpen() error {
	_, err := exec.LookPath("xdg-open")
	return err
}
