package main

import (
	"log"
	"os"

	"github.com/shuaibizhang/codecoverage/internal/commands"
)

func main() {
	err := commands.Execute()
	if err != nil {
		log.Fatalf("Error: %v", err)
		os.Exit(1)
	}
}
