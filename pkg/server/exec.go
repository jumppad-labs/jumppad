package server

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
)

type ExecResponse struct {
	ExitCode int    `json:"exit_code"`
	Message  string `json:"message"`
}

type ExecRequest struct {
	WorkDir string `json:"work_dir"`
	User    string `json:"user"`
	Target  string `json:"target"`
	Command string `json:"command"`
	Timeout int    `json:"timeout"`
}

var defaultFailedCode = 254

func (a *API) handleExec(c *fiber.Ctx) error {
	execRequest := ExecRequest{
		WorkDir: "/",
		User:    "root",
		Timeout: 10,
	}

	err := json.Unmarshal(c.Body(), &execRequest)
	if err != nil {
		c.Status(500).SendString(err.Error())
	}

	var cancel context.CancelFunc
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(execRequest.Timeout)*time.Second)
	defer cancel()

	var message string
	var exitCode int

	args := []string{"exec", "-t", "-w", execRequest.WorkDir, "-u", execRequest.User, execRequest.Target}
	args = append(args, strings.Split(execRequest.Command, " ")...)

	a.log.Info("Executing command", "args", args)

	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Env = append(os.Environ(), "TERM=xterm")

	output, err := cmd.CombinedOutput()
	message = string(output)

	if err != nil {
		// try to get the exit code
		if exitError, ok := err.(*exec.ExitError); ok {
			ws := exitError.Sys().(syscall.WaitStatus)
			exitCode = ws.ExitStatus()
		} else {
			a.log.Error("could not get exit code for failed program", "command", execRequest.Command)
			exitCode = defaultFailedCode
			if message == "" {
				message = err.Error()
			}
		}
	} else {
		// success, exitCode should be 0 if go is ok
		ws := cmd.ProcessState.Sys().(syscall.WaitStatus)
		exitCode = ws.ExitStatus()
	}

	resp, err := json.Marshal(&ExecResponse{
		ExitCode: exitCode,
		Message:  message,
	})
	if err != nil {
		c.Status(500).SendString(err.Error())
	}

	return c.Status(200).SendString(string(resp))
}
