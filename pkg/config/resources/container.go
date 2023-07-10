package resources

import (
	"github.com/jumppad-labs/hclconfig/types"
)

// TypeContainer is the resource string for a Container resource
const TypeContainer string = "container"

// Container defines a structure for creating Docker containers
type Container struct {
	// embedded type holding name, etc
	types.ResourceMetadata `hcl:",remain"`

	Networks        []NetworkAttachment `hcl:"network,block" json:"networks,omitempty"`           // Attach to the correct network // only when Image is specified
	Image           *Image              `hcl:"image,block" json:"image"`                          // Image to use for the container
	Entrypoint      []string            `hcl:"entrypoint,optional" json:"entrypoint,omitempty"`   // entrypoint to use when starting the container
	Command         []string            `hcl:"command,optional" json:"command,omitempty"`         // command to use when starting the container
	Environment     map[string]string   `hcl:"environment,optional" json:"environment,omitempty"` // environment variables to set when starting the container
	Volumes         []Volume            `hcl:"volume,block" json:"volumes,omitempty"`             // volumes to attach to the container
	Ports           []Port              `hcl:"port,block" json:"ports,omitempty"`                 // ports to expose
	PortRanges      []PortRange         `hcl:"port_range,block" json:"port_ranges,omitempty"`     // range of ports to expose
	DNS             []string            `hcl:"dns,optional" json:"dns,omitempty"`                 // Add custom DNS servers to the container
	Privileged      bool                `hcl:"privileged,optional" json:"privileged,omitempty"`   // run the container in privileged mode?
	MaxRestartCount int                 `hcl:"max_restart_count,optional" json:"max_restart_count,omitempty"`

	// resource constraints
	Resources *Resources `hcl:"resources,block" json:"resources,omitempty"` // resource constraints for the container

	// health checks for the container
	HealthCheck *HealthCheckContainer `hcl:"health_check,block" json:"health_check,omitempty"`

	// User block for mapping the user id and group id inside the container
	RunAs *User `hcl:"run_as,block" json:"run_as,omitempty"`

	// Enables containers to be built on the fly
	Build *Build `hcl:"build,block" json:"build"`

	// Output parameters

	// FQRN is the fully qualified domain name for the container, this can be used
	// to access the container from other sources
	FQRN string `hcl:"fqrn,optional" json:"fqrn,omitempty"`
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

// Resources allows the setting of resource constraints for the Container
type Resources struct {
	CPU    int   `hcl:"cpu,optional" json:"cpu,omitempty"`         // cpu limit for the container where 1 CPU = 1000
	CPUPin []int `hcl:"cpu_pin,optional" json:"cpu_pin,omitempty"` // pin the container to one or more cpu cores
	Memory int   `hcl:"memory,optional" json:"memory,omitempty"`   // max memory the container can consume in MB
}

// Volume defines a folder, Docker volume, or temp folder to mount to the Container
type Volume struct {
	Source                      string `hcl:"source" json:"source"`                                                                    // source path on the local machine for the volume
	Destination                 string `hcl:"destination" json:"destination"`                                                          // path to mount the volume inside the container
	Type                        string `hcl:"type,optional" json:"type,omitempty"`                                                     // type of the volume to mount [bind, volume, tmpfs]
	ReadOnly                    bool   `hcl:"read_only,optional" json:"read_only,omitempty"`                                           // specify that the volume is mounted read only
	BindPropagation             string `hcl:"bind_propagation,optional" json:"bind_propagation,omitempty"`                             // propagation mode for bind mounts [shared, private, slave, rslave, rprivate]
	BindPropagationNonRecursive bool   `hcl:"bind_propagation_non_recursive,optional" json:"bind_propagation_non_recursive,omitempty"` // recursive bind mount, default true
}

// Build allows you to define the conditions for building a container
// on run from a Dockerfile
type Build struct {
	DockerFile string            `hcl:"dockerfile,optional" json:"dockerfile,omitempty"` // Location of build file inside build context defaults to ./Dockerfile
	Context    string            `hcl:"context" json:"context"`                          // Path to build context
	Tag        string            `hcl:"tag,optional" json:"tag,omitempty"`               // Image tag, defaults to latest
	Args       map[string]string `hcl:"args,optional" json:"args,omitempty"`             // Build args to pass  to the container

	// output

	// Checksum is calculated from the Context files
	Checksum string `hcl:"checksum,optional" json:"checksum,omitempty"`
}

func (c *Container) Process() error {
	// process volumes
	for i, v := range c.Volumes {
		// make sure mount paths are absolute when type is bind
		if v.Type == "" || v.Type == "bind" {
			c.Volumes[i].Source = ensureAbsolute(v.Source, c.File)
		}
	}

	// make sure build paths are absolute
	if c.Build != nil {
		c.Build.Context = ensureAbsolute(c.Build.Context, c.File)
	}

	// do we have an existing resource in the state?
	// if so we need to set any computed resources for dependents
	cfg, err := LoadState()
	if err == nil {
		// try and find the resource in the state
		r, _ := cfg.FindResource(c.ID)
		if r != nil {
			kstate := r.(*Container)
			c.FQRN = kstate.FQRN

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
