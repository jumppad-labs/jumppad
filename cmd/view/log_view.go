package view

import (
	"fmt"
	"math"
	"os"
	"time"

	"github.com/jumppad-labs/jumppad/pkg/clients/logger"
)

type LogView struct {
	logger           logger.Logger
	statusMessage    string
	statusTimer      bool
	statusTimerStart time.Time
}

func NewLogView() (*LogView, error) {
	c := &LogView{}
	c.logger = logger.NewLogger(os.Stdout, logger.LogLevelDebug)

	return c, nil
}

// Display starts the view, this is a blocking function
func (c *LogView) Display() error {
	for {
		if c.statusTimer {
			et := time.Since(c.statusTimerStart)
			elapsed := fmt.Sprintf("%ds", int(math.Round(float64(et)/float64(time.Second))))

			c.Logger().Info(c.statusMessage, "elapsed_time", elapsed)
		}

		time.Sleep(1 * time.Second)
	}

	return nil
}

// Logger returns the logger used by the view
func (c *LogView) Logger() logger.Logger {
	return c.logger
}

// UpdateStatus shows the current status message, if withTimer is set
// the elapsed time that the the status has been shown for will also
// be displayed
func (c *LogView) UpdateStatus(message string, withTimer bool) {
	c.statusTimer = withTimer

	if withTimer {
		c.statusMessage = message
		c.statusTimerStart = time.Now()

		return
	}

	c.Logger().Info(message)
}
