package providers

// Provider defines an interface to be implemented by providers
type Provider interface {
	// Create is called when a resource does not exist or creation has previously
	// failed and 'up' is run
	Create() error

	// Destroy is called when a resource is failed or created and 'down' is run
	Destroy() error

	// Refresh is called when a resource is created and 'up' is run
	Refresh() error

	// Changed returns if a resource has changed since the last run
	Changed() (bool, error)

	// Lookup is a utility to determine the existence of a resource
	Lookup() ([]string, error)
}

// ConfigWrapper allows the provider config to be deserialized to a type
type ConfigWrapper struct {
	Type  string
	Value interface{}
}
