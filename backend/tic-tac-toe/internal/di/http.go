package di

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/fx"

	"tic-tac-toe/internal/config"
	"tic-tac-toe/internal/logging"
)

type HTTPConfig struct {
	Port string
}

const (
	httpReadHeaderTimeout = 5 * time.Second
	httpReadTimeout       = 10 * time.Second
	httpWriteTimeout      = 10 * time.Second
	httpIdleTimeout       = 60 * time.Second
)

func (c HTTPConfig) Validate() error {
	if c.Port == "" {
		return fmt.Errorf("http port must not be empty")
	}
	return nil
}

func NewHTTPConfig() HTTPConfig {
	return HTTPConfig{Port: normalizeHTTPPort(config.String("HTTP_PORT", "8080"))}
}

func normalizeHTTPPort(port string) string {
	port = strings.TrimSpace(port)
	if port == "" {
		return ":8080"
	}

	port = strings.TrimPrefix(port, ":")
	if _, err := strconv.Atoi(port); err != nil {
		return port
	}

	return ":" + port
}

func (c HTTPConfig) Addr() string {
	return normalizeHTTPPort(c.Port)
}

func NewHTTPServer(router chi.Router, config HTTPConfig) *http.Server {
	return &http.Server{
		Addr:              config.Addr(),
		Handler:           router,
		ReadHeaderTimeout: httpReadHeaderTimeout,
		ReadTimeout:       httpReadTimeout,
		WriteTimeout:      httpWriteTimeout,
		IdleTimeout:       httpIdleTimeout,
		MaxHeaderBytes:    1 << 20,
	}
}

func RegisterHTTPServer(lifecycle fx.Lifecycle, server *http.Server) {
	var listener net.Listener

	lifecycle.Append(fx.Hook{
		OnStart: func(context.Context) error {
			log.Printf("starting HTTP server on %s", server.Addr)

			var err error
			listener, err = net.Listen("tcp", server.Addr)
			if err != nil {
				return fmt.Errorf("listen on %s: %w", server.Addr, err)
			}

			go func() {
				if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
					log.Printf("HTTP server failed: %v", err)
				}
			}()

			return nil
		},
		OnStop: func(ctx context.Context) error {
			log.Printf("stopping HTTP server on %s", server.Addr)

			if err := server.Shutdown(ctx); err != nil {
				log.Printf("HTTP server shutdown failed: %v", err)
				return err
			}

			log.Printf("HTTP server stopped on %s", server.Addr)
			return nil
		},
	})
}
