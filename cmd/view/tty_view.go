package view

import (
	"fmt"
	"os"

	"github.com/jumppad-labs/jumppad/pkg/clients/logger"

	tea "github.com/charmbracelet/bubbletea"
)

var statuses = []string{
	"idle",
	"checking for changes",
	"applying changes",
}

type TTYView struct {
	program      *tea.Program
	logger       logger.Logger
	initialModel model
}

func NewTTYView() (*TTYView, error) {
	c := &TTYView{}
	c.initialModel = initialModel()

	mw := &messageWriter{}

	level, present := os.LookupEnv("LOG_LEVEL")
	if !present {
		level = logger.LogLevelInfo
	}

	c.logger = logger.NewTTYLogger(mw, level)
	c.initialModel.logger = c.logger

	c.program = tea.NewProgram(c.initialModel, tea.WithAltScreen())

	// once the program has been created set a reference to the writer so that
	// log lines get directed to bubbletea
	mw.program = c.program

	return c, nil
}

// Display starts the view, this is a blocking function
func (c *TTYView) Display() error {
	if _, err := c.program.Run(); err != nil {
		return fmt.Errorf("unable to start bubbletea view: %s", err)
	}

	return nil
}

// Logger returns the logger used by the view
func (c *TTYView) Logger() logger.Logger {
	return c.logger
}

// UpdateStatus shows the current status message, if withTimer is set
// the elapsed time that the the status has been shown for will also
// be displayed
func (c *TTYView) UpdateStatus(message string, withTimer bool) {
	c.program.Send(StatusMsg{Message: message, ShowElapsed: withTimer})
}
