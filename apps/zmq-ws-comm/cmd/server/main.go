package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"

	"github.com/gin-gonic/gin"
	srv "github.com/ryuichi24/zmq-ws-comm/pkg/server"
	ws "github.com/ryuichi24/zmq-ws-comm/pkg/websocket"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	wg.Add(2)

	// apiBaseRouter := r.Group("/api")

	// web socket connection manager setup
	wscm := ws.NewWebSocketConnectionManager()
	wscm.EventRegistry.On("example:event", func(evt ws.Event) error {
		log.Printf("Handling example_event for WSockId: %s", evt.WSockId)
		return nil
	},
	)
	go wscm.Start(ctx, &wg)

	// server setup
	router := gin.Default()
	router.GET("ws", wscm.HandleWSConnection)
	server := srv.NewServer(":8080", router)
	go server.Start(ctx, &wg)

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

	// Wait for shutdown signal
	log.Println("Press Ctrl+C to stop the server...")
	<-sigChan
	log.Println("Received shutdown signal...")

	cancel()

	// Wait for all goroutines to finish
	wg.Wait()

	log.Println("Application stopped gracefully")
}
