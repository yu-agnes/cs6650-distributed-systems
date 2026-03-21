package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
)

func main() {
	// ==================== Read DB config from environment ====================
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "3306")
	dbUser := getEnv("DB_USER", "admin")
	dbPassword := getEnv("DB_PASSWORD", "password")
	dbName := getEnv("DB_NAME", "shopping")

	// ==================== Connect to MySQL with connection pool ====================
	// DSN format: user:password@tcp(host:port)/dbname?params
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		dbUser, dbPassword, dbHost, dbPort, dbName)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Connection pool settings (like HikariCP in Spring Boot)
	db.SetMaxOpenConns(25)                 // max simultaneous connections
	db.SetMaxIdleConns(10)                 // idle connections kept alive
	db.SetConnMaxLifetime(5 * time.Minute) // recycle connections every 5 min

	// Wait for DB to be ready (RDS might take a moment)
	if err := waitForDB(db, 30); err != nil {
		log.Fatalf("Database not reachable: %v", err)
	}
	log.Println("Connected to MySQL successfully")

	// ==================== Run schema migration ====================
	if err := createTables(db); err != nil {
		log.Fatalf("Failed to create tables: %v", err)
	}
	log.Println("Database tables ready")

	// ==================== Set up Gin router ====================
	router := gin.Default()

	// Health check (ALB uses this)
	router.GET("/health", func(c *gin.Context) {
		if err := db.Ping(); err != nil {
			c.JSON(500, gin.H{"status": "unhealthy", "error": err.Error()})
			return
		}
		c.JSON(200, gin.H{"status": "healthy"})
	})

	// Shopping Cart API endpoints (matching OpenAPI spec)
	router.POST("/shopping-carts", createCartHandler(db))
	router.GET("/shopping-carts/:id", getCartHandler(db))
	router.POST("/shopping-carts/:id/items", addItemsHandler(db))

	// Start server on port 8080
	log.Println("Starting cart-api server on :8080")
	if err := router.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// waitForDB retries db.Ping until success or timeout
func waitForDB(db *sql.DB, maxRetries int) error {
	for i := 0; i < maxRetries; i++ {
		if err := db.Ping(); err == nil {
			return nil
		}
		log.Printf("Waiting for database... attempt %d/%d", i+1, maxRetries)
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("database not ready after %d attempts", maxRetries)
}

// getEnv reads env variable with a fallback default
func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
