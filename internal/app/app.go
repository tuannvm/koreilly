package app

import (
	"context"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/tuannvm/goreilly/internal/auth"
	"github.com/tuannvm/goreilly/internal/config"
	"github.com/tuannvm/goreilly/internal/tui"
)

// Run initializes and runs the application.
func Run() error {
	// Set up context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		log.Printf("Received signal %s, shutting down...\n", sig)
		cancel()
	}()

	// Initialize configuration
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// Set up logger
	setupLogger(cfg)


	// Initialize services
	authSvc, err := auth.NewService(cfg)
	if err != nil {
		return err
	}

	// Initialize TUI
	ui, err := tui.NewApp(cfg, authSvc)
	if err != nil {
		return err
	}

	// Run the application
	return ui.Run(ctx)
}

func setupLogger(cfg *config.Config) {
	// Configure standard logger
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.SetPrefix("koreilly: ")

	// In debug mode, we'll log more verbosely
	if !cfg.Debug {
		// In non-debug mode, discard debug logs
		log.SetOutput(io.Discard)
	}
}
