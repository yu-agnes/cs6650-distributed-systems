package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// GetHealth handles GET /health
func GetHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
