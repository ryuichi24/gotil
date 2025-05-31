package main // Declares the main package for an executable program

import (
	"fmt" // For formatted I/O operations like printing to console
	"log" // For logging errors or info

	"github.com/gin-gonic/gin" // Gin web framework for building HTTP servers
	"github.com/ryuichi24/goup/internal/upload"
)

func main() {
	router := gin.Default()

	baseRouter := router.Group("/api")
	{
		upload.NewUploadController(baseRouter) // Initialize the upload controller with the base router
	}

	// Start the server on port 8080 and log any fatal errors
	log.Fatal(router.Run(":8080"))

	// Log startup message
	fmt.Println("Server started at http://localhost:8080")
}
