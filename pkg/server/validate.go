package server

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/gofiber/fiber/v2"
	"github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/clients"
	"github.com/jumppad-labs/jumppad/pkg/config/resources"
	"github.com/jumppad-labs/jumppad/pkg/utils"
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

func (a *API) handleValidate(c *fiber.Ctx) error {
	validateRequest := ValidateRequest{
		WorkDir: "/",
		User:    "root",
		Timeout: 10,
	}

	err := json.Unmarshal(c.Body(), &validateRequest)
	if err != nil {
		return c.Status(500).JSON(err.Error())
	}

	checksPath := utils.GetLibraryFolder("checks", 0775)
	checksFile := filepath.Join(checksPath, "checks.json")
	content, err := os.ReadFile(checksFile)
	if err != nil {
		return c.Status(500).JSON(err.Error())
	}

	checks := []resources.Validation{}
	err = json.Unmarshal(content, &checks)
	if err != nil {
		return c.Status(500).JSON(err.Error())
	}

	var target string
	var message string
	var exitCode int

	for _, check := range checks {
		if check.ID == validateRequest.Task {
			for _, condition := range check.Conditions {
				if condition.ID == validateRequest.Condition {
					fqrn, err := types.ParseFQRN(condition.Target)
					if err != nil {
						return c.Status(500).JSON(err.Error())
					}

					target = utils.FQDN(fqrn.Resource, fqrn.Module, fqrn.Type)
					message = condition.FailureMessage
					break
				}
			}
			break
		}
	}

	checkPath := filepath.Join(checksPath, validateRequest.Task, validateRequest.Condition)
	checkDestination := filepath.Join("/tmp", validateRequest.Condition)

	dc, err := clients.NewDocker()
	if err != nil {
		return c.Status(500).JSON(err.Error())
	}

	il := clients.NewImageFileLog(utils.ImageCacheLog())
	tgz := &clients.TarGz{}

	ct := clients.NewDockerTasks(dc, il, tgz, a.log)

	id, err := ct.FindContainerIDs(target)
	if err != nil {
		a.log.Error("Could not find container for target", "target", target)
		return c.Status(500).JSON(err.Error())
	}

	err = ct.CopyFileToContainer(id[0], checkPath, "/tmp")
	if err != nil {
		a.log.Error("Could not copy file to container", "error", err.Error(), "id", id[0], "from", checkPath, "to", checkDestination)
		return c.Status(500).JSON(err.Error())
	}

	env := os.Environ()
	env = append(env, "TERM=xterm")

	args := []string{"bash", "-e", "-c", checkDestination}

	output := bytes.NewBufferString("")

	a.log.Info("Executing command", "args", args)
	exitCode, err = ct.ExecuteCommand(
		id[0],
		args,
		env,
		validateRequest.WorkDir,
		validateRequest.User,
		"",
		validateRequest.Timeout,
		output,
	)

	if err != nil {
		if exitCode != 1 {
			a.log.Error("exec failed", "error", err.Error(), "message", message)
			message = err.Error()
		}
	}

	return c.Status(200).JSON(ValidateResponse{
		Task:      validateRequest.Task,
		Condition: validateRequest.Condition,
		ExitCode:  exitCode,
		Message:   message,
	})
}
