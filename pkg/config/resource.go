package config

// Resource defines an interface for resources which can be created with
// Shipyard
type Resource interface {
	// Validate the config ensuring all fields have the correct values
	Validate() error
}
