package providers

// Provider defines an interface to be implemented by providers
type Provider interface {
	Create() error
	Destroy() error
	Lookup() (string, error)
}
