package database

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"wa-proxy/internal/config"
	"wa-proxy/internal/model"
)

// EnsureDatabase connects to the default "postgres" maintenance database and
// creates the target application database if it does not already exist. This
// keeps self-hosted setup to a single step (no manual createdb).
func EnsureDatabase(cfg *config.Config) error {
	db, err := sql.Open("pgx", cfg.AdminDSN())
	if err != nil {
		return fmt.Errorf("open admin db: %w", err)
	}
	defer db.Close()

	var attempts int
	for {
		if err = db.Ping(); err == nil {
			break
		}
		attempts++
		if attempts >= 10 {
			return fmt.Errorf("ping admin db: %w", err)
		}
		time.Sleep(2 * time.Second)
	}

	var exists bool
	err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)", cfg.DBName).Scan(&exists)
	if err != nil {
		return fmt.Errorf("check database: %w", err)
	}
	if !exists {
		// Identifier can't be parameterized; DBName comes from trusted config.
		if _, err := db.Exec(fmt.Sprintf(`CREATE DATABASE "%s"`, cfg.DBName)); err != nil {
			return fmt.Errorf("create database %q: %w", cfg.DBName, err)
		}
		log.Printf("database: created %q", cfg.DBName)
	}
	return nil
}

// Connect opens a GORM connection to PostgreSQL with a basic retry loop so the
// app can start alongside a freshly-booted postgres container.
func Connect(cfg *config.Config) (*gorm.DB, error) {
	logLevel := gormlogger.Warn
	if cfg.IsProduction() {
		logLevel = gormlogger.Error
	}

	var db *gorm.DB
	var err error
	for attempt := 1; attempt <= 10; attempt++ {
		db, err = gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{
			Logger: gormlogger.Default.LogMode(logLevel),
		})
		if err == nil {
			break
		}
		log.Printf("database: connection attempt %d failed: %v", attempt, err)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		return nil, fmt.Errorf("connect database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql.DB: %w", err)
	}
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(time.Hour)

	return db, nil
}

// Migrate runs GORM auto-migration for all application models.
func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(model.AllModels()...); err != nil {
		return fmt.Errorf("auto migrate: %w", err)
	}
	return nil
}
