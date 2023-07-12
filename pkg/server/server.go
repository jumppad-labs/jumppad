package server

import (
	"github.com/gofiber/fiber/v2"
	"github.com/jumppad-labs/jumppad/pkg/clients"

	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/websocket/v2"
)

type API struct {
	bindAddr string
	app      *fiber.App
	log      clients.Logger
}

// New creates a new server
func New(addr string, l clients.Logger) *API {
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

	s.app.Use(cors.New())
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
	s.app.Post("/validate", s.handleValidate)

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
