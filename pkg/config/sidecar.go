package config

// TypeSidecar is the resource string for a Sidecar resource
const TypeSidecar ResourceType = "sidecar"

// Sidecar defines a structure for creating Docker containers
type Sidecar struct {
	// embedded type holding name, etc
	ResourceInfo `hcl:",remain" mapstructure:",squash"`

	Depends []string `hcl:"depends_on,optional" json:"depends,omitempty"`

	Target string `hcl:"target" json:"target"`

	Image       Image             `hcl:"image,block" json:"image"`                                         // image to use for the container
	Entrypoint  []string          `hcl:"entrypoint,optional" json:"entrypoint,omitempty"`                  // entrypoint to use when starting the container
	Command     []string          `hcl:"command,optional" json:"command,omitempty"`                        // command to use when starting the container
	Environment []KV              `hcl:"env,block" json:"env,omitempty" mapstructure:"env"`                // environment variables to set when starting the container
	EnvVar      map[string]string `hcl:"env_var,optional" json:"env_var,omitempty" mapstructure:"env_var"` // environment variables to set when starting the container
	Volumes     []Volume          `hcl:"volume,block" json:"volumes,omitempty"`                            // volumes to attach to the container

	Privileged bool `hcl:"privileged,optional" json:"privileged,omitempty"` // run the container in privileged mode?

	// resource constraints
	Resources *Resources `hcl:"resources,block" json:"resources,omitempty"` // resource constraints for the container

	// health checks for the container
	HealthCheck *HealthCheck `hcl:"health_check,block" json:"health_check,omitempty" mapstructure:"health_check"`

	MaxRestartCount int `hcl:"max_restart_count,optional" json:"max_restart_count,omitempty" mapstructure:"max_restart_count"`
}

// NewSidecar returns a new Container resource with the correct default options
func NewSidecar(name string) *Sidecar {
	return &Sidecar{ResourceInfo: ResourceInfo{Name: name, Type: TypeSidecar, Status: PendingCreation}}
}
