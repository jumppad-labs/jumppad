package constants

// PropertyStatus is the key for the Metadata property that contains the status
const PropertyStatus = "status"

const (
	// StatusCreated is set once the resource has been successfully created
	StatusCreated = "created"

	// StatusTainted indicates that the resources has been successfully created
	// but should be destroyed and re-created
	StatusTainted = "tainted"

	// StatusFailed indicates that the resource failed to create
	StatusFailed = "failed"

	// StatusDisabled indicates that the resources has been disabled and no
	// resources have been created
	StatusDisabled = "disabled"
)
