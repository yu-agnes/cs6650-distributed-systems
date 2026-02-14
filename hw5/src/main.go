package main

import (
	"net/http"
	"strconv"
	"sync"

	"github.com/gin-gonic/gin"
)

// ==================== Data Models ====================

// Product represents the product schema from api.yaml
type Product struct {
	ProductID    int    `json:"product_id"`
	SKU          string `json:"sku"`
	Manufacturer string `json:"manufacturer"`
	CategoryID   int    `json:"category_id"`
	Weight       int    `json:"weight"`
	SomeOtherID  int    `json:"some_other_id"`
}

// ErrorResponse represents the error schema from api.yaml
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// ==================== In-Memory Storage ====================

// Thread-safe storage using sync.RWMutex + map
var (
	products = make(map[int]*Product)
	mu       sync.RWMutex
)

// ==================== Validation ====================

// validateProduct validates the product data according to api.yaml schema
func validateProduct(p *Product, pathProductID int) (bool, string) {
	// Check product_id matches path parameter
	if p.ProductID != pathProductID {
		return false, "product_id in body must match productId in path"
	}

	// Check product_id >= 1
	if p.ProductID < 1 {
		return false, "product_id must be at least 1"
	}

	// Check sku: minLength 1, maxLength 100
	if len(p.SKU) < 1 || len(p.SKU) > 100 {
		return false, "sku must be between 1 and 100 characters"
	}

	// Check manufacturer: minLength 1, maxLength 200
	if len(p.Manufacturer) < 1 || len(p.Manufacturer) > 200 {
		return false, "manufacturer must be between 1 and 200 characters"
	}

	// Check category_id >= 1
	if p.CategoryID < 1 {
		return false, "category_id must be at least 1"
	}

	// Check weight >= 0
	if p.Weight < 0 {
		return false, "weight must be at least 0"
	}

	// Check some_other_id >= 1
	if p.SomeOtherID < 1 {
		return false, "some_other_id must be at least 1"
	}

	return true, ""
}

// ==================== HTTP Handlers ====================

// getProduct handles GET /products/:id
// Response codes: 200 (found), 400 (invalid id), 404 (not found)
func getProduct(c *gin.Context) {
	// Parse and validate product ID from path
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id < 1 {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "INVALID_INPUT",
			Message: "Invalid product ID",
			Details: "productId must be a positive integer",
		})
		return
	}

	// Look up product in storage
	mu.RLock()
	product, exists := products[id]
	mu.RUnlock()

	if !exists {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "NOT_FOUND",
			Message: "Product not found",
			Details: "No product exists with ID " + strconv.Itoa(id),
		})
		return
	}

	// Return product with 200 OK
	c.JSON(http.StatusOK, product)
}

// addProductDetails handles POST /products/:id/details
// Response codes: 204 (success), 400 (invalid input)
func addProductDetails(c *gin.Context) {
	// Parse and validate product ID from path
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil || id < 1 {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "INVALID_INPUT",
			Message: "Invalid product ID",
			Details: "productId must be a positive integer",
		})
		return
	}

	// Parse JSON body
	var product Product
	if err := c.ShouldBindJSON(&product); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "INVALID_INPUT",
			Message: "Invalid JSON body",
			Details: err.Error(),
		})
		return
	}

	// Validate product data according to api.yaml schema
	if valid, errMsg := validateProduct(&product, id); !valid {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "INVALID_INPUT",
			Message: "Validation failed",
			Details: errMsg,
		})
		return
	}

	// Store product
	mu.Lock()
	products[id] = &product
	mu.Unlock()

	// Return 204 No Content (success, no body)
	c.Status(http.StatusNoContent)
}

// healthCheck handles GET /health
func healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "healthy"})
}

// ==================== Main ====================

func main() {
	// Create Gin router with default middleware (logger, recovery)
	router := gin.Default()

	// Register routes
	router.GET("/products/:id", getProduct)
	router.POST("/products/:id/details", addProductDetails)
	router.GET("/health", healthCheck)

	// Start server on port 8080
	router.Run("0.0.0.0:8080")
}
