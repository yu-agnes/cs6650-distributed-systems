package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// ==================== Product Data (same as HW6) ====================

type Product struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Category    string `json:"category"`
	Description string `json:"description"`
	Brand       string `json:"brand"`
}

type InventoryInfo struct {
	InStock  bool `json:"in_stock"`
	Quantity int  `json:"quantity"`
}

type ProductWithInventory struct {
	Product
	Inventory *InventoryInfo `json:"inventory,omitempty"`
}

type SearchResponse struct {
	Products   []ProductWithInventory `json:"products"`
	TotalFound int                    `json:"total_found"`
	SearchTime string                 `json:"search_time"`
}

var (
	productStore  sync.Map
	totalProducts = 100000
)

var (
	brands       = []string{"Alpha", "Beta", "Gamma", "Delta", "Epsilon", "Zeta", "Eta", "Theta", "Iota", "Kappa"}
	categories   = []string{"Electronics", "Books", "Home", "Garden", "Sports", "Toys", "Clothing", "Food", "Health", "Automotive"}
	descriptions = []string{
		"A high-quality product for everyday use",
		"Premium grade item with excellent durability",
		"Best seller in its category",
		"Affordable and reliable choice",
		"Top-rated by customers worldwide",
	}
)

func generateProducts() {
	log.Println("Generating products...")
	start := time.Now()
	for i := 1; i <= totalProducts; i++ {
		p := Product{
			ID:          i,
			Name:        fmt.Sprintf("Product %s %d", brands[i%len(brands)], i),
			Category:    categories[i%len(categories)],
			Description: descriptions[i%len(descriptions)],
			Brand:       brands[i%len(brands)],
		}
		productStore.Store(i, p)
	}
	log.Printf("Generated %d products in %v\n", totalProducts, time.Since(start))
}

func searchProducts(query string) ([]Product, int) {
	query = strings.ToLower(query)
	var results []Product
	totalFound := 0
	checked := 0
	for i := 1; i <= totalProducts; i++ {
		if checked >= 100 {
			break
		}
		val, ok := productStore.Load(i)
		if !ok {
			continue
		}
		p := val.(Product)
		checked++
		if strings.Contains(strings.ToLower(p.Name), query) ||
			strings.Contains(strings.ToLower(p.Category), query) {
			totalFound++
			if len(results) < 20 {
				results = append(results, p)
			}
		}
	}
	return results, totalFound
}

// ==================== THE PROBLEM: No protection ====================

// callInventory calls inventory service with NO timeout, NO circuit breaker
// If inventory service is slow, this goroutine blocks for the full duration
func callInventory(productID int) *InventoryInfo {
	url := fmt.Sprintf("http://localhost:8081/inventory/%d", productID)

	// NO TIMEOUT - this is the problem!
	// Uses http.DefaultClient which has no timeout set
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Inventory call failed for product %d: %v", productID, err)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil
	}

	body, _ := io.ReadAll(resp.Body)
	var info InventoryInfo
	json.Unmarshal(body, &info)
	return &info
}

func main() {
	generateProducts()

	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":   "healthy",
			"products": totalProducts,
			"version":  "broken - no circuit breaker",
		})
	})

	r.GET("/products/search", func(c *gin.Context) {
		query := c.Query("q")
		if query == "" {
			c.JSON(400, gin.H{"error": "query parameter 'q' is required"})
			return
		}

		start := time.Now()
		products, totalFound := searchProducts(query)

		// For each product found, call inventory service (NO PROTECTION)
		// If inventory is slow, EVERY request gets slow
		var results []ProductWithInventory
		for _, p := range products {
			inv := callInventory(p.ID)
			results = append(results, ProductWithInventory{
				Product:   p,
				Inventory: inv,
			})
		}

		c.JSON(200, SearchResponse{
			Products:   results,
			TotalFound: totalFound,
			SearchTime: time.Since(start).String(),
		})
	})

	log.Println("[BROKEN VERSION] Search Service starting on port 8080...")
	log.Println("WARNING: No timeout, no circuit breaker, no bulkhead!")
	r.Run(":8080")
}
