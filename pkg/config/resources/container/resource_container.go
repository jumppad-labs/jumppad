package container

import (
	"strings"

	"github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/config"
	"github.com/jumppad-labs/jumppad/pkg/config/resources/healthcheck"
	"github.com/jumppad-labs/jumppad/pkg/utils"
)

// TypeContainer is the resource string for a Container resource
const TypeContainer string = "container"

// Container defines a structure for creating Docker containers
type Container struct {
	// embedded type holding name, etc
	types.ResourceMetadata `hcl:",remain"`

	Networks        []NetworkAttachment `hcl:"network,block" json:"networks,omitempty"`           // Attach to the correct network // only when Image is specified
	Image           *Image              `hcl:"image,block" json:"image"`                          // Image to use for the container
	Entrypoint      []string            `hcl:"entrypoint,optional" json:"entrypoint,omitempty"`   // Entrypoint to use when starting the container
	Command         []string            `hcl:"command,optional" json:"command,omitempty"`         // Command to use when starting the container
	Environment     map[string]string   `hcl:"environment,optional" json:"environment,omitempty"` // Environment variables to set when starting the container
	Volumes         []Volume            `hcl:"volume,block" json:"volumes,omitempty"`             // Volumes to attach to the container
	Ports           []Port              `hcl:"port,block" json:"ports,omitempty"`                 // Ports to expose
	PortRanges      []PortRange         `hcl:"port_range,block" json:"port_ranges,omitempty"`     // Range of ports to expose
	DNS             []string            `hcl:"dns,optional" json:"dns,omitempty"`                 // Add custom DNS servers to the container
	Privileged      bool                `hcl:"privileged,optional" json:"privileged,omitempty"`   // Run the container in privileged mode?
	Capabilities    *Capabilities       `hcl:"capabilities,block" json:"capabilities,omitempty"`  // Capabilities to add or drop from the container
	MaxRestartCount int                 `hcl:"max_restart_count,optional" json:"max_restart_count,omitempty"`

	// resource constraints
	Resources *Resources `hcl:"resources,block" json:"resources,omitempty"` // resource constraints for the container

	// health checks for the container
	HealthCheck *healthcheck.HealthCheckContainer `hcl:"health_check,block" json:"health_check,omitempty"`

	// User block for mapping the user id and group id inside the container
	RunAs *User `hcl:"run_as,block" json:"run_as,omitempty"`

	// Output parameters

	// ContainerName is the fully qualified domain name for the container, this can be used
	// to access the container from other sources
	ContainerName string `hcl:"container_name,optional" json:"container_name,omitempty"`
}

type User struct {
	// Username or UserID of the user to run the container as
	User string `hcl:"user" json:"user,omitempty"`
	// Group is the GroupID of the user to run the container as
	Group string `hcl:"group" json:"group,omitempty"`
}

type NetworkAttachment struct {
	ID        string   `hcl:"id" json:"id"`
	IPAddress string   `hcl:"ip_address,optional" json:"ip_address,omitempty"` // Optional address to assign
	Aliases   []string `hcl:"aliases,optional" json:"aliases,omitempty"`       // Network aliases for the resource

	// output

	// Name will equal the name of the network as created by jumppad
	Name string `hcl:"name,optional" json:"name,omitempty"`

	// AssignedAddress will equal if IPAddress is set, else it will be the value automatically
	// assigned from the network
	AssignedAddress string `hcl:"assigned_address,optional" json:"assigned_address,omitempty"`
}

type NetworkAttachments []NetworkAttachment

// Resources allows the setting of resource constraints for the Container
type Resources struct {
	CPU    int   `hcl:"cpu,optional" json:"cpu,omitempty"`         // cpu limit for the container where 1 CPU = 1000
	CPUPin []int `hcl:"cpu_pin,optional" json:"cpu_pin,omitempty"` // pin the container to one or more cpu cores
	Memory int   `hcl:"memory,optional" json:"memory,omitempty"`   // max memory the container can consume in MB
}

type Capabilities struct {
	Add  []string `hcl:"add,optional" json:"add"`   // CapAdd is a list of kernel capabilities to add to the container
	Drop []string `hcl:"drop,optional" json:"drop"` // CapDrop is a list of kernel capabilities to remove from the container
}

// Volume defines a folder, Docker volume, or temp folder to mount to the Container
type Volume struct {
	Source                      string `hcl:"source" json:"source"`                                                                    // source path on the local machine for the volume
	Destination                 string `hcl:"destination" json:"destination"`                                                          // path to mount the volume inside the container
	Type                        string `hcl:"type,optional" json:"type,omitempty"`                                                     // type of the volume to mount [bind, volume, tmpfs]
	ReadOnly                    bool   `hcl:"read_only,optional" json:"read_only,omitempty"`                                           // specify that the volume is mounted read only
	BindPropagation             string `hcl:"bind_propagation,optional" json:"bind_propagation,omitempty"`                             // propagation mode for bind mounts [shared, private, slave, rslave, rprivate]
	BindPropagationNonRecursive bool   `hcl:"bind_propagation_non_recursive,optional" json:"bind_propagation_non_recursive,omitempty"` // recursive bind mount, default true
	SelinuxRelabel              string `hcl:"selinux_relabel,optional" json:"selinux_relabel,optional"`                                // selinux_relabeling ["", shared, private]
}

type Volumes []Volume

func (c *Container) Process() error {
	// process volumes
	for i, v := range c.Volumes {
		// make sure mount paths are absolute when type is bind, unless this is the docker sock
		if v.Type == "" || v.Type == "bind" {
			c.Volumes[i].Source = utils.EnsureAbsolute(v.Source, c.File)
		}
	}

	// make sure line endings are linux
	if c.HealthCheck != nil {
		for i := range c.HealthCheck.Exec {
			c.HealthCheck.Exec[i].Script = strings.Replace(c.HealthCheck.Exec[i].Script, "\r\n", "\n", -1)
		}
	}

	// do we have an existing resource in the state?
	// if so we need to set any computed resources for dependents
	cfg, err := config.LoadState()
	if err == nil {
		// try and find the resource in the state
		r, _ := cfg.FindResource(c.ID)
		if r != nil {
			kstate := r.(*Container)
			c.ContainerName = kstate.ContainerName

			// add the image id from state
			c.Image.ID = kstate.Image.ID

			// add the network addresses
			for _, a := range kstate.Networks {
				for i, m := range c.Networks {
					if m.ID == a.ID {
						c.Networks[i].AssignedAddress = a.AssignedAddress
						c.Networks[i].Name = a.Name
						break
					}
				}
			}
		}
	}

	return nil
}
