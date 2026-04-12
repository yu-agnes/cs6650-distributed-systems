package handlers

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"

	"chaosarena/internal/models"
	"chaosarena/internal/store"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const smallFileThreshold = 5 * 1024 * 1024 // 5MB

type PhotoHandler struct {
	db  *store.DynamoStore
	s3  *store.S3Store
	sqs *store.SQSStore
}

func NewPhotoHandler(db *store.DynamoStore, s3 *store.S3Store, sqs *store.SQSStore) *PhotoHandler {
	return &PhotoHandler{db: db, s3: s3, sqs: sqs}
}

// albumResult carries the result of an async GetAlbum call.
type albumResult struct {
	album *models.Album
	err   error
}

// UploadPhoto handles POST /albums/:album_id/photos
// Small files (≤5MB): return 202 immediately, upload S3 + mark completed asynchronously (no SQS).
// Large files (>5MB): upload S3 synchronously, send SQS, then return 202.
func (h *PhotoHandler) UploadPhoto(c *gin.Context) {
	albumID := c.Param("album_id")
	ctx := c.Request.Context()

	// Start GetAlbum and multipart body reading in parallel.
	albumCh := make(chan albumResult, 1)
	go func() {
		a, err := h.db.GetAlbum(ctx, albumID)
		albumCh <- albumResult{a, err}
	}()

	// Stream the multipart body — find the "photo" part (runs concurrently with GetAlbum)
	mr, err := c.Request.MultipartReader()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing or malformed multipart form"})
		return
	}
	var photoPart io.Reader
	for {
		part, partErr := mr.NextPart()
		if partErr == io.EOF {
			break
		}
		if partErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "malformed multipart form"})
			return
		}
		if part.FormName() == "photo" {
			photoPart = part
			break
		}
	}
	if photoPart == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing photo field"})
		return
	}

	// Peek up to 5MB+1 to determine file size (also runs while GetAlbum may still be in flight)
	peek := make([]byte, smallFileThreshold+1)
	n, readErr := io.ReadFull(photoPart, peek)
	isSmall := readErr == io.ErrUnexpectedEOF || readErr == io.EOF
	if !isSmall && readErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read photo"})
		return
	}

	// Wait for album check result
	ar := <-albumCh
	if ar.err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check album"})
		return
	}
	if ar.album == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	// Assign seq and generate IDs
	seq, err := h.db.IncrementPhotoSeq(ctx, albumID)
	if err != nil {
		log.Printf("ERROR: increment seq for album %s: %v", albumID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to assign sequence number"})
		return
	}
	photoID := uuid.New().String()
	finalKey := fmt.Sprintf("albums/%s/photos/%s", albumID, photoID)

	if isSmall {
		// Small file: create DB record, return 202 immediately, upload S3 + mark completed in background.
		// No SQS involved — worker is bypassed entirely for small files.
		buf := make([]byte, n)
		copy(buf, peek[:n])

		photo := models.Photo{
			PhotoID: photoID,
			AlbumID: albumID,
			Seq:     seq,
			Status:  "processing",
			S3Key:   finalKey,
		}
		if err := h.db.PutPhoto(ctx, photo); err != nil {
			log.Printf("ERROR: put photo record %s: %v", photoID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create photo record"})
			return
		}

		// Return 202 before S3 upload
		c.JSON(http.StatusAccepted, models.PhotoAccepted{
			PhotoID: photoID,
			Seq:     seq,
			Status:  "processing",
		})

		// Async: upload to S3, then directly mark completed in DynamoDB (skip SQS + worker roundtrip)
		go func() {
			bgCtx := context.Background()
			if err := h.s3.UploadBytes(bgCtx, finalKey, buf); err != nil {
				log.Printf("ERROR: async S3 upload for photo %s: %v", photoID, err)
				_ = h.db.UpdatePhotoFailed(bgCtx, photoID)
				return
			}
			publicURL := h.s3.PublicURL(finalKey)
			if err := h.db.UpdatePhotoCompleted(bgCtx, photoID, publicURL, finalKey); err != nil {
				log.Printf("ERROR: async mark completed for photo %s: %v", photoID, err)
			}
		}()
		return
	}

	// Large file: synchronous upload (stream rest of body with buffered prefix)
	combined := io.MultiReader(bytes.NewReader(peek[:n]), photoPart)
	if err := h.s3.Upload(ctx, finalKey, combined); err != nil {
		log.Printf("ERROR: upload photo %s to S3: %v", photoID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to upload photo"})
		return
	}

	photo := models.Photo{
		PhotoID: photoID,
		AlbumID: albumID,
		Seq:     seq,
		Status:  "processing",
		S3Key:   finalKey,
	}
	if err := h.db.PutPhoto(ctx, photo); err != nil {
		log.Printf("ERROR: put photo record %s: %v", photoID, err)
		_ = h.s3.Delete(ctx, finalKey)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create photo record"})
		return
	}

	msg := store.PhotoMessage{PhotoID: photoID, AlbumID: albumID, S3Key: finalKey}
	if err := h.sqs.SendPhotoMessage(ctx, msg); err != nil {
		log.Printf("ERROR: send SQS message for photo %s: %v", photoID, err)
	}

	c.JSON(http.StatusAccepted, models.PhotoAccepted{
		PhotoID: photoID,
		Seq:     seq,
		Status:  "processing",
	})
}

// GetPhoto handles GET /albums/:album_id/photos/:photo_id
func (h *PhotoHandler) GetPhoto(c *gin.Context) {
	albumID := c.Param("album_id")
	photoID := c.Param("photo_id")

	photo, err := h.db.GetPhoto(c.Request.Context(), photoID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get photo"})
		return
	}
	if photo == nil || photo.AlbumID != albumID {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	c.JSON(http.StatusOK, photo)
}

// DeletePhoto handles DELETE /albums/:album_id/photos/:photo_id
func (h *PhotoHandler) DeletePhoto(c *gin.Context) {
	albumID := c.Param("album_id")
	photoID := c.Param("photo_id")
	ctx := c.Request.Context()

	photo, err := h.db.GetPhoto(ctx, photoID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get photo"})
		return
	}
	if photo == nil || photo.AlbumID != albumID {
		c.Status(http.StatusNoContent)
		return
	}

	if photo.S3Key != "" {
		if err := h.s3.Delete(ctx, photo.S3Key); err != nil {
			log.Printf("ERROR: delete S3 object %s: %v", photo.S3Key, err)
		}
	}

	if err := h.db.DeletePhoto(ctx, photoID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete photo"})
		return
	}

	c.Status(http.StatusNoContent)
}
