package exec

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/jumppad-labs/hclconfig/convert"
	htypes "github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/clients"
	cmdClient "github.com/jumppad-labs/jumppad/pkg/clients/command"
	cmdTypes "github.com/jumppad-labs/jumppad/pkg/clients/command/types"
	contClient "github.com/jumppad-labs/jumppad/pkg/clients/container"
	"github.com/jumppad-labs/jumppad/pkg/clients/container/types"
	"github.com/jumppad-labs/jumppad/pkg/clients/logger"
	"github.com/jumppad-labs/jumppad/pkg/utils"
	sdk "github.com/jumppad-labs/plugin-sdk"
	"github.com/zclconf/go-cty/cty"
)

// checks Provider implements the sdk.Provider interface
var _ sdk.Provider = &Provider{}

// ExecRemote provider allows the execution of arbitrary commands on an existing target or
// can create a new container before running
type Provider struct {
	config    *Exec
	container contClient.ContainerTasks
	command   cmdClient.Command
	log       logger.Logger
}

// Intit creates a new Exec provider
func (p *Provider) Init(cfg htypes.Resource, l sdk.Logger) error {
	c, ok := cfg.(*Exec)
	if !ok {
		return fmt.Errorf("unable to initialize provider, resource is not of type Exec")
	}

	cli, err := clients.GenerateClients(l)
	if err != nil {
		return err
	}

	p.config = c
	p.command = cli.Command
	p.container = cli.ContainerTasks
	p.log = l

	return nil
}

func (p *Provider) Create(ctx context.Context) error {
	if ctx.Err() != nil {
		p.log.Debug("Skipping create, context cancelled", "ref", p.config.Meta.ID)
		return nil
	}

	p.log.Info("Executing script", "ref", p.config.Meta.ID, "script", p.config.Script)

	outPath := fmt.Sprintf("%s/%s.out", utils.JumppadTemp(), p.config.Meta.ID)

	if _, err := os.Stat(outPath); err != nil {
		err := os.WriteFile(outPath, []byte{}, 0755)
		if err != nil {
			return fmt.Errorf("unable to create output file: %w", err)
		}
	}

	// cleanup the local output file
	defer os.Remove(outPath)

	// check if we have a target or image specified
	if p.config.Image != nil || p.config.Target != nil {
		// remote exec
		err := p.createRemoteExec(outPath)
		if err != nil {
			return fmt.Errorf("unable to create remote exec: %w", err)
		}
	} else {
		// local exec
		pid, err := p.createLocalExec(outPath)
		if err != nil {
			return fmt.Errorf("unable to create local exec: %w", err)
		}

		p.config.PID = pid
	}

	err := p.generateOutput()
	if err != nil {
		return fmt.Errorf("unable to generate output: %w", err)
	}

	return nil
}

func (p *Provider) Destroy(ctx context.Context, force bool) error {
	if ctx.Err() != nil {
		p.log.Debug("Skipping destroy, context cancelled", "ref", p.config.Meta.ID)
		return nil
	}

	// check that we don't we have a target or image specified as
	// remote execs are not daemonized
	if p.config.Daemon && p.config.Image == nil && p.config.Target == nil {
		if p.config.PID < 1 {
			p.log.Warn("unable to stop local process, no pid")
			return nil
		}

		err := p.command.Kill(p.config.PID)
		if err != nil {
			p.log.Warn("error cleaning up daemonized process", "error", err)
		}
	}

	return nil
}

func (p *Provider) Lookup() ([]string, error) {
	return []string{}, nil
}

func (p *Provider) Refresh(ctx context.Context) error {
	if ctx.Err() != nil {
		p.log.Debug("Skipping refresh, context cancelled", "ref", p.config.Meta.ID)
		return nil
	}

	changed, err := p.Changed()
	if err != nil {
		return err
	}

	if changed {
		p.log.Debug("Refresh Exec", "ref", p.config.Meta.Name)
		return p.Create(ctx)
	}

	return nil
}

func (p *Provider) Changed() (bool, error) {
	p.log.Debug("Checking changes", "ref", p.config.Meta.ID)

	cs, err := utils.ChecksumFromInterface(p.config.Script)
	if err != nil {
		return false, fmt.Errorf("unable to generate checksum for script: %s", err)
	}

	if cs != p.config.Checksum {
		p.log.Debug("Script has changed", "ref", p.config.Meta.ID)
		return true, nil
	}

	return false, nil
}

func (p *Provider) createRemoteExec(outputPath string) error {
	// execution target id
	targetID := ""
	if p.config.Target == nil {
		// Not using existing target create new container
		id, err := p.createRemoteExecContainer()
		if err != nil {
			return fmt.Errorf("unable to create container for exec.%s: %w", p.config.Meta.Name, err)
		}

		targetID = id
	} else {
		ids, err := p.container.FindContainerIDs(p.config.Target.ContainerName)
		if err != nil {
			return fmt.Errorf("unable to find exec target: %w", err)
		}

		if len(ids) != 1 {
			return fmt.Errorf("unable to find exec target %s", p.config.Target.ContainerName)
		}

		targetID = ids[0]
	}

	// execute the script in the container
	script := p.config.Script

	containerOut := "/tmp/exec.out"

	// build the environment variables
	envs := []string{"EXEC_OUTPUT=" + containerOut}

	for k, v := range p.config.Environment {
		envs = append(envs, fmt.Sprintf("%s=%s", k, v))
	}

	user := ""
	group := ""

	if p.config.RunAs != nil {
		user = p.config.RunAs.User
		group = p.config.RunAs.Group
	}

	if p.config.Timeout == "" {
		p.config.Timeout = "300s"
	}

	timeout, err := time.ParseDuration(p.config.Timeout)
	if err != nil {
		p.log.Error("Unable to parse timeout duration", "ref", p.config.Meta.Name, "timeout", p.config.Timeout, "error", err)
		return fmt.Errorf("unable to parse timeout duration: %w", err)
	}

	_, err = p.container.ExecuteScript(targetID, script, envs, p.config.WorkingDirectory, user, group, int(timeout.Seconds()), p.log.StandardWriter())
	if err != nil {
		p.log.Error("Unable to execute command", "ref", p.config.Meta.Name, "image", p.config.Image, "script", p.config.Script)
		return fmt.Errorf("unable to execute command: in remote container: %w", err)
	}

	// copy the output file
	err = p.container.CopyFromContainer(targetID, containerOut, outputPath)
	if err != nil {
		// copy might fail as the file does not exist, only log
		p.log.Debug("Error copying output file", "ref", p.config.Meta.Name, "output", outputPath, "container", targetID)
	}
	// remove the output file
	p.container.ExecuteCommand(targetID, []string{"rm", containerOut}, nil, "", "", "", 30, p.log.StandardWriter())

	// destroy the container if we created one
	if p.config.Target == nil {
		p.container.RemoveContainer(targetID, true)
	}

	return nil
}

func (p *Provider) createRemoteExecContainer() (string, error) {
	// generate the ID for the new container based on the clock time and a string
	fqdn := utils.FQDN(p.config.Meta.Name, p.config.Meta.Module, p.config.Meta.Type)

	new := types.Container{
		Name:        fqdn,
		Image:       &types.Image{Name: p.config.Image.Name, Username: p.config.Image.Username, Password: p.config.Image.Password},
		Environment: p.config.Environment,
	}

	for _, v := range p.config.Networks {
		new.Networks = append(new.Networks, types.NetworkAttachment{
			ID:        v.ID,
			Name:      v.Name,
			IPAddress: v.IPAddress,
			Aliases:   v.Aliases,
		})
	}

	for _, v := range p.config.Volumes {
		new.Volumes = append(new.Volumes, types.Volume{
			Source:                      v.Source,
			Destination:                 v.Destination,
			Type:                        v.Type,
			ReadOnly:                    v.ReadOnly,
			BindPropagation:             v.BindPropagation,
			BindPropagationNonRecursive: v.BindPropagationNonRecursive,
			SelinuxRelabel:              v.SelinuxRelabel,
		})
	}

	new.Entrypoint = []string{}
	new.Command = []string{"/bin/sh"} // ensure container does not immediately exit

	// pull any images needed for this container
	err := p.container.PullImage(*new.Image, false)
	if err != nil {
		p.log.Error("Unable to pull container image", "ref", p.config.Meta.ID, "image", new.Image.Name)

		return "", err
	}

	id, err := p.container.CreateContainer(&new)
	if err != nil {
		p.log.Error("Unable to create container for remote exec", "ref", p.config.Meta.Name, "image", p.config.Image, "networks", p.config.Networks, "volumes", p.config.Volumes)
		return "", err
	}

	return id, err
}

func (p *Provider) createLocalExec(outputPath string) (int, error) {
	// depending on the OS, we might need to replace line endings
	// just in case the script was created on a different OS
	contents := p.config.Script
	if runtime.GOOS != "windows" {
		contents = strings.Replace(contents, "\r\n", "\n", -1)
	}

	// create a temporary file for the script
	scriptPath := filepath.Join(utils.JumppadTemp(), fmt.Sprintf("exec_%s.sh", p.config.Meta.Name))
	err := os.WriteFile(scriptPath, []byte(contents), 0755)
	if err != nil {
		return 0, fmt.Errorf("unable to write script to file: %s", err)
	}

	// build the environment variables
	envs := []string{}

	for k, v := range p.config.Environment {
		envs = append(envs, fmt.Sprintf("%s=%s", k, v))
	}

	// create the folders for logs and pids
	logPath := filepath.Join(utils.LogsDir(), fmt.Sprintf("exec_%s.log", p.config.Meta.Name))

	// do we have a duration to parse otherwise set default
	if p.config.Timeout == "" {
		p.config.Timeout = "300s"
	} else {
		if p.config.Daemon {
			p.log.Warn("Timeout will be ignored when exec is running in daemon mode")
		}
	}

	timeout, err := time.ParseDuration(p.config.Timeout)
	if err != nil {
		p.log.Error("Unable to parse timeout duration", "ref", p.config.Meta.Name, "timeout", p.config.Timeout, "error", err)
		return 1, fmt.Errorf("unable to parse timeout duration: %w", err)
	}

	// inject the output file into the environment
	envs = append(envs, fmt.Sprintf("EXEC_OUTPUT=%s", outputPath))

	// create the config
	cc := cmdTypes.CommandConfig{
		Command:          scriptPath,
		Env:              envs,
		WorkingDirectory: p.config.WorkingDirectory,
		RunInBackground:  p.config.Daemon,
		LogFilePath:      logPath,
		Timeout:          timeout,
	}

	pid, err := p.command.Execute(cc)
	if err != nil {
		return 0, err
	}

	return pid, nil
}

func (p *Provider) generateOutput() error {
	outPath := fmt.Sprintf("%s/%s.out", utils.JumppadTemp(), p.config.Meta.ID)

	// parse any output from the script
	if _, err := os.Stat(outPath); err != nil {
		p.log.Debug("Output file not found", "ref", p.config.Meta.ID, "path", outPath)
		return nil
	}

	d, err := os.ReadFile(outPath)
	if err != nil {
		return fmt.Errorf("unable to read output file: %w", err)
	}

	output := map[string]string{}
	outs := strings.Split(string(d), "\n")
	for _, v := range outs {
		parts := strings.Split(v, "=")
		if len(parts) != 2 {
			continue
		}

		output[parts[0]] = parts[1]
	}

	values := map[string]cty.Value{}
	for k, v := range output {
		value, err := convert.GoToCtyValue(v)
		if err != nil {
			return fmt.Errorf("unable to convert output value to cty: %w", err)
		}

		values[k] = value
	}

	p.config.Output = cty.ObjectVal(values)

	return nil
}
