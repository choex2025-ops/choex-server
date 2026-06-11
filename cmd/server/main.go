package main

import (
	"log"

	"github.com/choex2025-ops/choex-server/internal/config"
	"github.com/choex2025-ops/choex-server/internal/database"
	"github.com/choex2025-ops/choex-server/internal/model"
)

func main() {
	cfg := config.Load()

	database.Init(cfg)
	if err := database.DB.AutoMigrate(&model.User{}); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	log.Printf("Server ready on :%s", cfg.ServerPort)
}
