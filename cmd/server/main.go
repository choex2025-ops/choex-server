package main

import (
	"log"

	"github.com/choex2025-ops/choex-server/internal/config"
)

func main() {
	cfg := config.Load()
	log.Printf("Starting server on :%s", cfg.ServerPort)
}
