package models

// Photo represents a photo record in DynamoDB.
// Covers both the "processing" and "completed" lifecycle stages.
type Photo struct {
	PhotoID string `json:"photo_id" dynamodbav:"photo_id"`
	AlbumID string `json:"album_id" dynamodbav:"album_id"`
	Seq     int    `json:"seq" dynamodbav:"seq"`
	Status  string `json:"status" dynamodbav:"status"`
	// URL is the public S3 URL. Only present when status == "completed".
	// omitempty ensures it's excluded from JSON when empty (during "processing").
	URL string `json:"url,omitempty" dynamodbav:"url,omitempty"`
	// S3Key is used internally for deletion. Not exposed in API responses.
	S3Key string `json:"-" dynamodbav:"s3_key,omitempty"`
}

// PhotoAccepted is the response for POST /albums/:album_id/photos (202).
// Only includes photo_id, seq, status — no album_id or url.
type PhotoAccepted struct {
	PhotoID string `json:"photo_id"`
	Seq     int    `json:"seq"`
	Status  string `json:"status"`
}
