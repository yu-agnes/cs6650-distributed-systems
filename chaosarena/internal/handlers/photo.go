package handlers

import (
	"fmt"
	"io"
	"log"
	"net/http"

	"chaosarena/internal/models"
	"chaosarena/internal/store"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type PhotoHandler struct {
	db  *store.DynamoStore
	s3  *store.S3Store
	sqs *store.SQSStore
}

func NewPhotoHandler(db *store.DynamoStore, s3 *store.S3Store, sqs *store.SQSStore) *PhotoHandler {
	return &PhotoHandler{db: db, s3: s3, sqs: sqs}
}

// UploadPhoto handles POST /albums/:album_id/photos
// Streams the upload directly to the final S3 location (no temp copy).
// Worker only needs to update DynamoDB status — no S3 operations.
func (h *PhotoHandler) UploadPhoto(c *gin.Context) {
	albumID := c.Param("album_id")
	ctx := c.Request.Context()

	// Verify the album exists before consuming the upload
	album, err := h.db.GetAlbum(ctx, albumID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check album"})
		return
	}
	if album == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	// Stream the multipart body — find the "photo" part without buffering to disk
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

	// Step 1: Atomically get next seq number
	seq, err := h.db.IncrementPhotoSeq(ctx, albumID)
	if err != nil {
		log.Printf("ERROR: increment seq for album %s: %v", albumID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to assign sequence number"})
		return
	}

	// Step 2: Stream directly to final S3 location (no temp key, no disk buffering)
	photoID := uuid.New().String()
	finalKey := fmt.Sprintf("albums/%s/photos/%s", albumID, photoID)

	if err := h.s3.Upload(ctx, finalKey, photoPart); err != nil {
		log.Printf("ERROR: upload photo %s to S3: %v", photoID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to upload photo"})
		return
	}

	// Step 3: Write photo record with status "processing"
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

	// Step 4: Send message to worker via SQS
	msg := store.PhotoMessage{
		PhotoID: photoID,
		AlbumID: albumID,
		S3Key:   finalKey,
	}
	if err := h.sqs.SendPhotoMessage(ctx, msg); err != nil {
		log.Printf("ERROR: send SQS message for photo %s: %v", photoID, err)
	}

	// Step 5: Return 202 Accepted
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
