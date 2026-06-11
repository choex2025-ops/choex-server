package database

import (
	"context"
	"log"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/choex2025-ops/choex-server/internal/config"
	"github.com/redis/go-redis/v9"
)

var (
	DB    *gorm.DB
	Redis *redis.Client
)

func Init(cfg *config.Config) {
	var err error
	DB, err = gorm.Open(mysql.Open(cfg.DSN()), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatalf("Failed to connect to MySQL: %v", err)
	}

	Redis = redis.NewClient(&redis.Options{
		Addr: cfg.RedisAddr(),
	})
	if err := Redis.Ping(context.Background()).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	log.Println("Database and Redis connected")
}
