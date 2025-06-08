package main

import (
	"log"
	"os"

	"github.com/tuannvm/goreilly/internal/app"
)

func main() {
	if err := app.Run(); err != nil {
		log.Fatalf("Error: %v", err)
		os.Exit(1)
	}
}
