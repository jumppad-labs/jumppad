package types

type Container struct {
	Name            string
	Networks        []NetworkAttachment
	Image           *Image
	Entrypoint      []string
	Command         []string
	Environment     map[string]string
	Volumes         []Volume
	Ports           []Port
	PortRanges      []PortRange
	DNS             []string
	Privileged      bool
	MaxRestartCount int

	// resource constraints
	Resources *Resources

	// User block for mapping the user id and group id inside the container
	RunAs *User
}

type User struct {
	// Username or UserID of the user to run the container as
	User string
	// Group is the GroupID of the user to run the container as
	Group string
}

type NetworkAttachment struct {
	ID          string // network or container id
	Name        string
	IPAddress   string
	Aliases     []string
	Subnet      string
	IsContainer bool // is the network attachment a container or normal network
}

// Resources allows the setting of resource constraints for the Container
type Resources struct {
	CPU    int
	CPUPin []int
	Memory int
}

// Volume defines a folder, Docker volume, or temp folder to mount to the Container
type Volume struct {
	Source                      string
	Destination                 string
	Type                        string
	ReadOnly                    bool
	BindPropagation             string
	BindPropagationNonRecursive bool
	SelinuxRelabel              string
}

// Port is a port mapping
type Port struct {
	Local         string
	Remote        string
	Host          string
	Protocol      string
	OpenInBrowser string
}

// PortRange allows a range of ports to be mapped
type PortRange struct {
	Range      string
	EnableHost bool
	Protocol   string
}

// Image defines a docker image which will be pushed to the clusters Docker
// registry
type Image struct {
	ID   string
	Name string
	// Username is the Docker registry user to use for private repositories
	Username string
	// Password is the Docker registry password to use for private repositories
	Password string
}

type Build struct {
	Name       string
	DockerFile string            // Name of the Dockerfile to use, must be in context
	Context    string            // Context to copy to the build process
	Ignore     []string          // globbed list of files to ignore in the context, same as .dockerignore
	Args       map[string]string // Arguments to pass to the build process
}
