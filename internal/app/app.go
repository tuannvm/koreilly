package app

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

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
	authSvc, err := auth.NewService(cfg)
	if err != nil {
		return fmt.Errorf("failed to initialize auth service: %w", err)
	}

	// If we already have a valid token, skip the login TUI entirely.
	if authSvc.IsAuthenticated() {
		msg := "You are already authenticated! (orm-jwt cookie found).\nMain book browsing features are not implemented yet."
		fmt.Println(msg)
		log.Println(msg)
		return nil
	}

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
	// Create logs directory if it doesn't exist
	if err := os.MkdirAll("logs", 0755); err != nil {
		log.Printf("Failed to create logs directory: %v", err)
	}

	// Create log file with timestamp
	logFile := fmt.Sprintf("logs/goreilly_%s.log", time.Now().Format("20060102_150405"))
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Printf("Failed to open log file: %v", err)
	} else {
		// Log to both file and stderr
		multiWriter := io.MultiWriter(os.Stderr, file)
		log.SetOutput(multiWriter)
	}

	// Configure logger
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.SetPrefix("goreilly: ")

	// In non-debug mode, we'll still log to file but not to stderr
	if !cfg.Debug {
		log.SetOutput(file)
	}

	log.Printf("Logging initialized. Debug mode: %v", cfg.Debug)
}

// ImportCookie loads a Netscape-format cookie file and stores the JWT token for future use.
func ImportCookie(cookieSrc string) error {
	// Currently supports only a direct file path; browser extraction can be added later.
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	authSvc, err := auth.NewService(cfg)
	if err != nil {
		return fmt.Errorf("init auth service: %w", err)
	}

	if _, err := authSvc.TokenFromCookieFile(cookieSrc); err != nil {
		return fmt.Errorf("import cookie: %w", err)
	}

	log.Printf("Cookie imported successfully from %s", cookieSrc)
	return nil
}
