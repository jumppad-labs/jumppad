package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"

	"github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/clients"
	"github.com/jumppad-labs/jumppad/pkg/config/resources"
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
	Timeout   int    `json:"timeout"`
	Task      string `json:"task"`
	Condition string `json:"condition"`
	WorkDir   string `json:"work_dir"`
	User      string `json:"user"`
	Target    string `json:"target"`
}

func (a *API) validation(w http.ResponseWriter, r *http.Request) {
	req := ValidationRequest{
		Action:  "check",
		WorkDir: "/",
		User:    "root",
		Timeout: 10,
	}

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		a.log.Error("could not decode validation request", "error", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(err.Error())
		return
	}

	state, err := resources.LoadState()
	if err != nil {
		a.log.Error("could not load state", "error", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(err.Error())
		return
	}

	res, err := state.FindResource(req.Task)
	if err != nil {
		a.log.Error("could not find task", "error", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(err.Error())
		return
	}

	task := res.(*resources.Task)

	var target string
	var message string
	var script string

	for _, c := range task.Conditions {
		if c.Name == req.Condition {
			fqrn, err := types.ParseFQRN(c.Target)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(err.Error())
				return
			}

			target = utils.FQDN(fqrn.Resource, fqrn.Module, fqrn.Type)

			message = c.FailureMessage

			if req.Action == "solve" {
				script = c.Solve
			} else {
				script = c.Check
			}

			break
		}
	}

	if script == "" {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(ValidationResponse{
			Task:      req.Task,
			Condition: req.Condition,
			ExitCode:  0,
		})
		return
	}

	dc, err := clients.NewDocker()
	if err != nil {
		a.log.Error("Could not create docker client", "error", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(err.Error())
		return
	}

	il := clients.NewImageFileLog(utils.ImageCacheLog())
	tz := &clients.TarGz{}
	ct := clients.NewDockerTasks(dc, il, tz, a.log)

	id, err := ct.FindContainerIDs(target)
	if err != nil || len(id) == 0 {
		a.log.Error("Could not find container for target", "target", target)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode("Could not find container for target")
		return
	}

	env := os.Environ()
	env = append(env, "TERM=xterm")

	output := bytes.NewBufferString("")

	exitCode, err := ct.ExecuteScript(id[0], script, env, req.WorkDir, req.User, "", req.Timeout, output)
	if err != nil {
		if exitCode != 1 {
			a.log.Error("exec failed", "error", err.Error(), "message", message)
			message = err.Error()
		}
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(ValidationResponse{
		Task:      req.Task,
		Condition: req.Condition,
		ExitCode:  exitCode,
		Message:   message,
	})
}
