package providers

// Provider defines an interface to be implemented by providers
type Provider interface {
	Create() error
	Destroy() error
	Lookup() ([]string, error)
}

// ConfigWrapper alows the provider config to be deserialized to a type
type ConfigWrapper struct {
	Type  string
	Value interface{}
}
