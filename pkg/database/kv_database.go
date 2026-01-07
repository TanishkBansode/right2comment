package database

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// KVClient handles interactions with Vercel KV via REST API
type KVClient struct {
	apiURL   string
	apiToken string
	client   *http.Client
}

// NewKVClient creates a new Vercel KV client
func NewKVClient() *KVClient {
	return &KVClient{
		apiURL:   os.Getenv("KV_REST_API_URL"),
		apiToken: os.Getenv("KV_REST_API_TOKEN"),
		client:   &http.Client{Timeout: 10 * time.Second},
	}
}

// IsConfigured checks if KV environment variables are set
func (kv *KVClient) IsConfigured() bool {
	return kv.apiURL != "" && kv.apiToken != ""
}

// makeRequest is a helper to make authenticated requests to Vercel KV REST API
func (kv *KVClient) makeRequest(method, endpoint string, body interface{}) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, kv.apiURL+endpoint, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+kv.apiToken)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := kv.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("KV API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// Get retrieves a value from KV
func (kv *KVClient) Get(key string) ([]byte, error) {
	respBody, err := kv.makeRequest("GET", "/get/"+key, nil)
	if err != nil {
		return nil, err
	}

	var response struct {
		Result string `json:"result"`
	}
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if response.Result == "" {
		return nil, nil // Key doesn't exist
	}

	return []byte(response.Result), nil
}

// Set stores a value in KV
func (kv *KVClient) Set(key string, value []byte) error {
	payload := map[string]string{
		"key":   key,
		"value": string(value),
	}

	_, err := kv.makeRequest("POST", "/set", payload)
	return err
}

// AddCommentKV adds a comment to Vercel KV
func AddCommentKV(kv *KVClient, videoID, text string) error {
	if videoID == "" {
		return fmt.Errorf("videoID required")
	}
	if text == "" {
		return fmt.Errorf("comment text required")
	}

	key := "comments:" + videoID

	// Get existing comments
	var comments []comment
	data, err := kv.Get(key)
	if err != nil {
		return fmt.Errorf("failed to get existing comments: %w", err)
	}

	if data != nil && len(data) > 0 {
		if err := json.Unmarshal(data, &comments); err != nil {
			return fmt.Errorf("failed to parse existing comments: %w", err)
		}
	}

	// Append new comment
	comments = append(comments, comment{
		Text:      text,
		CreatedAt: time.Now().Format(time.RFC3339),
	})

	// Save back to KV
	commentsJSON, err := json.Marshal(comments)
	if err != nil {
		return fmt.Errorf("failed to marshal comments: %w", err)
	}

	if err := kv.Set(key, commentsJSON); err != nil {
		return fmt.Errorf("failed to save comments: %w", err)
	}

	return nil
}

// GetCommentsKV retrieves comments from Vercel KV
func GetCommentsKV(kv *KVClient, videoID string) ([]map[string]string, error) {
	if videoID == "" {
		return nil, fmt.Errorf("videoID required")
	}

	key := "comments:" + videoID

	data, err := kv.Get(key)
	if err != nil {
		return nil, fmt.Errorf("failed to get comments: %w", err)
	}

	// Return empty slice if no comments exist
	if data == nil || len(data) == 0 {
		return []map[string]string{}, nil
	}

	var comments []comment
	if err := json.Unmarshal(data, &comments); err != nil {
		return nil, fmt.Errorf("failed to parse comments: %w", err)
	}

	out := make([]map[string]string, 0, len(comments))
	for _, c := range comments {
		out = append(out, map[string]string{
			"text":      c.Text,
			"createdAt": c.CreatedAt,
		})
	}

	return out, nil
}
