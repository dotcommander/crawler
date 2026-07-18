package main

import (
	"log"

	"github.com/dotcommander/crawler/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		log.Fatalf("Error: %v", err)
	}
}
