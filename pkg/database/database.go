package database

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	dataDir  string
	mu       sync.Mutex
	kvClient *KVClient
	useKV    bool
)

type comment struct {
	Text      string `json:"text"`
	CreatedAt string `json:"createdAt"` // RFC3339
}

// InitDB initializes the storage. Uses Vercel KV if configured, otherwise falls back to file storage.
func InitDB(dir string) error {
	// Try to initialize KV client
	kvClient = NewKVClient()
	useKV = kvClient.IsConfigured()

	if useKV {
		fmt.Println("Database: Using Vercel KV Storage")
		return nil
	}

	fmt.Println("Database: Using File Storage (Fallback)")

	// Fallback to file-based storage
	// If dir is relative or risky, force /tmp on Vercel or use safe default
	if os.Getenv("VERCEL") != "" || dir == "" {
		fmt.Println("Environment: Vercel/Serverless detected or no dir provided. Using /tmp/comments-data")
		dir = "/tmp/comments-data"
	}

	abs, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(abs, 0755); err != nil {
		return err
	}
	dataDir = abs
	fmt.Printf("Database: File storage initialized at %s\n", dataDir)
	return nil
}

// AddComment appends a comment for the given videoID.
// Uses Vercel KV if configured, otherwise uses file-based storage.
func AddComment(videoID, text string) error {
	if videoID == "" {
		return fmt.Errorf("videoID required")
	}
	if text == "" {
		return fmt.Errorf("comment text required")
	}

	// Use Vercel KV if available
	if useKV {
		return AddCommentKV(kvClient, videoID, text)
	}

	// Fallback to file-based storage
	if dataDir == "" {
		return fmt.Errorf("database not initialized")
	}

	mu.Lock()
	defer mu.Unlock()

	path := filepath.Join(dataDir, videoID+".json")

	var comments []comment
	// If file exists, load existing comments.
	if b, err := os.ReadFile(path); err == nil {
		if len(b) > 0 {
			if err := json.Unmarshal(b, &comments); err != nil {
				// If corrupted, start fresh but report error to caller.
				return fmt.Errorf("failed to parse existing comments: %w", err)
			}
		}
	}

	comments = append(comments, comment{
		Text:      text,
		CreatedAt: time.Now().Format(time.RFC3339),
	})

	out, err := json.MarshalIndent(comments, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(path, out, 0644); err != nil {
		return err
	}

	return nil
}

// GetComments returns stored comments for a video as []map[string]string
// where each map contains "text" and "createdAt" keys (createdAt in RFC3339).
// Uses Vercel KV if configured, otherwise uses file-based storage.
func GetComments(videoID string) ([]map[string]string, error) {
	if videoID == "" {
		return nil, fmt.Errorf("videoID required")
	}

	// Use Vercel KV if available
	if useKV {
		return GetCommentsKV(kvClient, videoID)
	}

	// Fallback to file-based storage
	if dataDir == "" {
		return nil, fmt.Errorf("database not initialized")
	}

	mu.Lock()
	defer mu.Unlock()

	path := filepath.Join(dataDir, videoID+".json")

	// If file doesn't exist, return empty slice (no error).
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return []map[string]string{}, nil
	} else if err != nil {
		return nil, err
	}

	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(b) == 0 {
		return []map[string]string{}, nil
	}

	var comments []comment
	if err := json.Unmarshal(b, &comments); err != nil {
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
