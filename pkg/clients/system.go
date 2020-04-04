package clients

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"

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
	return cmd.Run()
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
