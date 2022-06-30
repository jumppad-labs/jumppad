package server

import (
	"github.com/gofiber/fiber/v2"
	"github.com/hashicorp/go-hclog"

	"github.com/gofiber/websocket/v2"
)

type API struct {
	bindAddr string
	app      *fiber.App
	log      hclog.Logger
}

// New creates a new server
func New(addr string, l hclog.Logger) *API {
	config := fiber.Config{
		DisableStartupMessage: true,
	}

	return &API{
		bindAddr: addr,
		app:      fiber.New(config),
		log:      l,
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

	// Start the server
	err := s.app.Listen(s.bindAddr)
	if err != nil {
		s.log.Error("Listen exit with", "error", err)
	}

	s.log.Info("Listen exit")
}

// Stop the API server
func (s *API) Stop() {
	s.log.Info("Shutdown API server")
	s.app.Shutdown()
}
