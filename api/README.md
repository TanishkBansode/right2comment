# RTC-API

RTC-API is a lightweight Go API server built with Fiber that provides commenting functionality for the Right to Comment system. It serves as the backend API for managing comments associated with YouTube videos.

**Storage**: Uses Vercel KV (Redis) when deployed, with automatic fallback to file-based storage for local development.

## Project Structure

```
RTC-API/
│   go.mod            # Go module file
│   go.sum            # Go module checksum
│   main.go           # Main application file
│
└───database/         # Database operations
        database.go   # Database initialization and queries
```

## Features

- RESTful API endpoints for comment management
- Vercel KV (Redis) storage for production
- Automatic fallback to file-based storage for local development
- Serverless deployment on Vercel
- Timestamp formatting for comments
- Support for various YouTube URL formats

## Prerequisites

### Local Development
- Go (1.23 or later)

### Production Deployment
- Vercel account
- Vercel KV database (can be created in Vercel dashboard)

## Installation

1. Clone the repository:
```bash
git clone https://github.com/TanishkBansode/rtc.git
cd rtc/rtc-api
```

2. Install dependencies:
```bash
go mod tidy
```

## Local Development

Start the server with:
```bash
cd api
go run main.go
```

The server will start on `http://localhost:3000` using file-based storage in `/tmp/comments-data`.

**Optional**: To test with Vercel KV locally, set environment variables:
```bash
export KV_REST_API_URL="your-kv-url"
export KV_REST_API_TOKEN="your-kv-token"
go run main.go
```

## API Endpoints

### Root Endpoint, just to test if it works
```
GET /
Response: "API is running."
```

### Get Comments
```
GET /comments/:videoId
Or
GET /comments?url={videoURL}
Or
GET /comments?videoID={videoId}
Response: JSON array of comments
```

Example response:
```json
[
  {
    "text": "Comment text here",
    "createdAt": "2 Jan 2024"
  }
]
```

### Add Comment
```
curl -X POST \
  -d "comment=This is a truly magnificent video!" \
  "http://localhost:3000/comments/:videoID"
  
Or

curl -X POST \
  -d "comment=Posting this comment via a full URL in the form data!" \
  -d "videoUrl={videoURL}" \
  "http://localhost:3000/comments"
```

## Dependencies

- [Fiber v2](https://github.com/gofiber/fiber) - Fast HTTP framework
- SQLite3 - Database engine

## Error Handling

The API returns appropriate HTTP status codes and error messages:

- 200: Successful operation
- 500: Internal server error with error message

Example error response:
```json
{
  "error": "Failed to load comments"
}
```

## Deployment to Vercel

### Step 1: Create Vercel KV Database

1. Go to your [Vercel Dashboard](https://vercel.com/dashboard)
2. Navigate to **Storage** tab
3. Click **Create Database** → **KV** (Redis)
4. Note the **KV_REST_API_URL** and **KV_REST_API_TOKEN** (these are auto-added to your project)

### Step 2: Deploy

You have two options:

**Option A: Deploy via GitHub (Recommended)**
1. Push your code to GitHub
2. Import the repository in Vercel
3. Vercel will automatically detect the Go API
4. Environment variables from KV are automatically linked

**Option B: Deploy via CLI**
```bash
npm i -g vercel
vercel login
vercel --prod
```

### Step 3: Verify Deployment

Test your deployed API:
```bash
curl https://your-project.vercel.app/
# Should return: "API is running."

# Test adding a comment
curl -X POST -F "comment=Hello World" https://your-project.vercel.app/comments/VIDEO_ID

# Test getting comments
curl https://your-project.vercel.app/comments/VIDEO_ID
```

## Configuration

The API automatically detects the environment:
- **Vercel (Production)**: Uses Vercel KV for persistent storage
- **Local Development**: Uses file-based storage in `/tmp/comments-data`
- **Date Format**: Comments are formatted as "2 Jan 2006"
- **Response Format**: All responses are JSON

## Testing

To test the API endpoints:

1. Start the server
2. Use curl or Postman(or Thunderclient!) to make requests:

```bash
# Get comments
curl http://localhost:3000/comments/VIDEO_ID

# Add comment
curl -X POST -F "comment=Your comment" http://localhost:3000/comments/VIDEO_ID
```

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.
