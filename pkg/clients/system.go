package clients

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
)

const (
	Red    = "\033[1;31m%s\033[0m"
	Green  = "\033[1;32m%s\033[0m"
	Yellow = "\033[1;33m%s\033[0m"
)

// System handles interactions between Shipyard and the OS
type System interface {
	OpenBrowser(string) error
	Preflight() (string, error)
	CheckVersion(string) (string, bool)
	PromptInput(in io.Reader, out io.Writer, message string) string
}

// SystemImpl is a concrete implementation of the System interface
type SystemImpl struct{}

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

// Preflight checks that the required software is installed and is
// working correctly
func (b *SystemImpl) Preflight() (string, error) {
	dockerPass := true
	gitPass := true
	errors := ""
	output := ""

	// check docker

	if checkDocker() != nil {
		output += fmt.Sprintf(" [ %s ] Docker\n", fmt.Sprintf(Red, " ERROR "))
		errors += "* Unable to connect to Docker, ensure Docker is installed and running.\n"
		dockerPass = false
	} else {
		output += fmt.Sprintf(" [ %s ] Docker\n", fmt.Sprintf(Green, "  OK   "))
	}

	if checkGit() != nil {
		output += fmt.Sprintf(" [ %s ] Git\n", fmt.Sprintf(Red, " ERROR "))
		errors += "* Unable to find 'git' command, ensure Git is installed. Shipyard uses the git CLI to download blueprints.\n"
		gitPass = false
	} else {
		output += fmt.Sprintf(" [ %s ] Git\n", fmt.Sprintf(Green, "  OK   "))
	}

	if runtime.GOOS == "linux" {
		if checkXdgOpen() != nil {
			output += fmt.Sprintf(" [ %s ] xdg-open\n", fmt.Sprintf(Yellow, "WARNING"))
			errors += "* Unable to find 'xdg-open' command, ensure 'xdg-open' is installed. Shipyard uses the 'xdg-open' to open browser windows.\n"
		} else {
			output += fmt.Sprintf(" [ %s ] xdg-open\n", fmt.Sprintf(Green, "  OK   "))
		}
	}

	if !dockerPass || !gitPass {
		return fmt.Sprintf("%s\n\n%s", output, errors), fmt.Errorf("Errors preflighting system")
	}

	return output, nil
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
                   SHIPYARD UPDATE
########################################################

The current version of shipyard is "%s", you have "%s".

To upgrade Shipyard please use your package manager or, 
see the documentation at:
https://shipyard.run/docs/install for other options.
`

func checkDocker() error {
	d, err := NewDocker()
	if err != nil {
		return err
	}

	_, err = d.ContainerList(context.Background(), types.ContainerListOptions{All: true})
	if err != nil {
		return err
	}

	return nil
}

func checkGit() error {
	_, err := exec.LookPath("git")
	return err
}

func checkXdgOpen() error {
	_, err := exec.LookPath("xdg-open")
	return err
}
