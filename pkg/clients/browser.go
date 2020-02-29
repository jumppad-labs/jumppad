package clients

import (
	"os/exec"
	"runtime"
)

// Browser handles interactions between Shipyard and the browser
type Browser interface {
	Open(string) error
}

// BrowserImpl is a concrete implementation of the Browser interface
type BrowserImpl struct{}

// Open a URI in a new browser window
func (b *BrowserImpl) Open(uri string) error {
	openCommand := "open"
	if runtime.GOOS == "linux" {
		openCommand = "xdg-open"
	}

	cmd := exec.Command(openCommand, uri)
	return cmd.Run()
}
