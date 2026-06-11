package main

import (
	"log"

	"github.com/choex2025-ops/choex-server/internal/config"
	"github.com/choex2025-ops/choex-server/internal/database"
	"github.com/choex2025-ops/choex-server/internal/model"
	"github.com/choex2025-ops/choex-server/internal/router"
)

func main() {
	cfg := config.Load()

	database.Init(cfg)
	if err := database.DB.AutoMigrate(
		&model.User{},
		&model.Event{},
		&model.Bill{},
		&model.Password{},
		&model.AgentMemory{},
		&model.MemoryVersion{},
	); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	r := router.Setup(cfg)
	log.Printf("Server running on :%s", cfg.ServerPort)
	if err := r.Run(":" + cfg.ServerPort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
