package models

// Album represents an album record in DynamoDB.
// JSON field names match the OpenAPI spec exactly.
type Album struct {
	AlbumID     string `json:"album_id" dynamodbav:"album_id"`
	Title       string `json:"title" dynamodbav:"title"`
	Description string `json:"description" dynamodbav:"description"`
	Owner       string `json:"owner" dynamodbav:"owner"`
	// PhotoSeqCounter is used internally for atomic seq assignment.
	// Not included in JSON responses (json:"-").
	PhotoSeqCounter int `json:"-" dynamodbav:"photo_seq_counter"`
}
