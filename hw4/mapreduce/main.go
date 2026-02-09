package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"unicode"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gin-gonic/gin"
)

var s3Client *s3.Client
var bucketName = "mapreduce-bucket-yu"

func main() {
	// Initialize AWS S3 client
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("us-west-2"))
	if err != nil {
		panic("unable to load AWS config: " + err.Error())
	}
	s3Client = s3.NewFromConfig(cfg)

	// Setup router
	router := gin.Default()

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	// Splitter endpoint: splits input file into N chunks
	router.GET("/split", handleSplit)

	// Mapper endpoint: counts words in a chunk
	router.GET("/map", handleMap)

	// Reducer endpoint: aggregates results from mappers
	router.GET("/reduce", handleReduce)

	router.Run("0.0.0.0:8080")
}

// handleSplit reads the input file and splits it into N chunks
// Query params: key (S3 key of input file), n (number of chunks, default 3)
func handleSplit(c *gin.Context) {
	key := c.Query("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing 'key' parameter"})
		return
	}

	// Default to 3 chunks
	n := 3

	// Read file from S3
	content, err := readFromS3(key)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read from S3: " + err.Error()})
		return
	}

	// Split content into n chunks by lines
	lines := strings.Split(string(content), "\n")
	chunkSize := (len(lines) + n - 1) / n

	var chunkKeys []string
	for i := 0; i < n; i++ {
		start := i * chunkSize
		end := start + chunkSize
		if end > len(lines) {
			end = len(lines)
		}
		if start >= len(lines) {
			break
		}

		chunkContent := strings.Join(lines[start:end], "\n")
		chunkKey := "chunks/chunk_" + string(rune('0'+i)) + ".txt"

		// Write chunk to S3
		err := writeToS3(chunkKey, []byte(chunkContent))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to write chunk to S3: " + err.Error()})
			return
		}
		chunkKeys = append(chunkKeys, chunkKey)
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "split complete",
		"chunk_keys": chunkKeys,
	})
}

// handleMap reads a chunk and counts word occurrences
// Query params: key (S3 key of chunk file)
func handleMap(c *gin.Context) {
	key := c.Query("key")
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing 'key' parameter"})
		return
	}

	// Read chunk from S3
	content, err := readFromS3(key)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read from S3: " + err.Error()})
		return
	}

	// Count words
	wordCounts := countWords(string(content))

	// Convert to JSON
	jsonData, err := json.Marshal(wordCounts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to marshal word counts"})
		return
	}

	// Extract chunk identifier from key (e.g., "chunk_0" from "chunks/chunk_0.txt")
	parts := strings.Split(key, "/")
	filename := parts[len(parts)-1]
	chunkID := strings.TrimSuffix(filename, ".txt")

	// Write result to S3
	resultKey := "results/" + chunkID + "_result.json"
	err = writeToS3(resultKey, jsonData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to write result to S3: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "map complete",
		"result_key": resultKey,
		"word_count": len(wordCounts),
	})
}

// handleReduce aggregates results from all mappers
// Query params: keys (comma-separated S3 keys of result files)
func handleReduce(c *gin.Context) {
	keysParam := c.Query("keys")
	if keysParam == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing 'keys' parameter"})
		return
	}

	keys := strings.Split(keysParam, ",")
	finalCounts := make(map[string]int)

	// Read and aggregate all results
	for _, key := range keys {
		content, err := readFromS3(strings.TrimSpace(key))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read from S3: " + err.Error()})
			return
		}

		var wordCounts map[string]int
		err = json.Unmarshal(content, &wordCounts)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to unmarshal word counts"})
			return
		}

		// Aggregate counts
		for word, count := range wordCounts {
			finalCounts[word] += count
		}
	}

	// Convert to JSON
	jsonData, err := json.Marshal(finalCounts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to marshal final counts"})
		return
	}

	// Write final result to S3
	finalKey := "results/final_result.json"
	err = writeToS3(finalKey, jsonData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to write final result to S3: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":          "reduce complete",
		"final_result_key": finalKey,
		"unique_words":     len(finalCounts),
	})
}

// countWords counts occurrences of each word in the text
func countWords(text string) map[string]int {
	counts := make(map[string]int)

	// Convert to lowercase and split by non-letter characters
	words := strings.FieldsFunc(strings.ToLower(text), func(c rune) bool {
		return !unicode.IsLetter(c)
	})

	for _, word := range words {
		if word != "" {
			counts[word]++
		}
	}

	return counts
}

// readFromS3 reads a file from S3 bucket
func readFromS3(key string) ([]byte, error) {
	output, err := s3Client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: &bucketName,
		Key:    &key,
	})
	if err != nil {
		return nil, err
	}
	defer output.Body.Close()

	return io.ReadAll(output.Body)
}

// writeToS3 writes data to S3 bucket
func writeToS3(key string, data []byte) error {
	_, err := s3Client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket: &bucketName,
		Key:    &key,
		Body:   bytes.NewReader(data),
	})
	return err
}
