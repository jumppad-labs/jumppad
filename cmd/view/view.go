package view

import "github.com/jumppad-labs/jumppad/pkg/clients"

type View interface {

	// Display starts the view, this is a blocking function
	Display() error

	// Logger returns the logger used by the view
	Logger() clients.Logger

	// UpdateStatus shows the current status message, if withTimer is set
	// the elapsed time that the the status has been shown for will also
	// be displayed
	UpdateStatus(message string, withTimer bool)
}
