package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/tuannvm/goreilly/internal/app"
)

func main() {
	// Simple search command: goreilly search &lt;query&gt;
	if len(os.Args) > 2 && os.Args[1] == "search" {
		query := strings.Join(os.Args[2:], " ")
		fmt.Printf("Searching for books matching %q ... (feature not fully implemented)\n", query)
		// TODO: integrate with O'Reilly API to fetch and display results.
		return
	}

	// Support manual cookie injection:
	//   goreilly cookie import <cookie-file|browser>
	if len(os.Args) > 3 && os.Args[1] == "cookie" && os.Args[2] == "import" {
		cookieSrc := os.Args[3]
		if err := app.ImportCookie(cookieSrc); err != nil {
			log.Fatalf("Cookie import failed: %v", err)
		}
		fmt.Println("Cookies imported successfully. You can now run `goreilly` normally.")
		return
	}

	if err := app.Run(); err != nil {
		log.Fatalf("Error: %v", err)
		os.Exit(1)
	}
}
