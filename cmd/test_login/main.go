package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/tuannvm/goreilly/internal/auth"
	"github.com/tuannvm/goreilly/internal/config"
)

func main() {
	// Parse command line flags
	username := flag.String("username", "", "O'Reilly username (email)")
	password := flag.String("password", "", "O'Reilly password")
	jwt := flag.String("jwt", "", "O'Reilly orm-jwt token (if you want to skip login)")
	flag.Parse()

	if *jwt != "" {
		// Use the JWT directly for an authenticated request (show /api/v2/me/)
		url := "https://learning.oreilly.com/api/v2/me/"
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			log.Fatalf("Failed to create request: %v", err)
		}
		req.Header.Set("Authorization", "Bearer "+*jwt)
		req.AddCookie(&http.Cookie{
			Name:   "orm-jwt",
			Value:  *jwt,
			Domain: ".oreilly.com",
			Path:   "/",
		})

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("Response status: %s\nBody:\n%s\n", resp.Status, string(body))
		return
	}

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
