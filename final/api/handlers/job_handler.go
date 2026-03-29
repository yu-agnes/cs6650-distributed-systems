package handlers

import (
	"net/http"

	"api/models"
	"api/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type JobHandler struct {
	dynamoSvc *services.DynamoService
	sqsSvc    *services.SQSService
}

func NewJobHandler(dynamoSvc *services.DynamoService, sqsSvc *services.SQSService) *JobHandler {
	return &JobHandler{
		dynamoSvc: dynamoSvc,
		sqsSvc:    sqsSvc,
	}
}

// POST /jobs - submit a list of URLs to scrape
type CreateJobRequest struct {
	URLs []string `json:"urls" binding:"required,min=1"`
}

type CreateJobResponse struct {
	JobID string `json:"jobID"`
}

func (h *JobHandler) CreateJob(c *gin.Context) {
	var req CreateJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "urls field is required and must be non-empty"})
		return
	}

	// Generate unique job ID
	jobID := uuid.New().String()

	// Create job record in DynamoDB with status "pending"
	job := models.NewJob(jobID, req.URLs)
	if err := h.dynamoSvc.CreateJob(c.Request.Context(), job); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create job"})
		return
	}

	// Send job to SQS for workers
	if err := h.sqsSvc.SendJob(c.Request.Context(), jobID, req.URLs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to enqueue job"})
		return
	}

	c.JSON(http.StatusAccepted, CreateJobResponse{JobID: jobID})
}

// GET /jobs/:id - check job status and results
func (h *JobHandler) GetJob(c *gin.Context) {
	jobID := c.Param("id")

	job, err := h.dynamoSvc.GetJob(c.Request.Context(), jobID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get job"})
		return
	}

	if job == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}

	c.JSON(http.StatusOK, job)
}
