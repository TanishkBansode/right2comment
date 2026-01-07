package handler

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/TanishkBansode/right2comment/pkg/database"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/gofiber/fiber/v2/middleware/cors"
)

// Global regular expression for a valid 11-character YouTube ID.
// Compiling this once at startup is more efficient than recompiling on each request.
var validIDRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]{11}$`)

// isValidURL checks if an input string is a syntactically valid HTTP or HTTPS URL.
// This is a crucial first-pass filter before more complex parsing.
func isValidURL(input string) bool {
	parsedURL, err := url.ParseRequestURI(input)
	if err != nil {
		return false
	}
	scheme := strings.ToLower(parsedURL.Scheme)
	return scheme == "http" || scheme == "https"
}

// extractVideoID is a highly robust function designed to extract a YouTube video ID
// from various formats, including raw IDs, standard watch URLs, shortlinks (youtu.be),
// and embed/shorts URLs. It employs multiple strategies for maximum compatibility.
func extractVideoID(input string) (string, error) {
	if input == "" {
		return "", fmt.Errorf("input cannot be empty")
	}

	// Strategy 1: The input is already a valid 11-character ID.
	if validIDRegex.MatchString(input) {
		return input, nil
	}

	// Strategy 2: Attempt to parse as a URL for structured extraction.
	parsedURL, err := url.Parse(input)
	if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https-") {
		// If it's not a valid URL, fall back to a broad regex search.
		fallbackRe := regexp.MustCompile(`[a-zA-Z0-9_-]{11}`)
		if match := fallbackRe.FindString(input); len(match) == 11 {
			return match, nil
		}
		return "", fmt.Errorf("input is not a valid URL and no video ID could be found")
	}

	hostname := strings.ToLower(parsedURL.Hostname())
	path := parsedURL.Path

	// Handle youtu.be shortlinks (e.g., youtu.be/ID)
	if strings.Contains(hostname, "youtu.be") {
		videoID := strings.Trim(path, "/")
		if validIDRegex.MatchString(videoID) {
			return videoID, nil
		}
	}

	// Handle standard youtube.com and youtube-nocookie.com URLs
	if strings.Contains(hostname, "youtube.com") || strings.Contains(hostname, "youtube-nocookie.com") {
		// Handle ?v=ID from the query parameters
		if videoID := parsedURL.Query().Get("v"); validIDRegex.MatchString(videoID) {
			return videoID, nil
		}

		// Handle paths like /embed/ID, /shorts/ID, /v/ID
		pathSegments := strings.Split(path, "/")
		for _, segment := range pathSegments {
			if validIDRegex.MatchString(segment) {
				return segment, nil
			}
		}
	}

	// Final Fallback: As a last resort, search the entire raw input string for an ID.
	// This can catch malformed URLs that the parser missed.
	fallbackRe := regexp.MustCompile(`[a-zA-Z0-9_-]{11}`)
	if match := fallbackRe.FindString(input); len(match) == 11 {
		return match, nil
	}

	return "", fmt.Errorf("no valid YouTube video ID found in input: %s", input)
}

// getVideoIDFromRequest encapsulates the logic for retrieving and validating a video ID
// from the Fiber context. It intelligently checks route parameters, query strings,
// and form values, making handlers cleaner and more DRY (Don't Repeat Yourself).
func getVideoIDFromRequest(c *fiber.Ctx) (string, error) {
	// Prioritize sources: Route Param > Query Param > Form Value
	identifier := c.Params("videoID")
	if identifier == "" {
		identifier = c.Query("videoID")
	}
	if identifier == "" {
		identifier = c.Query("url") // Add an intuitive "url" query param
	}
	if identifier == "" {
		identifier = c.FormValue("videoID")
	}
	if identifier == "" {
		identifier = c.FormValue("videoUrl") // Aliases for form values
	}

	if identifier == "" {
		return "", fmt.Errorf("a video ID or URL must be provided")
	}

	videoID, err := extractVideoID(identifier)
	if err != nil {
		return "", fmt.Errorf("could not extract a valid video ID from '%s'", identifier)
	}

	return videoID, nil
}

// setupRoutes defines and organizes all application routes.
// Using route grouping enhances maintainability.
func setupRoutes(app *fiber.App) {
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("API is running.")
	})

	// Group all comment-related endpoints under the "/comments" prefix.
	commentsGroup := app.Group("/comments")
	{
		// This route handles both /:videoID and query params like ?url=...
		commentsGroup.Get("/:videoID?", getComments)
		commentsGroup.Post("/:videoID?", addComment)
	}
}

// Initialize Fiber app at package level for serverless reuse
var app *fiber.App

func init() {
	// Initialize database (KV if available, otherwise file storage)
	// Empty string triggers default logic (/tmp on Vercel, or custom logic)
	database.InitDB("")

	// Create and configure Fiber app
	app = fiber.New()

	// Enable CORS for all origins (or restrict to youtube.com in production)
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*", // For development/demo. Ideally: "https://www.youtube.com"
		AllowHeaders: "Origin, Content-Type, Accept",
	}))

	setupRoutes(app)
}

// Handler is the entry point for Vercel serverless functions
func Handler(w http.ResponseWriter, r *http.Request) {
	// Convert Fiber app to standard HTTP handler
	handler := adaptor.FiberApp(app)
	handler.ServeHTTP(w, r)
}

// main is used for local development only
// When deployed to Vercel, only the Handler function is used
func main() {
	log.Println("Starting server on port 3000...")
	log.Println("NOTE: This is local development mode. Database will use file storage unless KV env vars are set.")
	if err := app.Listen(":3000"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// addComment handles the creation of a new comment.
// It follows REST principles by returning 201 Created on success.
func addComment(c *fiber.Ctx) error {
	videoID, err := getVideoIDFromRequest(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	commentText := c.FormValue("comment")
	if commentText == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "comment text is required"})
	}

	if err := database.AddComment(videoID, commentText); err != nil {
		log.Printf("ERROR: Failed to add comment for video %s: %v\n", videoID, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Failed to save comment"})
	}

	// On successful creation, it's best practice to return the updated resource.
	// We'll fetch all comments for the video and return them.
	allComments, err := fetchAndFormatComments(videoID)
	if err != nil {
		// If fetching fails, we still successfully created the comment,
		// so we return a success status with an empty list.
		log.Printf("WARN: Comment was added for video %s, but failed to fetch updated list: %v\n", videoID, err)
		return c.Status(fiber.StatusCreated).JSON([]fiber.Map{})
	}

	return c.Status(fiber.StatusCreated).JSON(allComments)
}

// getComments handles retrieving all comments for a given video.
// It now correctly handles URLs passed as parameters.
func getComments(c *fiber.Ctx) error {
	videoID, err := getVideoIDFromRequest(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	commentsJSON, err := fetchAndFormatComments(videoID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusOK).JSON(commentsJSON)
}

// fetchAndFormatComments is a helper to abstract the logic of fetching
// comments from the database and formatting them for the JSON response.
func fetchAndFormatComments(videoID string) ([]fiber.Map, error) {
	// This function is now corrected to assume database.GetComments returns []map[string]string
	comments, err := database.GetComments(videoID)
	if err != nil {
		log.Printf("ERROR: Failed to get comments for video %s: %v\n", videoID, err)
		return nil, fmt.Errorf("failed to load comments")
	}

	// Pre-allocate the slice for better performance.
	commentsJSON := make([]fiber.Map, 0, len(comments))
	for _, comment := range comments {
		// Since `comment` is map[string]string, we access its values directly.
		// No type assertion is needed.
		text, textOk := comment["text"]
		createdAt, dateOk := comment["createdAt"]
		if !textOk || !dateOk {
			log.Printf("WARN: Skipping malformed comment data for video %s: %+v\n", videoID, comment)
			continue
		}

		parsedDate, err := time.Parse(time.RFC3339, createdAt)
		if err != nil {
			log.Printf("WARN: Could not parse date '%s' for video %s. Using current time as fallback. Error: %v\n", createdAt, videoID, err)
			parsedDate = time.Now()
		}

		commentsJSON = append(commentsJSON, fiber.Map{
			"text":      text,
			"createdAt": parsedDate.Format("2 Jan 2006"),
		})
	}

	return commentsJSON, nil
}
