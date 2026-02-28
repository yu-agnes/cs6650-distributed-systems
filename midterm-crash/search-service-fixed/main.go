package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
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
	Inventory     *InventoryInfo `json:"inventory,omitempty"`
	InventoryNote string         `json:"inventory_note,omitempty"`
}

type SearchResponse struct {
	Products      []ProductWithInventory `json:"products"`
	TotalFound    int                    `json:"total_found"`
	SearchTime    string                 `json:"search_time"`
	CircuitState  string                 `json:"circuit_state"`
	BulkheadInUse int32                  `json:"bulkhead_in_use"`
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

// ==================== FIX 1: Circuit Breaker ====================

type CircuitState int

const (
	StateClosed   CircuitState = iota // Normal: allow requests
	StateOpen                         // Tripped: reject requests immediately
	StateHalfOpen                     // Testing: allow one request to test recovery
)

func (s CircuitState) String() string {
	switch s {
	case StateClosed:
		return "CLOSED"
	case StateOpen:
		return "OPEN"
	case StateHalfOpen:
		return "HALF-OPEN"
	default:
		return "UNKNOWN"
	}
}

type CircuitBreaker struct {
	mu               sync.Mutex
	state            CircuitState
	failureCount     int
	successCount     int
	failureThreshold int           // failures before opening
	successThreshold int           // successes in half-open before closing
	openTimeout      time.Duration // how long to wait before half-open
	lastFailureTime  time.Time
}

func NewCircuitBreaker(failureThreshold, successThreshold int, openTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		state:            StateClosed,
		failureThreshold: failureThreshold,
		successThreshold: successThreshold,
		openTimeout:      openTimeout,
	}
}

func (cb *CircuitBreaker) AllowRequest() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		// Check if enough time has passed to try half-open
		if time.Since(cb.lastFailureTime) > cb.openTimeout {
			cb.state = StateHalfOpen
			cb.successCount = 0
			log.Println("[CIRCUIT BREAKER] State: OPEN -> HALF-OPEN (testing recovery)")
			return true
		}
		return false
	case StateHalfOpen:
		return true
	}
	return false
}

func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateHalfOpen:
		cb.successCount++
		if cb.successCount >= cb.successThreshold {
			cb.state = StateClosed
			cb.failureCount = 0
			log.Println("[CIRCUIT BREAKER] State: HALF-OPEN -> CLOSED (service recovered!)")
		}
	case StateClosed:
		cb.failureCount = 0 // reset on success
	}
}

func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.lastFailureTime = time.Now()

	switch cb.state {
	case StateClosed:
		cb.failureCount++
		if cb.failureCount >= cb.failureThreshold {
			cb.state = StateOpen
			log.Printf("[CIRCUIT BREAKER] State: CLOSED -> OPEN (failed %d times)\n", cb.failureCount)
		}
	case StateHalfOpen:
		cb.state = StateOpen
		log.Println("[CIRCUIT BREAKER] State: HALF-OPEN -> OPEN (still failing)")
	}
}

func (cb *CircuitBreaker) GetState() CircuitState {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state
}

// ==================== FIX 2: Bulkhead ====================
// Limit concurrent inventory calls to prevent resource exhaustion

var bulkheadSem = make(chan struct{}, 5) // max 5 concurrent inventory calls
var bulkheadInUse int32                  // atomic counter for monitoring

// ==================== FIX 3: Fail Fast (Timeout) ====================
// HTTP client with strict timeout

var httpClient = &http.Client{
	Timeout: 300 * time.Millisecond, // 300ms timeout - fail fast!
}

// ==================== Protected Inventory Call ====================

var circuitBreaker = NewCircuitBreaker(
	3,              // open after 3 consecutive failures
	2,              // close after 2 successes in half-open
	10*time.Second, // wait 10s before testing recovery
)

func callInventoryProtected(productID int) (*InventoryInfo, string) {
	// CHECK 1: Circuit Breaker - is the service known to be down?
	if !circuitBreaker.AllowRequest() {
		return nil, "circuit breaker open - skipped"
	}

	// CHECK 2: Bulkhead - is the inventory pool full?
	select {
	case bulkheadSem <- struct{}{}:
		// Got a slot in the bulkhead pool
		atomic.AddInt32(&bulkheadInUse, 1)
		defer func() {
			<-bulkheadSem
			atomic.AddInt32(&bulkheadInUse, -1)
		}()
	default:
		// Pool is full — degrade immediately
		return nil, "bulkhead full - skipped"
	}

	// CHECK 3: Fail Fast - call with strict timeout
	url := fmt.Sprintf("http://localhost:8081/inventory/%d", productID)
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	resp, err := httpClient.Do(req)
	if err != nil {
		circuitBreaker.RecordFailure()
		return nil, "timeout or error - failed fast"
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		circuitBreaker.RecordFailure()
		return nil, fmt.Sprintf("inventory returned %d", resp.StatusCode)
	}

	// Success!
	circuitBreaker.RecordSuccess()
	body, _ := io.ReadAll(resp.Body)
	var info InventoryInfo
	json.Unmarshal(body, &info)
	return &info, ""
}

// ==================== Main ====================

func main() {
	generateProducts()

	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":        "healthy",
			"products":      totalProducts,
			"version":       "fixed - with circuit breaker + bulkhead + fail fast",
			"circuit_state": circuitBreaker.GetState().String(),
			"bulkhead_use":  atomic.LoadInt32(&bulkheadInUse),
		})
	})

	// Expose circuit breaker status for monitoring
	r.GET("/circuit-status", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"circuit_state":   circuitBreaker.GetState().String(),
			"bulkhead_in_use": atomic.LoadInt32(&bulkheadInUse),
			"bulkhead_max":    5,
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

		var results []ProductWithInventory
		for _, p := range products {
			inv, note := callInventoryProtected(p.ID)
			results = append(results, ProductWithInventory{
				Product:       p,
				Inventory:     inv,
				InventoryNote: note,
			})
		}

		c.JSON(200, SearchResponse{
			Products:      results,
			TotalFound:    totalFound,
			SearchTime:    time.Since(start).String(),
			CircuitState:  circuitBreaker.GetState().String(),
			BulkheadInUse: atomic.LoadInt32(&bulkheadInUse),
		})
	})

	log.Println("[FIXED VERSION] Search Service starting on port 8080...")
	log.Println("Protected with: Circuit Breaker (threshold=3, timeout=10s) + Bulkhead (max=5) + Fail Fast (300ms)")
	r.Run(":8080")
}
