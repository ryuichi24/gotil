package upload

import (
	"fmt"           // For formatted I/O operations like printing to console
	"io"            // For copying data between streams (e.g., file copy)
	"log"           // For logging errors or info
	"net/http"      // Provides HTTP client and server implementations
	"os"            // For interacting with the file system
	"path/filepath" // For safely constructing file paths

	// For string manipulation (not used here but useful)
	"time" // For working with timestamps (used in filename generation)

	"github.com/gin-gonic/gin"
)

const (
	uploadDir     = "./.upload" // Directory where uploaded files will be saved
	maxUploadSize = 10 << 20    // Maximum file size allowed (10 MB = 10 * 2^20 bytes)
)

func NewUploadController(baseRouter *gin.RouterGroup) {
	routerGroup := baseRouter.Group("/upload")

	// Ensure the upload directory exists; create it if necessary
	if err := os.MkdirAll(uploadDir, os.ModePerm); err != nil {
		log.Fatalf("Could not create upload directory: %v", err) // Exit if directory creation fails
	}

	// Register handlers for the upload routes
	fmt.Println("Initializing upload controller...")

	uploadHandler(routerGroup)
	imagesHandler(routerGroup)
}

func uploadHandler(routerGroup *gin.RouterGroup) {
	routerGroup.POST("", func(c *gin.Context) {
		req := c.Request
		res := c.Writer

		// Only allow POST requests
		if req.Method != http.MethodPost {
			http.Error(res, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Limit the size of the request body to prevent large uploads
		req.Body = http.MaxBytesReader(res, req.Body, maxUploadSize)

		// Parse the multipart form data
		if err := req.ParseMultipartForm(maxUploadSize); err != nil {
			http.Error(res, "File too large", http.StatusBadRequest)
			return
		}

		// Retrieve the uploaded file from the "image" form field
		file, handler, err := req.FormFile("image")
		if err != nil {
			http.Error(res, "Invalid file", http.StatusBadRequest)
			return
		}
		defer file.Close() // Ensure the uploaded file stream is closed at the end

		// Extract the file extension (e.g., ".jpg", ".png")
		ext := filepath.Ext(handler.Filename)

		// Generate a unique filename using the current timestamp
		filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)

		// Construct the full file path for saving the uploaded file
		filepath := filepath.Join(uploadDir, filename)

		// Create the destination file on disk
		dst, err := os.Create(filepath)
		if err != nil {
			http.Error(res, "Unable to save the file", http.StatusInternalServerError)
			return
		}
		defer dst.Close() // Ensure the destination file is closed

		// Copy the uploaded file contents to the new file on disk
		if _, err := io.Copy(dst, file); err != nil {
			http.Error(res, "Unable to save the file", http.StatusInternalServerError)
			return
		}

		// Build the URL to access the uploaded file
		fileURL := fmt.Sprintf("http://%s/api/images/%s", req.Host, filename)

		// Respond with a JSON payload containing the image URL
		res.Header().Set("Content-Type", "application/json")
		res.WriteHeader(http.StatusCreated)
		fmt.Fprintf(res, `{"imageUrl": "%s"}`, fileURL)
	})
}

func imagesHandler(routerGroup *gin.RouterGroup) {
	routerGroup.Static("/images/", uploadDir)
}
