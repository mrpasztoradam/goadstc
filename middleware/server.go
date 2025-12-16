package middleware

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/mrpasztoradam/goadstc"
	_ "github.com/mrpasztoradam/goadstc/docs" // Import generated docs
	"github.com/mrpasztoradam/goadstc/internal/ams"
	httpSwagger "github.com/swaggo/http-swagger/v2"
)

// Server represents the HTTP server
type Server struct {
	config     *Config
	middleware *Middleware
	handler    *Handler
	router     *chi.Mux
	httpServer *http.Server
}

// NewServer creates a new HTTP server
func NewServer(config *Config) (*Server, error) {
	// Parse AMS Net IDs
	var plcNetID ams.NetID
	if _, err := fmt.Sscanf(config.PLC.AMSNetID, "%d.%d.%d.%d.%d.%d",
		&plcNetID[0], &plcNetID[1], &plcNetID[2], &plcNetID[3], &plcNetID[4], &plcNetID[5]); err != nil {
		return nil, fmt.Errorf("invalid PLC AMS Net ID: %w", err)
	}

	var sourceNetID ams.NetID
	if _, err := fmt.Sscanf(config.PLC.SourceNetID, "%d.%d.%d.%d.%d.%d",
		&sourceNetID[0], &sourceNetID[1], &sourceNetID[2], &sourceNetID[3], &sourceNetID[4], &sourceNetID[5]); err != nil {
		return nil, fmt.Errorf("invalid source AMS Net ID: %w", err)
	}

	// Create ADS client
	client, err := goadstc.New(
		goadstc.WithTarget(config.PLC.Target),
		goadstc.WithAMSNetID(plcNetID),
		goadstc.WithSourceNetID(sourceNetID),
		goadstc.WithAMSPort(ams.Port(config.PLC.AMSPort)),
		goadstc.WithTimeout(config.Timeout()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create ADS client: %w", err)
	}

	// Create middleware and handler
	mw := NewMiddleware(client, config)
	h := NewHandler(mw)

	// Create server
	s := &Server{
		config:     config,
		middleware: mw,
		handler:    h,
	}

	// Setup router
	s.setupRouter()

	// Create HTTP server
	s.httpServer = &http.Server{
		Addr:         config.Address(),
		Handler:      s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return s, nil
}

// setupRouter configures the HTTP router
func (s *Server) setupRouter() {
	r := chi.NewRouter()

	// Middleware
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.Timeout(30 * time.Second))

	// CORS
	if s.config.Server.CORS.Enabled {
		r.Use(cors.Handler(cors.Options{
			AllowedOrigins:   s.config.Server.CORS.AllowedOrigins,
			AllowedMethods:   s.config.Server.CORS.AllowedMethods,
			AllowedHeaders:   s.config.Server.CORS.AllowedHeaders,
			AllowCredentials: s.config.Server.CORS.AllowCredentials,
			MaxAge:           300,
		}))
	}

	// API v1 routes
	r.Route("/api/v1", func(r chi.Router) {
		// Symbol operations
		r.Route("/symbols", func(r chi.Router) {
			r.Get("/", s.handler.HandleGetSymbolTable)
			r.Post("/read", s.handler.HandleBatchRead)
			r.Post("/write", s.handler.HandleBatchWrite)

			r.Route("/{name}", func(r chi.Router) {
				r.Get("/", s.handler.HandleGetSymbolInfo)
				r.Get("/value", s.handler.HandleReadSymbol)
				r.Post("/value", s.handler.HandleWriteSymbol)
			})
		})

		// Struct operations
		r.Route("/structs/{name}", func(r chi.Router) {
			r.Get("/", s.handler.HandleReadStruct)
			r.Post("/fields", s.handler.HandleWriteStructFields)
		})

		// System operations
		r.Get("/health", s.handler.HandleHealth)
		r.Get("/info", s.handler.HandleInfo)
		r.Get("/version", s.handler.HandleGetVersion)

		// PLC control operations
		r.Get("/state", s.handler.HandleGetState)
		r.Post("/control", s.handler.HandleControl)
	})

	// WebSocket endpoint
	r.Get("/ws/subscribe", s.handler.HandleWebSocket)

	// Swagger UI
	r.Get("/swagger-ui/*", httpSwagger.WrapHandler)

	// Root
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"name":"GoADS HTTP/WebSocket API","version":"1.0","docs":"/swagger-ui/index.html","websocket":"ws://localhost:8080/ws/subscribe"}`)
	})

	s.router = r
}

// Start starts the HTTP server
func (s *Server) Start() error {
	log.Printf("Starting server on %s", s.config.Address())
	log.Printf("PLC Target: %s", s.config.PLC.Target)
	log.Printf("API endpoints available at http://%s/api/v1", s.config.Address())

	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	log.Println("Shutting down server...")

	// Shutdown HTTP server
	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("failed to shutdown HTTP server: %w", err)
	}

	// Close ADS client
	// TODO: Add Close method if needed

	log.Println("Server stopped")
	return nil
}

// Router returns the chi router (useful for testing)
func (s *Server) Router() *chi.Mux {
	return s.router
}
