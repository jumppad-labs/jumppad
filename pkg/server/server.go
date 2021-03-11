package server

import (
	"github.com/gofiber/fiber/v2"
	"github.com/hashicorp/go-hclog"

	"github.com/gofiber/websocket/v2"
)

type API struct {
	app *fiber.App
	log hclog.Logger
}

// New creates a new server
func New(l hclog.Logger) *API {
	return &API{
		app: fiber.New(),
		log: l,
	}
}

// Start the API server
func (s *API) Start() {
	s.log.Debug("Starting API server")

	s.app.Use("/terminal", func(c *fiber.Ctx) error {
		// IsWebSocketUpgrade returns true if the client
		// requested upgrade to the WebSocket protocol.
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})

	s.app.Get("/terminal", websocket.New(s.terminalWebsocket))

	// Start the server but do not block
	go s.app.Listen(":3000")
}

// Stop the API server
func (s *API) Stop() {
	s.app.Shutdown()
}
