package server

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
)

type ValidateResponse struct {
	Task      string `json:"task"`
	Condition string `json:"condition"`
	ExitCode  int    `json:"exit_code"`
	Message   string `json:"message,omitempty"`
}

type ValidateRequest struct {
	Timeout   int    `json:"timeout"`
	Task      string `json:"task"`
	Condition string `json:"condition"`
	WorkDir   string `json:"work_dir"`
	User      string `json:"user"`
	Target    string `json:"target"`
}

var defaultFailedCode = 254

func (a *API) handleValidate(c *fiber.Ctx) error {
	validateRequest := ValidateRequest{
		WorkDir: "/",
		User:    "root",
		Timeout: 10,
	}

	err := json.Unmarshal(c.Body(), &validateRequest)
	if err != nil {
		c.Status(500).JSON(err.Error())
	}

	var cancel context.CancelFunc
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(validateRequest.Timeout)*time.Second)
	defer cancel()

	var message string
	var exitCode int

	check := path.Join("/var/lib/jumppad", validateRequest.Task, validateRequest.Condition)

	args := []string{"exec", "-t", "-w", validateRequest.WorkDir, "-u", validateRequest.User, validateRequest.Target}
	args = append(args, "bash", "-e", "-c", check)

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
			a.log.Error("could not get exit code for failed program", "check", check)
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

	return c.Status(200).JSON(ValidateResponse{
		Task:      validateRequest.Task,
		Condition: validateRequest.Condition,
		ExitCode:  exitCode,
		Message:   message,
	})
}
