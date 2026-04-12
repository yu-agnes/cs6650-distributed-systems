package handlers

import (
	"fmt"
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
// 1. Atomically increment seq counter
// 2. Upload photo to S3 temp location
// 3. Write "processing" record to DynamoDB
// 4. Send SQS message to worker
// 5. Return 202 immediately
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

	// Parse multipart form - get the photo file and its header (for size)
	file, header, err := c.Request.FormFile("photo")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing or malformed photo field"})
		return
	}
	defer file.Close()

	// Step 1: Atomically get next seq number
	seq, err := h.db.IncrementPhotoSeq(ctx, albumID)
	if err != nil {
		log.Printf("ERROR: increment seq for album %s: %v", albumID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to assign sequence number"})
		return
	}

	// Step 2: Generate photo ID and upload to S3 temp location
	photoID := uuid.New().String()
	tempKey := fmt.Sprintf("temp/%s/%s", albumID, photoID)

	if err := h.s3.Upload(ctx, tempKey, file, header.Size); err != nil {
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
		S3Key:   tempKey,
	}
	if err := h.db.PutPhoto(ctx, photo); err != nil {
		log.Printf("ERROR: put photo record %s: %v", photoID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create photo record"})
		return
	}

	// Step 4: Send message to worker via SQS
	msg := store.PhotoMessage{
		PhotoID:   photoID,
		AlbumID:   albumID,
		S3TempKey: tempKey,
	}
	if err := h.sqs.SendPhotoMessage(ctx, msg); err != nil {
		log.Printf("ERROR: send SQS message for photo %s: %v", photoID, err)
		// Photo is already in "processing" state - worker won't pick it up
		// but we still return 202 since the record exists
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
// Must complete within 5 seconds. Deletes both DynamoDB record and S3 file.
func (h *PhotoHandler) DeletePhoto(c *gin.Context) {
	albumID := c.Param("album_id")
	photoID := c.Param("photo_id")
	ctx := c.Request.Context()

	// Get the photo to find its S3 key
	photo, err := h.db.GetPhoto(ctx, photoID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get photo"})
		return
	}
	if photo == nil || photo.AlbumID != albumID {
		// Already deleted, never existed, or belongs to a different album - return 204
		c.Status(http.StatusNoContent)
		return
	}

	// Delete from S3 (if there's a file).
	// After completion, s3_key points to the final location; for in-progress
	// photos it points to the temp location. One delete covers both cases.
	if photo.S3Key != "" {
		if err := h.s3.Delete(ctx, photo.S3Key); err != nil {
			log.Printf("ERROR: delete S3 object %s: %v", photo.S3Key, err)
			// Continue to delete DynamoDB record even if S3 fails
		}
	}

	// Delete from DynamoDB
	if err := h.db.DeletePhoto(ctx, photoID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete photo"})
		return
	}

	c.Status(http.StatusNoContent)
}
