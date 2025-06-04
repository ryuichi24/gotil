package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func handleWebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
	}
	defer conn.Close()

	log.Println("WebSocket connection established")

	for {
		messageType, msg, err := conn.ReadMessage()

		log.Printf("Received message type: %d", messageType)

		if err != nil {
			log.Printf("Error reading message: %v", err)
			break
		}

		log.Printf("Received message: %s", msg)
		err = conn.WriteMessage(messageType, msg)
		if err != nil {
			log.Printf("Error writing message: %v", err)
			break
		}
		log.Printf("Echoed message: %s", msg)
	}
}
func main() {
	r := gin.Default()

	r.GET("/ws", handleWebSocket)

	r.GET("/", func(c *gin.Context) {
		c.String(http.StatusOK, "WebSocket server is running. Connect to /ws")
	})

	log.Println("Starting WebSocket server on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
	fmt.Println("WebSocket server is running on http://localhost:8080")
}
