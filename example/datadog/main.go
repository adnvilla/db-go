package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"

	"github.com/DataDog/dd-trace-go/v2/ddtrace/tracer"
	dbgo "github.com/adnvilla/db-go"
	"github.com/joho/godotenv"
)

type User struct {
	ID   uint
	Name string
}

func main() {
	// Load .env file from project root
	rootDir, err := findProjectRoot()
	if err != nil {
		log.Printf("Warning: couldn't find project root: %v", err)
	} else {
		envPath := filepath.Join(rootDir, ".env")
		err := godotenv.Load(envPath)
		if err != nil {
			log.Printf("Warning: Error loading .env file: %v", err)
		} else {
			log.Printf("Loaded environment from %s", envPath)
		}
	}

	// Start the Datadog tracer with environment variables
	serviceName := getEnv("DD_SERVICE_NAME", "example-service")
	environment := getEnv("DD_ENV", "development")

	tracer.Start(
		tracer.WithService(serviceName),
		tracer.WithEnv(environment),
	)
	defer tracer.Stop()

	// Create database connection string from environment variables
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "postgres")
	dbPassword := getEnv("DB_PASSWORD", "postgres")
	dbName := getEnv("DB_NAME", "example")

	dsn := fmt.Sprintf("postgresql://%s:%s@%s:%s/%s?sslmode=disable",
		dbUser, dbPassword, dbHost, dbPort, dbName)

	// Create configuration with Datadog tracing enabled
	config := dbgo.Config{
		PrimaryDSN: dsn,
	}

	// Enable tracing with custom options
	config = *dbgo.WithTracing(&config)
	config = *dbgo.WithTracingServiceName(getEnv("DD_SERVICE_NAME", "db-example"))(&config)

	analyticsRate, err := strconv.ParseFloat(getEnv("DD_ANALYTICS_RATE", "1.0"), 64)
	if err != nil {
		analyticsRate = 1.0
	}
	config = *dbgo.WithTracingAnalyticsRate(analyticsRate)(&config)
	config = *dbgo.WithTracingErrorCheck(func(err error) bool {
		// Custom error handling logic
		return err != nil
	})(&config)

	// Get database connection
	dbConn := dbgo.GetConnection(config)
	if dbConn.Error != nil {
		log.Fatalf("Failed to connect to database: %v", dbConn.Error)
	}

	// Create table if not exists
	migrateErr := dbConn.Instance.AutoMigrate(&User{})
	if migrateErr != nil {
		log.Fatalf("Failed to migrate: %v", migrateErr)
	}

	// Start a parent span
	span, ctx := tracer.StartSpanFromContext(context.Background(), "database-operations",
		tracer.ServiceName("db-example"),
		tracer.ResourceName("user-operations"),
	)
	defer span.Finish()

	// Use the context with the span for database operations
	db := dbgo.WithContext(ctx, dbConn.Instance)

	// Create a user
	user := User{Name: "John Doe"}
	result := db.Create(&user)
	if result.Error != nil {
		log.Fatalf("Failed to create user: %v", result.Error)
	}

	// Query the user
	var retrievedUser User
	result = db.First(&retrievedUser, user.ID)
	if result.Error != nil {
		log.Fatalf("Failed to retrieve user: %v", result.Error)
	}

	fmt.Printf("Retrieved user: %v\n", retrievedUser)
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}
	return value
}

// findProjectRoot tries to find the root directory of the project
func findProjectRoot() (string, error) {
	// Start from the current working directory
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Go up the directory tree until we find a marker file like .env or go.mod
	for {
		// Check if .env or go.mod exists in this directory
		if _, err := os.Stat(filepath.Join(dir, ".env")); err == nil {
			return dir, nil
		}
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}

		// Go up one directory
		parentDir := filepath.Dir(dir)
		if parentDir == dir {
			// We've reached the root directory without finding any marker
			return "", fmt.Errorf("couldn't find project root")
		}
		dir = parentDir
	}
}
