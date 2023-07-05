package resources

import "github.com/jumppad-labs/hclconfig/types"

// TypeSidecar is the resource string for a Sidecar resource
const TypeSidecar string = "sidecar"

// Sidecar defines a structure for creating Docker containers
type Sidecar struct {
	// embedded type holding name, etc
	types.ResourceMetadata `hcl:",remain"`

	Target string `hcl:"target" json:"target"`

	Image       Image             `hcl:"image,block" json:"image"`                          // image to use for the container
	Entrypoint  []string          `hcl:"entrypoint,optional" json:"entrypoint,omitempty"`   // entrypoint to use when starting the container
	Command     []string          `hcl:"command,optional" json:"command,omitempty"`         // command to use when starting the container
	Environment map[string]string `hcl:"environment,optional" json:"environment,omitempty"` // environment variables to set when starting the container
	Volumes     []Volume          `hcl:"volume,block" json:"volumes,omitempty"`             // volumes to attach to the container

	Privileged bool `hcl:"privileged,optional" json:"privileged,omitempty"` // run the container in privileged mode?

	// resource constraints
	Resources *Resources `hcl:"resources,block" json:"resources,omitempty"` // resource constraints for the container

	// health checks for the container
	HealthCheck *HealthCheckContainer `hcl:"health_check,block" json:"health_check,omitempty"`

	MaxRestartCount int `hcl:"max_restart_count,optional" json:"max_restart_count,omitempty"`

	// Output parameters

	// FQRN is the fully qualified domain name for the container the sidecar is linked to, this can be used
	// to access the sidecar from other sources
	FQRN string `hcl:"fqrn,optional" json:"fqrn,omitempty"`
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

	// do we have an existing resource in the state?
	// if so we need to set any computed resources for dependents
	cfg, err := LoadState()
	if err == nil {
		// try and find the resource in the state
		r, _ := cfg.FindResource(c.ID)
		if r != nil {
			kstate := r.(*Sidecar)
			c.FQRN = kstate.FQRN
		}
	}

	return nil
}
