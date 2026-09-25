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

// ConnectDatabase opens SQLite for community-care-service using pure-Go driver with WAL mode.
func ConnectDatabase() {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "/app/data/community.db"
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
		log.Fatal("[Community-Care-Service] Failed to connect database: ", err)
	}

	for _, pragma := range []string{
		"PRAGMA foreign_keys = OFF",
		"PRAGMA busy_timeout = 5000",
		"PRAGMA journal_mode = WAL",
	} {
		_ = db.Exec(pragma).Error
	}

	// Attach auth.db for author / member user resolution
	authDbPath := os.Getenv("AUTH_DB_PATH")
	if authDbPath == "" {
		authDbPath = "/app/data/auth.db"
	}
	if _, err := os.Stat(authDbPath); err == nil {
		if err := db.Exec("ATTACH DATABASE '" + authDbPath + "' AS auth_db").Error; err != nil {
			log.Printf("[Community-Care-Service] Note on attaching auth.db: %v", err)
		} else {
			log.Printf("[Community-Care-Service] Attached auth database: %s", authDbPath)
		}
	}

	if sqlDB, err := db.DB(); err == nil {
		sqlDB.SetMaxOpenConns(1)
		sqlDB.SetMaxIdleConns(1)
		sqlDB.SetConnMaxLifetime(time.Hour)
	}

	DB = db
	log.Printf("[Community-Care-Service] Connected to SQLite database at %s", dbPath)
}
