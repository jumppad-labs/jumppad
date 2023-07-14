package view

import (
	"fmt"

	"github.com/jumppad-labs/jumppad/pkg/clients"

	tea "github.com/charmbracelet/bubbletea"
)

var statuses = []string{
	"idle",
	"checking for changes",
	"applying changes",
}

type CmdView struct {
	program      *tea.Program
	logger       clients.Logger
	initialModel model
}

func NewCmdView(useTTY bool) (*CmdView, error) {
	c := &CmdView{}
	c.initialModel = initialModel()
	if useTTY {
		c.program = tea.NewProgram(c.initialModel, tea.WithAltScreen())

		mw := &messageWriter{
			program: c.program,
		}

		c.logger = clients.NewTTYLogger(mw, clients.LogLevelInfo)
	} else {
		c.program = tea.NewProgram(c.initialModel, tea.WithoutRenderer())

		mw := &messageWriter{
			program: c.program,
		}

		c.logger = clients.NewLogger(mw, clients.LogLevelInfo)
	}

	return c, nil
}

// Display starts the view, this is a blocking function
func (c *CmdView) Display() error {
	if _, err := c.program.Run(); err != nil {
		return fmt.Errorf("unable to start bubbletea view: %s", err)
	}

	return nil
}

// Logger returns the logger used by the view
func (c *CmdView) Logger() clients.Logger {
	return c.logger
}

// UpdateStatus shows the current status message, if withTimer is set
// the elapsed time that the the status has been shown for will also
// be displayed
func (c *CmdView) UpdateStatus(message string, withTimer bool) {
	c.program.Send(StatusMsg{Message: message, ShowElapsed: withTimer})
}
