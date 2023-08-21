package server

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"
	"github.com/jumppad-labs/jumppad/pkg/clients/logger"
)

type API struct {
	server *http.Server
	log    logger.Logger
}

// New creates a new server
func New(addr string, l logger.Logger) *API {
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.RequestLogger(&middleware.DefaultLogFormatter{Logger: log.New(l.StandardWriter(), "", log.Default().Flags()), NoColor: true}))
	router.Use(middleware.Recoverer)
	router.Use(cors.Handler(cors.Options{
		// AllowedOrigins: []string{"https://foo.com"}, // Use this to allow specific origin hosts
		// AllowedOrigins: []string{"https://*", "http://*"},
		AllowOriginFunc:  func(r *http.Request, origin string) bool { return true },
		AllowedMethods:   []string{"HEAD", "GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	server := &http.Server{
		Addr:     addr,
		Handler:  router,
		ErrorLog: log.New(l.StandardWriter(), "", log.Default().Flags()),
	}

	api := &API{
		server: server,
		log:    l,
	}

	router.HandleFunc("/", api.catchAll)
	router.Get("/terminal", api.terminal)
	router.Post("/validate/{task}/{action}", api.validation)

	return api
}

// Start the API server
func (a *API) Start(tlsCert, tlsKey string) {
	a.log.Debug("Starting API server")

	// // Start the server
	err := a.server.ListenAndServeTLS(tlsCert, tlsKey)
	if err != nil && err != http.ErrServerClosed {
		a.log.Error("Listen exit with", "error", err)
	}

	a.log.Info("Listen exit")
}

// Stop the API server
func (s *API) Stop() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	s.log.Info("Shutdown API server")
	s.server.Shutdown(ctx)
}
