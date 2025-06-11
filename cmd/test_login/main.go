package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/tuannvm/goreilly/internal/auth"
	"github.com/tuannvm/goreilly/internal/config"
)

func main() {
	// Parse command line flags
	username := flag.String("username", "", "O'Reilly username (email)")
	password := flag.String("password", "", "O'Reilly password")
	flag.Parse()

	// Validate required flags
	if *username == "" || *password == "" {
		flag.Usage()
		os.Exit(1)
	}

	// Initialize config
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize auth service
	authSvc, err := auth.NewService(cfg)
	if err != nil {
		log.Fatalf("Failed to create auth service: %v", err)
	}

	// Authenticate
	token, err := authSvc.Authenticate(context.Background(), *username, *password)
	if err != nil {
		log.Fatalf("Authentication failed: %v", err)
	}

	// Print success message
	fmt.Println("Successfully authenticated with O'Reilly!")
	fmt.Printf("Token: %s...\n", token.AccessToken[:20])
	fmt.Printf("Expires at: %s\n", token.ExpiresAt.Format("2006-01-02 15:04:05"))
}
