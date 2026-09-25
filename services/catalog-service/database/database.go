package database

import (
	"log"
	"os"
	"time"

	sqlite "github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// ConnectDatabase opens SQLite for catalog-service using pure-Go driver with WAL mode.
func ConnectDatabase() {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "/app/data/catalog.db"
	}

	logLevel := logger.Warn
	if os.Getenv("GIN_MODE") == "release" {
		logLevel = logger.Error
	}

	gormConfig := &gorm.Config{
		Logger:                                   logger.Default.LogMode(logLevel),
		PrepareStmt:                              false,
		DisableForeignKeyConstraintWhenMigrating: true,
	}

	db, err := gorm.Open(sqlite.Open(dbPath), gormConfig)
	if err != nil {
		log.Fatal("[Catalog-Service] Failed to connect database: ", err)
	}

	for _, pragma := range []string{
		"PRAGMA foreign_keys = ON",
		"PRAGMA busy_timeout = 5000",
		"PRAGMA journal_mode = WAL",
	} {
		_ = db.Exec(pragma).Error
	}

	if sqlDB, err := db.DB(); err == nil {
		sqlDB.SetMaxOpenConns(1)
		sqlDB.SetMaxIdleConns(1)
		sqlDB.SetConnMaxLifetime(time.Hour)
	}

	DB = db
	log.Printf("[Catalog-Service] Connected to SQLite database at %s", dbPath)
}
