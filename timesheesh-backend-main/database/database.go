package database

import (
	"fmt"
	"log"
	"os"

	"timesheesh-backend/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// InitDatabase initializes database connection
func InitDatabase() {
	var err error

	// Get database credentials from environment variables
	dbHost := os.Getenv("DB_HOST")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")
	dbPort := os.Getenv("DB_PORT")
	dbSSLMode := os.Getenv("DB_SSLMODE")

	// Set defaults if not provided
	if dbSSLMode == "" {
		dbSSLMode = "disable"
	}

	// Get timezone from environment variable, default to UTC
	dbTimeZone := os.Getenv("DB_TIMEZONE")
	if dbTimeZone == "" {
		dbTimeZone = "UTC"
	}

	// Validate required fields
	if dbUser == "" || dbPassword == "" || dbName == "" {
		log.Fatal("Database credentials are required. Please set DB_USER, DB_PASSWORD, and DB_NAME in .env file")
	}

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=%s",
		dbHost, dbUser, dbPassword, dbName, dbPort, dbSSLMode, dbTimeZone)

	// Debug: log connection details (without password)
	log.Printf("Connecting to database: host=%s, user=%s, dbname=%s, port=%s", dbHost, dbUser, dbName, dbPort)
	log.Printf("Password length: %d", len(dbPassword))

	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})

	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	log.Println("Database connected successfully")

	// Handle migration: Remove old client_id column if exists (we're using client_name and client_email now)
	var columnExists struct {
		Count int
	}
	DB.Raw(`
		SELECT COUNT(*) as count
		FROM information_schema.columns 
		WHERE table_name = 'projects' AND column_name = 'client_id'
	`).Scan(&columnExists)

	if columnExists.Count > 0 {
		log.Println("Found old client_id column. Removing it...")
		// Drop the client_id column
		if err := DB.Exec("ALTER TABLE projects DROP COLUMN IF EXISTS client_id").Error; err != nil {
			log.Printf("Warning: Could not drop client_id column: %v", err)
			log.Println("You may need to manually run: ALTER TABLE projects DROP COLUMN client_id")
		} else {
			log.Println("Successfully removed client_id column")
		}
	}

	// Auto migrate models
	err = DB.AutoMigrate(
		&models.User{},
		&models.Project{},
		&models.Contract{},       // Baru
		&models.ProjectMember{},  // Baru
		&models.Task{}, 		  // Baru
		&models.Timesheet{},      // Baru
		&models.ResourceRequest{},// Baru
		&models.AuditLog{},       // Baru
		&models.ContractPayment{}, 
	)
	if err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	log.Println("Database migration completed")

	// Run data seed
	Seed()
}
