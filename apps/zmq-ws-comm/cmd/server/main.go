package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pebbe/zmq4"
	"github.com/ryuichi24/zmq-ws-comm/pkg/network"
	srv "github.com/ryuichi24/zmq-ws-comm/pkg/server"
	"github.com/ryuichi24/zmq-ws-comm/pkg/util"
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

	// Find a free port for the zmq publisher
	retryConfig := util.RetryConfig{
		MaxAttempts:  5,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     2 * time.Second,
	}
	freePort, err := util.Retry(ctx, retryConfig, network.FindFreePort)
	if err != nil {
		log.Fatalf("Failed to find a free port: %v", err)
	}
	log.Printf("Found free port: %d", freePort)
	zmqPubAddr := fmt.Sprintf("tcp://localhost:%d", freePort)
	// Find a bound port of the zmq publisher

	// ZeroMQ manager setup
	zmqManager, err := NewZeroMQManager("tcp://localhost:63542", zmqPubAddr)
	if err != nil {
		log.Fatalf("Failed to create ZeroMQ manager: %v", err)
	}

	go zmqManager.Start(ctx, &wg)

	zmqManager.Listen(&ZmqMsgHandlerImpl{})

	go func() {
		for {
			zmqManager.PublishMessage([]string{"request:keymap", "Hello from ZeroMQ!"})
			time.Sleep(time.Second)
		}
	}()

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

type ZmqMsgHandlerImpl struct{}

func (h *ZmqMsgHandlerImpl) handle(msgFrames []string) error {
	log.Printf("Received ZeroMQ message: %v", msgFrames)
	return nil
}

type ZeroMQManager struct {
	subscriber *zmq4.Socket
	publisher  *zmq4.Socket
	subAddr    string
	pubAddr    string
}

func NewZeroMQManager(subAddr string, pubAddr string) (*ZeroMQManager, error) {
	subscriber, err := zmq4.NewSocket(zmq4.SUB)
	if err != nil {
		return nil, err
	}
	publisher, err := zmq4.NewSocket(zmq4.PUB)
	if err != nil {
		subscriber.Close()
		return nil, err
	}
	return &ZeroMQManager{
		subscriber: subscriber,
		publisher:  publisher,
		subAddr:    subAddr,
		pubAddr:    pubAddr,
	}, nil
}

// Start initializes and starts the ZeroMQ manager
func (zmq *ZeroMQManager) Start(ctx context.Context, wg *sync.WaitGroup) error {
	defer wg.Done()

	// Connect subscriber
	err := zmq.subscriber.Connect(zmq.subAddr)
	if err != nil {
		log.Printf("Error connecting ZeroMQ subscriber: %v", err)
		return err
	}

	// Subscribe to all messages (empty string means subscribe to all)
	err = zmq.subscriber.SetSubscribe("")
	if err != nil {
		log.Printf("Error setting ZeroMQ subscriber: %v", err)
		return err
	}

	// Bind publisher
	err = zmq.publisher.Bind(zmq.pubAddr)
	if err != nil {
		log.Printf("Error binding ZeroMQ subscriber: %v", err)
		return err
	}

	log.Printf("ZeroMQ subscriber connected to: %s", zmq.subAddr)
	log.Printf("ZeroMQ publisher bound to: %s", zmq.pubAddr)

	return nil
}

type ZmqMsgHandler interface {
	handle(msgFrames []string) error
}

// Listen starts listening for ZeroMQ messages
func (zmq *ZeroMQManager) Listen(msgHandler ZmqMsgHandler) {
	go func() {
		for {
			msgFrames, err := zmq.subscriber.RecvMessage(0)
			if err != nil {
				log.Printf("Error receiving ZeroMQ message: %v", err)
				continue
			}

			if 0 < len(msgFrames) {
				msgHandler.handle(msgFrames)
			}
		}
	}()
}

// PublishMessage publishes a message to ZeroMQ
func (zmq *ZeroMQManager) PublishMessage(frames []string) error {
	_, err := zmq.publisher.SendMessage(frames)
	if err != nil {
		log.Printf("Error publishing ZeroMQ message: %v", err)
		return err
	}
	log.Printf("Published to ZeroMQ: %s", frames)
	return nil
}
