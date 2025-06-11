package main

import (
	"fmt"
	"log"
	"os"

	"github.com/tuannvm/goreilly/internal/app"
)

func main() {
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
