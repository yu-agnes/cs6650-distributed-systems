package handlers

import (
	"net/http"

	"chaosarena/internal/models"
	"chaosarena/internal/store"

	"github.com/gin-gonic/gin"
)

type AlbumHandler struct {
	db *store.DynamoStore
}

func NewAlbumHandler(db *store.DynamoStore) *AlbumHandler {
	return &AlbumHandler{db: db}
}

// PutAlbum handles PUT /albums/:album_id
// Idempotent: creates if new, updates if existing.
func (h *AlbumHandler) PutAlbum(c *gin.Context) {
	albumID := c.Param("album_id")

	var req models.Album
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	// Use the album_id from the URL path (authoritative)
	req.AlbumID = albumID

	isNew, err := h.db.PutAlbum(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to store album"})
		return
	}

	// Return 201 for new, 200 for update
	status := http.StatusOK
	if isNew {
		status = http.StatusCreated
	}

	c.JSON(status, models.Album{
		AlbumID:     req.AlbumID,
		Title:       req.Title,
		Description: req.Description,
		Owner:       req.Owner,
	})
}

// GetAlbum handles GET /albums/:album_id
func (h *AlbumHandler) GetAlbum(c *gin.Context) {
	albumID := c.Param("album_id")

	album, err := h.db.GetAlbum(c.Request.Context(), albumID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get album"})
		return
	}
	if album == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	c.JSON(http.StatusOK, album)
}

// ListAlbums handles GET /albums
// Returns a bare array of all albums.
func (h *AlbumHandler) ListAlbums(c *gin.Context) {
	albums, err := h.db.ListAlbums(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list albums"})
		return
	}

	if albums == nil {
		albums = []models.Album{}
	}

	c.JSON(http.StatusOK, albums)
}
