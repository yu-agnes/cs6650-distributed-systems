package main

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// Product represents a product in our store
type Product struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Category    string `json:"category"`
	Description string `json:"description"`
	Brand       string `json:"brand"`
}

// SearchResponse is the JSON response for search queries
type SearchResponse struct {
	Products   []Product `json:"products"`
	TotalFound int       `json:"total_found"`
	SearchTime string    `json:"search_time"`
}

var (
	productStore  sync.Map
	totalProducts = 100000
)

// Sample data arrays for generating products
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

// generateProducts creates 100,000 products and stores them in sync.Map
func generateProducts() {
	log.Println("Generating products...")
	start := time.Now()

	for i := 1; i <= totalProducts; i++ {
		brand := brands[i%len(brands)]
		category := categories[i%len(categories)]
		description := descriptions[i%len(descriptions)]

		p := Product{
			ID:          i,
			Name:        fmt.Sprintf("Product %s %d", brand, i),
			Category:    category,
			Description: description,
			Brand:       brand,
		}
		productStore.Store(i, p)
	}

	log.Printf("Generated %d products in %v\n", totalProducts, time.Since(start))
}

// searchProducts searches through exactly 100 products and returns matches
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

func main() {
	// Generate products at startup
	generateProducts()

	r := gin.Default()

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":   "healthy",
			"products": totalProducts,
		})
	})

	// Search endpoint
	r.GET("/products/search", func(c *gin.Context) {
		query := c.Query("q")
		if query == "" {
			c.JSON(400, gin.H{"error": "query parameter 'q' is required"})
			return
		}

		start := time.Now()
		products, totalFound := searchProducts(query)
		searchTime := time.Since(start)

		c.JSON(200, SearchResponse{
			Products:   products,
			TotalFound: totalFound,
			SearchTime: searchTime.String(),
		})
	})

	log.Println("Server starting on port 8080...")
	r.Run(":8080")
}
