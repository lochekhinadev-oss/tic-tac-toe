package di

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/fx"

	"tic-tac-toe/internal/config"
	"tic-tac-toe/internal/logging"
)

type HTTPConfig struct {
	Addr string
}

const (
	httpReadHeaderTimeout = 5 * time.Second
	httpReadTimeout       = 10 * time.Second
	httpWriteTimeout      = 10 * time.Second
	httpIdleTimeout       = 60 * time.Second
)

func (c HTTPConfig) Validate() error {
	if c.Addr == "" {
		return fmt.Errorf("http addr must not be empty")
	}
	return nil
}

func NewHTTPConfig() HTTPConfig {
	return HTTPConfig{Addr: config.String("HTTP_ADDR", ":8080")}
}

func NewHTTPServer(router chi.Router, config HTTPConfig) *http.Server {
	return &http.Server{
		Addr:              config.Addr,
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
