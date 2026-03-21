package main

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// ==================== Data structures ====================

// CreateCartRequest matches OpenAPI spec: { "customer_id": int }
type CreateCartRequest struct {
	CustomerID int `json:"customer_id" binding:"required,min=1"`
}

// AddItemRequest matches OpenAPI spec: { "product_id": int, "quantity": int }
type AddItemRequest struct {
	ProductID int `json:"product_id" binding:"required,min=1"`
	Quantity  int `json:"quantity" binding:"required,min=1"`
}

// CartItem represents one item in a cart (used in GET response)
type CartItem struct {
	ProductID int `json:"product_id"`
	Quantity  int `json:"quantity"`
}

// CartResponse is the GET /shopping-carts/:id response
type CartResponse struct {
	ShoppingCartID int        `json:"shopping_cart_id"`
	CustomerID     int        `json:"customer_id"`
	Items          []CartItem `json:"items"`
}

// ==================== POST /shopping-carts ====================
// Creates a new cart, returns 201 + { "shopping_cart_id": int }

func createCartHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CreateCartRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "INVALID_INPUT",
				"message": "customer_id is required and must be a positive integer",
			})
			return
		}

		// Insert new cart row
		result, err := db.Exec("INSERT INTO carts (customer_id) VALUES (?)", req.CustomerID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "DB_ERROR",
				"message": "Failed to create shopping cart",
			})
			return
		}

		// Get the auto-generated cart ID
		cartID, err := result.LastInsertId()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "DB_ERROR",
				"message": "Failed to retrieve cart ID",
			})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"shopping_cart_id": cartID,
		})
	}
}

// ==================== GET /shopping-carts/:id ====================
// Returns cart + all items using a JOIN query

func getCartHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Parse cart ID from URL path
		cartID, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "INVALID_INPUT",
				"message": "Cart ID must be an integer",
			})
			return
		}

		// Step 1: Get cart info
		var cart CartResponse
		err = db.QueryRow("SELECT id, customer_id FROM carts WHERE id = ?", cartID).
			Scan(&cart.ShoppingCartID, &cart.CustomerID)

		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "NOT_FOUND",
				"message": "Shopping cart not found",
			})
			return
		}
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "DB_ERROR",
				"message": "Failed to retrieve cart",
			})
			return
		}

		// Step 2: Get all items in this cart
		// Uses the uk_cart_product index for fast lookup
		rows, err := db.Query(
			"SELECT product_id, quantity FROM cart_items WHERE cart_id = ?", cartID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "DB_ERROR",
				"message": "Failed to retrieve cart items",
			})
			return
		}
		defer rows.Close()

		cart.Items = []CartItem{} // empty slice instead of null in JSON
		for rows.Next() {
			var item CartItem
			if err := rows.Scan(&item.ProductID, &item.Quantity); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error":   "DB_ERROR",
					"message": "Failed to scan cart item",
				})
				return
			}
			cart.Items = append(cart.Items, item)
		}

		c.JSON(http.StatusOK, cart)
	}
}

// ==================== POST /shopping-carts/:id/items ====================
// Add or update item in cart. Returns 204 No Content (per OpenAPI spec).
// Uses INSERT ... ON DUPLICATE KEY UPDATE for upsert:
//   - If product not in cart yet → INSERT new row
//   - If product already in cart → UPDATE quantity (add to existing)

func addItemsHandler(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Parse cart ID from URL path
		cartID, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "INVALID_INPUT",
				"message": "Cart ID must be an integer",
			})
			return
		}

		var req AddItemRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "INVALID_INPUT",
				"message": "product_id and quantity are required and must be positive integers",
			})
			return
		}

		// Verify cart exists first
		var exists bool
		err = db.QueryRow("SELECT EXISTS(SELECT 1 FROM carts WHERE id = ?)", cartID).Scan(&exists)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "DB_ERROR",
				"message": "Failed to verify cart",
			})
			return
		}
		if !exists {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "NOT_FOUND",
				"message": "Shopping cart not found",
			})
			return
		}

		// Upsert: insert new item or update quantity if product already in cart
		// ON DUPLICATE KEY triggers on the UNIQUE(cart_id, product_id) constraint
		_, err = db.Exec(`
			INSERT INTO cart_items (cart_id, product_id, quantity)
			VALUES (?, ?, ?)
			ON DUPLICATE KEY UPDATE quantity = quantity + VALUES(quantity)
		`, cartID, req.ProductID, req.Quantity)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "DB_ERROR",
				"message": "Failed to add item to cart",
			})
			return
		}

		// 204 No Content per OpenAPI spec
		c.Status(http.StatusNoContent)
	}
}
