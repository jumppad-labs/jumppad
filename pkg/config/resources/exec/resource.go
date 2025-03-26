package exec

import (
	"fmt"
	"strings"

	"github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/config"
	ctypes "github.com/jumppad-labs/jumppad/pkg/config/resources/container"
	"github.com/jumppad-labs/jumppad/pkg/utils"
)

// TypeExec is the resource string for an Exec resource
const TypeExec string = "exec"

/*
The exec resource allows the execution of arbitrary commands and scripts. Depending on the parameters specified, the
commands are executed either on the local machine or inside of a container.

When either the `image` or `target` fields are specified, the command is executed inside of a container.
When neither of these fields are specified, the command is executed on the local machine.

```hcl

	resource "exec" "name" {
	  ...
	}

```

## Local execution

When running on the local machine, the command runs in the local user space, and has access to all the environment
variables that the user executing jumppad run has access too. Additional environment variables, and the working directory
for the command can be specified as part of the resource.

Log files for an exec running on the local machine are written to `$HOME/.jumppad/logs/exec_[name].log` and the rendered
script can be found in the jumppad temp directory `$HOME/.jumppad/tmp/exec[name].sh`.

## Remote execution

Execution can either be in a stand alone container or can target an existing and running container.
When targeting an existing container, the `target` field must be specified.
When running in a stand alone container, the `image` block must be specified.

## Setting outputs

Output variables for the exec resource can be set by echoing a key value pair to the output file inside the script.
An environment variable `${EXEC_OUTPUT}` is automatically added to the environment of the script and points to the output.

Any outputs set in the script are automatically parsed into a map and are available via the output parameter.

@include container.Image
@include container.NetworkAttachment
@include container.Volume
@include container.User

@resource

@example
```

	resource "exec" "inline" {
	  script = <<-EOF
	  #!/bin/bash
	  ls -lha

	  echo "FOO=BAR" > ${EXEC_OUTPUT}
	  EOF
	}

	output "foo" {
	  value = resource.exec.inline.output.FOO
	}

```

@example Local
```

	resource "exec" "install" {
	  script = <<-EOF
	  #!/bin/sh
	  OS=$(uname -s | tr '[:upper:]' '[:lower:]')
	  ARCH=$(uname -m | tr '[:upper:]' '[:lower:]')

	  if [ ! -f /tmp/consul ]; then
	    curl -L -o /tmp/consul.zip \
	      https://releases.hashicorp.com/consul/1.16.2/consul_1.16.2_$${OS}_$${ARCH}.zip
	    cd /tmp && unzip ./consul.zip
	  fi
	  EOF
	}

	resource "exec" "run" {
	  depends_on = ["resource.exec.install"]

	  script = <<-EOF
	  #!/bin/sh
	  /tmp/consul agent -dev
	  EOF

	  daemon = true
	}

```

@example Remote
```

	resource "container" "alpine" {
	  image {
	    name = "alpine"
	  }

	  command = ["tail", "-f", "/dev/null"]
	}

	resource "exec" "in_container" {
	  target = resource.container.alpine

	  script = <<-EOF
	  #/bin/sh
	  ls -las
	  EOF
	}

	resource "exec" "standalone" {
	  image {
	    name = "alpine"
	  }

	  script = <<-EOF
	  #/bin/sh
	  ls -las
	  EOF
	}

```
*/
type Exec struct {
	/*
	 embedded type holding name, etc

	 @ignore
	*/
	types.ResourceBase `hcl:",remain"`

	/*
		The script to execute.

		```hcl
		script = <<-EOF
		#!/bin/bash
		ls -lha
		EOF
		```

		```hcl
		script = file("script.sh")
		```

		```hcl
		script = template_file("script.sh.tpl", {
		  foo = "bar"
		})
		```
	*/
	Script string `hcl:"script" json:"script"`
	/*
		The working directory to execute the script in.

		```hcl
		working_directory = "/tmp"
		```
	*/
	WorkingDirectory string `hcl:"working_directory,optional" json:"working_directory,omitempty"`
	/*
		The process will be run as a daemon if set to true.

		Only valid for local execution.

		```hcl
		daemon = true
		```
	*/
	Daemon bool `hcl:"daemon,optional" json:"daemon,omitempty"`
	/*
		The timeout for the script to execute as a duration e.g. 30s.

		```hcl
		timeout = "60s"
		```
	*/
	Timeout string `hcl:"timeout,optional" json:"timeout,omitempty"`
	/*
		Environment variables to set for the script.

		```hcl
		resource "exec" "env" {
		  environment = {
		    FOO = "bar"
		  }

		  script = <<-EOF
		  #!/bin/bash
		  echo $${FOO}
		  EOF
		}
		```
	*/
	Environment map[string]string `hcl:"environment,optional" json:"environment,omitempty"`
	/*
		The image to use for the container.

		Only valid for remote execution in a standalone container.

		```hcl
		image {
		  name = "redis:latest"
		}
		```
	*/
	Image *ctypes.Image `hcl:"image,block" json:"image,omitempty"`
	/*
		A reference to a target container resource to execute the script in.

		Only valid for remote execution in an existing container.

		```hcl
		target = resource.container.alpine
		```
	*/
	Target *ctypes.Container `hcl:"target,optional" json:"target,omitempty"`
	/*
		The network to attach the container to.

		Only valid for remote execution in an existing container.

		```hcl
		network {
		  id = resource.network.main.meta.id
		}
		```
	*/
	Networks []ctypes.NetworkAttachment `hcl:"network,block" json:"networks,omitempty"`
	/*
		The volumes to mount to the container.

		Only valid for remote execution in an existing container.

		```hcl
		volume {
		  source = "./files"
		  destination = "/tmp/files"
		}
		```
	*/
	Volumes []ctypes.Volume `hcl:"volume,block" json:"volumes,omitempty"`
	/*
		The user to run the script as.

		Only valid for remote execution in an existing container.

		```hcl
		run_as = "root"
		```
	*/
	RunAs *ctypes.User `hcl:"run_as,block" json:"run_as,omitempty"`
	/*
		This is the pid of the parent process.

		Only valid for local execution.

		@computed
	*/
	PID int `hcl:"pid,optional" json:"pid,omitempty"`
	/*
		The exit code the script completed with.
	*/
	ExitCode int `hcl:"exit_code,optional" json:"exit_code,omitempty"`
	/*
		Any console output that the script outputs.
	*/
	Output map[string]string `hcl:"output,optional" json:"output,omitempty"`
	// @ignore
	Checksum string `hcl:"checksum,optional" json:"checksum,omitempty"`
}

func (e *Exec) Process() error {
	// check if it is a remote exec
	if e.Image != nil || e.Target != nil {
		// process volumes
		// make sure mount paths are absolute
		for i, v := range e.Volumes {
			e.Volumes[i].Source = utils.EnsureAbsolute(v.Source, e.Meta.File)
		}

		// make sure line endings are linux
		e.Script = strings.Replace(e.Script, "\r\n", "\n", -1)
	} else {
		if len(e.Networks) > 0 || len(e.Volumes) > 0 {
			return fmt.Errorf("unable to create local exec with networks or volumes")
		}
	}

	cs, err := utils.ChecksumFromInterface(e.Script)
	if err != nil {
		return fmt.Errorf("unable to generate checksum for script: %s", err)
	}

	e.Checksum = cs

	// do we have an existing resource in the state?
	// if so we need to set any computed resources for dependents
	cfg, err := config.LoadState()
	if err == nil {
		// try and find the resource in the state
		r, _ := cfg.FindResource(e.Meta.ID)

		if r != nil {
			kstate := r.(*Exec)
			e.PID = kstate.PID
			e.ExitCode = kstate.ExitCode
			e.Output = kstate.Output
		}
	}

	return nil
}
