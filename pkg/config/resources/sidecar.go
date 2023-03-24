package resources

import "github.com/shipyard-run/hclconfig/types"

// TypeSidecar is the resource string for a Sidecar resource
const TypeSidecar string = "sidecar"

// Sidecar defines a structure for creating Docker containers
type Sidecar struct {
	// embedded type holding name, etc
	types.ResourceMetadata `hcl:",remain"`

	Depends []string `hcl:"depends_on,optional" json:"depends,omitempty"`

	Target string `hcl:"target" json:"target"`

	Image      Image             `hcl:"image,block" json:"image"`                        // image to use for the container
	Entrypoint []string          `hcl:"entrypoint,optional" json:"entrypoint,omitempty"` // entrypoint to use when starting the container
	Command    []string          `hcl:"command,optional" json:"command,omitempty"`       // command to use when starting the container
	Env        map[string]string `hcl:"env,optional" json:"env,omitempty"`               // environment variables to set when starting the container
	Volumes    []Volume          `hcl:"volume,block" json:"volumes,omitempty"`           // volumes to attach to the container

	Privileged bool `hcl:"privileged,optional" json:"privileged,omitempty"` // run the container in privileged mode?

	// resource constraints
	Resources *Resources `hcl:"resources,block" json:"resources,omitempty"` // resource constraints for the container

	// health checks for the container
	HealthCheck *HealthCheck `hcl:"health_check,block" json:"health_check,omitempty" mapstructure:"health_check"`

	MaxRestartCount int `hcl:"max_restart_count,optional" json:"max_restart_count,omitempty" mapstructure:"max_restart_count"`
}

func (c *Sidecar) Process() error {
	// process volumes
	for i, v := range c.Volumes {
		// make sure mount paths are absolute when type is bind
		if v.Type == "" || v.Type == "bind" {
			c.Volumes[i].Source = ensureAbsolute(v.Source, c.File)
			c.Volumes[i].Destination = ensureAbsolute(v.Destination, c.File)
		}
	}

	return nil
}
