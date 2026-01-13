package server

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/dict-simulator/go/internal/logger"
)

// Server wraps the HTTP server with graceful shutdown support
type Server struct {
	httpServer *http.Server
	port       int
}

// New creates a new Server instance
func New(handler http.Handler, port int) *Server {
	return &Server{
		httpServer: &http.Server{
			Addr:         fmt.Sprintf(":%d", port),
			Handler:      handler,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		port: port,
	}
}

// Start begins listening and serving requests (blocks until server stops)
func (s *Server) Start() error {
	logger.Info("server starting", zap.Int("port", s.port))

	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// Shutdown gracefully stops the server with the given context
func (s *Server) Shutdown(ctx context.Context) error {
	logger.Info("server shutting down")
	return s.httpServer.Shutdown(ctx)
}

// ListenAndServeWithGracefulShutdown starts the server and handles OS signals for graceful shutdown
func (s *Server) ListenAndServeWithGracefulShutdown() {
	// Channel to signal shutdown
	done := make(chan bool, 1)

	// Listen for OS signals
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := s.Shutdown(ctx); err != nil {
			logger.Error("server shutdown error", zap.Error(err))
		}

		done <- true
	}()

	// Start server
	logger.Info("DICT Simulator running", zap.String("addr", fmt.Sprintf("http://localhost:%d", s.port)))

	if err := s.Start(); err != nil {
		logger.Fatal("server error", zap.Error(err))
	}

	<-done
	logger.Info("server stopped")
}
