package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"

	"github.com/go-chi/chi"
	"github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/clients/container"
	"github.com/jumppad-labs/jumppad/pkg/clients/images"
	"github.com/jumppad-labs/jumppad/pkg/clients/tar"
	"github.com/jumppad-labs/jumppad/pkg/config"
	"github.com/jumppad-labs/jumppad/pkg/config/resources/docs"
	"github.com/jumppad-labs/jumppad/pkg/utils"
)

type ValidationResponse struct {
	Task      string `json:"task"`
	Condition string `json:"condition"`
	ExitCode  int    `json:"exit_code"`
	Message   string `json:"message,omitempty"`
}

type ValidationRequest struct {
	Action    string `json:"action"`
	Task      string `json:"task"`
	Condition string `json:"condition"`
	WorkDir   string `json:"work_dir"`
	User      string `json:"user"`
	Group     string `json:"group"`
	Target    string `json:"target"`
}

type validationRequest struct {
	ContinueOnFail bool `json:"continue_on_fail"`
}

type validationResponse struct {
	ID         string                `json:"id"`
	Conditions []validationCondition `json:"conditions"`
	Status     string                `json:"status"`
}

type validationCondition struct {
	ID       string   `json:"id"`
	Messages []string `json:"messages"`
	Status   string   `json:"status"`
}

func getTarget(id string) (string, error) {
	fqrn, err := types.ParseFQRN(id)
	if err != nil {
		return "", err
	}

	target := utils.FQDN(fqrn.Resource, fqrn.Module, fqrn.Type)
	return target, nil
}

func (a *API) executeScript(target string, script string, workdir string, user string, group string, timeout int) (int, string) {
	dc, err := container.NewDocker()
	if err != nil {
		return 254, err.Error()
	}

	il := images.NewImageFileLog(utils.ImageCacheLog())
	tz := &tar.TarGz{}
	ct, err := container.NewDockerTasks(dc, il, tz, a.log)
	if err != nil {
		return 254, err.Error()
	}

	fqdn, err := getTarget(target)
	if err != nil {
		return 254, err.Error()
	}

	id, err := ct.FindContainerIDs(fqdn)
	if err != nil || len(id) == 0 {
		return 254, err.Error()
	}

	env := os.Environ()
	env = append(env, "TERM=xterm")

	output := bytes.NewBufferString("")

	var message string
	exitCode, err := ct.ExecuteScript(id[0], script, env, workdir, user, group, timeout, output)
	if err != nil {
		if exitCode != 1 {
			a.log.Error("exec failed", "error", err.Error(), "message", message)
			message = err.Error()
		}
	}

	return exitCode, message
}

func (a *API) validation(w http.ResponseWriter, r *http.Request) {
	taskID := chi.URLParam(r, "task")
	action := chi.URLParam(r, "action")

	req := validationRequest{
		ContinueOnFail: false,
	}

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		if err != io.EOF {
			a.log.Error("could not decode validation request", "error", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(err.Error())
			return
		}
	}

	state, err := config.LoadState()
	if err != nil {
		a.log.Error("could not load state", "error", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(err.Error())
		return
	}

	res, err := state.FindResource(taskID)
	if err != nil {
		a.log.Error("could not find task", "error", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(err.Error())
		return
	}

	task := res.(*docs.Task)

	var conditions []validationCondition

	completed := 0
	status := "failed"
	for _, condition := range task.Conditions {
		var validations []docs.Validation

		switch action {
		case "check":
			validations = condition.Checks
		case "solve":
			validations = condition.Solves
		case "setup":
			validations = condition.Setups
		case "cleanup":
			validations = condition.Cleanups
		}

		status := ""
		messages := []string{}

		for _, validation := range validations {
			if validation.Script == "" {
				continue
			}

			exitCode, message := a.executeScript(validation.Target, validation.Script, validation.WorkingDirectory, validation.User, validation.Group, validation.Timeout)
			if exitCode != 0 {
				status = "failed"
				if exitCode == 1 {
					message = validation.FailureMessage
				}

				messages = append(messages, message)
			}
		}

		if status == "" {
			if action == "solve" {
				status = "skipped"
			} else {
				status = "completed"
			}

			// TODO: account for more states?

			completed++
		}

		conditions = append(conditions, validationCondition{
			ID:       condition.Name,
			Messages: messages,
			Status:   status,
		})
	}

	if len(conditions) == completed {
		if action == "solve" {
			status = "skipped"
		} else {
			status = "completed"
		}
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(validationResponse{
		ID:         task.ID,
		Conditions: conditions,
		Status:     status,
	})

	// var target string
	// var message string
	// var script string
	// var timeout int
	// var user string
	// var group string
	// var workdir string

	// for _, c := range task.Conditions {
	// 	if c.Name == req.Condition {
	// 		defaultTarget := task.Config.Target
	// 		if c.Target != "" {
	// 			defaultTarget = c.Target
	// 		}

	// 		fqrn, err := types.ParseFQRN(defaultTarget)
	// 		if err != nil {
	// 			w.WriteHeader(http.StatusInternalServerError)
	// 			json.NewEncoder(w).Encode(err.Error())
	// 			return
	// 		}

	// 		target = utils.FQDN(fqrn.Resource, fqrn.Module, fqrn.Type)

	// 		message = c.FailureMessage
	// 		user = c.User
	// 		group = c.Group
	// 		workdir = c.WorkingDirectory

	// 		if req.Action == "solve" {
	// 			script = c.Solve
	// 			timeout = c.SolveTimeout
	// 		} else {
	// 			script = c.Check
	// 			timeout = c.CheckTimeout
	// 		}

	// 		break
	// 	}
	// }

	// if script == "" {
	// 	w.WriteHeader(http.StatusOK)
	// 	json.NewEncoder(w).Encode(ValidationResponse{
	// 		Task:      req.Task,
	// 		Condition: req.Condition,
	// 		ExitCode:  0,
	// 	})
	// 	return
	// }

	// dc, err := container.NewDocker()
	// if err != nil {
	// 	a.log.Error("Could not create docker client", "error", err.Error())
	// 	w.WriteHeader(http.StatusInternalServerError)
	// 	json.NewEncoder(w).Encode(err.Error())
	// 	return
	// }

	// il := images.NewImageFileLog(utils.ImageCacheLog())
	// tz := &tar.TarGz{}
	// ct := container.NewDockerTasks(dc, il, tz, a.log)

	// id, err := ct.FindContainerIDs(target)
	// if err != nil || len(id) == 0 {
	// 	a.log.Error("Could not find container for target", "target", target)
	// 	w.WriteHeader(http.StatusInternalServerError)
	// 	json.NewEncoder(w).Encode("Could not find container for target")
	// 	return
	// }

	// env := os.Environ()
	// env = append(env, "TERM=xterm")

	// output := bytes.NewBufferString("")

	// exitCode, err := ct.ExecuteScript(id[0], script, env, workdir, user, group, timeout, output)
	// if err != nil {
	// 	if exitCode != 1 {
	// 		a.log.Error("exec failed", "error", err.Error(), "message", message)
	// 		message = err.Error()
	// 	}
	// }

	// w.WriteHeader(http.StatusOK)
	// json.NewEncoder(w).Encode(ValidationResponse{
	// 	Task:      req.Task,
	// 	Condition: req.Condition,
	// 	ExitCode:  exitCode,
	// 	Message:   message,
	// })
}
