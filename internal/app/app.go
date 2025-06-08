package app

import (
	"context"
	"fmt"
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
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Set up logger
	setupLogger(cfg)

	// Initialize authentication service
	authSvc := auth.NewService(cfg)

	// Initialize TUI
	ui, err := tui.NewApp(cfg, authSvc)
	if err != nil {
		return fmt.Errorf("failed to initialize TUI: %w", err)
	}

	log.Println("Starting GOReily...")

	// Run the application
	if err := ui.Run(ctx); err != nil {
		return fmt.Errorf("application error: %w", err)
	}

	return nil
}

func setupLogger(cfg *config.Config) {
	// Configure standard logger
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.SetPrefix("goreilly: ")

	// In debug mode, we'll log more verbosely
	if !cfg.Debug {
		// In non-debug mode, discard debug logs
		log.SetOutput(io.Discard)
	}
}
