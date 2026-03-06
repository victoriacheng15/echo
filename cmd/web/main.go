package main

import (
	"log"

	"echo/internal/web"
)

func main() {
	if err := web.Generate("internal/web", "dist"); err != nil {
		log.Fatalf("Error: %v", err)
	}
}
