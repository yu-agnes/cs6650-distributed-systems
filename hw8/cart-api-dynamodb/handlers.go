package main

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
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

// CartItem represents one item in a cart
type CartItem struct {
	ProductID int `json:"product_id" dynamodbav:"product_id"`
	Quantity  int `json:"quantity" dynamodbav:"quantity"`
}

// CartRecord is the DynamoDB single-table record
// In MySQL we had 2 tables (carts + cart_items) joined by foreign key.
// In DynamoDB we store EVERYTHING in one record — items embedded as a list.
type CartRecord struct {
	CartID     string     `dynamodbav:"cart_id"`
	CustomerID int        `dynamodbav:"customer_id"`
	Items      []CartItem `dynamodbav:"items"`
	CreatedAt  string     `dynamodbav:"created_at"`
	UpdatedAt  string     `dynamodbav:"updated_at"`
}

// CartResponse is the JSON response for GET endpoint
type CartResponse struct {
	ShoppingCartID int        `json:"shopping_cart_id"`
	CustomerID     int        `json:"customer_id"`
	Items          []CartItem `json:"items"`
}

// ==================== ID generation ====================
// DynamoDB has no AUTO_INCREMENT like MySQL.
// We use timestamp-based IDs for simplicity.
// In production you'd use UUID or atomic counters.

var cartCounter int

func generateCartID() int {
	cartCounter++
	return int(time.Now().UnixNano()/1000000) + cartCounter
}

// ==================== POST /shopping-carts ====================
// MySQL version: INSERT INTO carts (customer_id) VALUES (?)
// DynamoDB version: PutItem — writes entire record at once

func createCartHandler(client *dynamodb.Client, tableName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CreateCartRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "INVALID_INPUT",
				"message": "customer_id is required and must be a positive integer",
			})
			return
		}

		cartID := generateCartID()
		now := time.Now().UTC().Format(time.RFC3339)

		// PutItem: write the full cart record in one call
		// No need for separate INSERT + foreign key like MySQL
		record := CartRecord{
			CartID:     strconv.Itoa(cartID),
			CustomerID: req.CustomerID,
			Items:      []CartItem{}, // empty cart
			CreatedAt:  now,
			UpdatedAt:  now,
		}

		item, err := attributevalue.MarshalMap(record)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "MARSHAL_ERROR",
				"message": "Failed to marshal cart data",
			})
			return
		}

		_, err = client.PutItem(context.TODO(), &dynamodb.PutItemInput{
			TableName: aws.String(tableName),
			Item:      item,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "DB_ERROR",
				"message": fmt.Sprintf("Failed to create cart: %v", err),
			})
			return
		}

		c.JSON(http.StatusCreated, gin.H{
			"shopping_cart_id": cartID,
		})
	}
}

// ==================== GET /shopping-carts/:id ====================
// MySQL version: SELECT from carts + SELECT from cart_items (or JOIN)
// DynamoDB version: GetItem — one call gets everything (items embedded)

func getCartHandler(client *dynamodb.Client, tableName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		cartID := c.Param("id")
		if _, err := strconv.Atoi(cartID); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "INVALID_INPUT",
				"message": "Cart ID must be an integer",
			})
			return
		}

		// GetItem: fetch by partition key — O(1) lookup, no table scan
		result, err := client.GetItem(context.TODO(), &dynamodb.GetItemInput{
			TableName: aws.String(tableName),
			Key: map[string]types.AttributeValue{
				"cart_id": &types.AttributeValueMemberS{Value: cartID},
			},
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "DB_ERROR",
				"message": fmt.Sprintf("Failed to get cart: %v", err),
			})
			return
		}

		// No item found
		if result.Item == nil {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "NOT_FOUND",
				"message": "Shopping cart not found",
			})
			return
		}

		// Unmarshal DynamoDB record back into Go struct
		var record CartRecord
		if err := attributevalue.UnmarshalMap(result.Item, &record); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "UNMARSHAL_ERROR",
				"message": "Failed to parse cart data",
			})
			return
		}

		cartIDInt, _ := strconv.Atoi(record.CartID)
		response := CartResponse{
			ShoppingCartID: cartIDInt,
			CustomerID:     record.CustomerID,
			Items:          record.Items,
		}
		if response.Items == nil {
			response.Items = []CartItem{}
		}

		c.JSON(http.StatusOK, response)
	}
}

// ==================== POST /shopping-carts/:id/items ====================
// MySQL version: INSERT ... ON DUPLICATE KEY UPDATE (upsert via unique constraint)
// DynamoDB version: GetItem → modify items list in Go → PutItem (read-modify-write)
//
// This is a key difference from MySQL:
// - MySQL handles upsert at the database level with ON DUPLICATE KEY
// - DynamoDB requires us to read the record, update it in application code, then write back

func addItemsHandler(client *dynamodb.Client, tableName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		cartID := c.Param("id")
		if _, err := strconv.Atoi(cartID); err != nil {
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

		// Step 1: Get current cart
		result, err := client.GetItem(context.TODO(), &dynamodb.GetItemInput{
			TableName: aws.String(tableName),
			Key: map[string]types.AttributeValue{
				"cart_id": &types.AttributeValueMemberS{Value: cartID},
			},
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "DB_ERROR",
				"message": "Failed to retrieve cart",
			})
			return
		}
		if result.Item == nil {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "NOT_FOUND",
				"message": "Shopping cart not found",
			})
			return
		}

		// Step 2: Unmarshal and update items in Go code
		var record CartRecord
		if err := attributevalue.UnmarshalMap(result.Item, &record); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "UNMARSHAL_ERROR",
				"message": "Failed to parse cart data",
			})
			return
		}

		// Upsert logic: same as MySQL's ON DUPLICATE KEY UPDATE
		// but done in application code instead of database
		found := false
		for i, item := range record.Items {
			if item.ProductID == req.ProductID {
				record.Items[i].Quantity += req.Quantity
				found = true
				break
			}
		}
		if !found {
			record.Items = append(record.Items, CartItem{
				ProductID: req.ProductID,
				Quantity:  req.Quantity,
			})
		}
		record.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

		// Step 3: Write updated record back
		item, err := attributevalue.MarshalMap(record)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "MARSHAL_ERROR",
				"message": "Failed to marshal cart data",
			})
			return
		}

		_, err = client.PutItem(context.TODO(), &dynamodb.PutItemInput{
			TableName: aws.String(tableName),
			Item:      item,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "DB_ERROR",
				"message": "Failed to update cart",
			})
			return
		}

		c.Status(http.StatusNoContent)
	}
}
