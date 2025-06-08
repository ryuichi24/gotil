package main

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/pebbe/zmq4"
	"github.com/ryuichi24/zmq-ws-comm/pkg/mmap"
	"github.com/ryuichi24/zmq-ws-comm/pkg/network"
	"github.com/ryuichi24/zmq-ws-comm/pkg/util"
)

const (
	ZMQ_PUB_MMEM_NAME = "ZMQ_PUB_MMEM_NAME"
	ZMQ_EXTERNAL_PUB_MMEM_NAME = "ZMQ_EXTERNAL_PUB_MMEM_NAME"
)

func main() {
	serverPortPtr := flag.Int("port", 8080, "Port number to run the server on")
	flag.Parse()

	serverPort := *serverPortPtr
	if serverPort < 1 || serverPort > 65535 {
		log.Fatalf("Invalid port number: %d. Port must be between 1 and 65535.", serverPort)
	}

	// init context to manage multiple goroutines
	ctx, cancel := context.WithCancel(context.Background())

	// init wait group to wait for goroutines to finish
	var wg sync.WaitGroup
	wg.Add(3)
	// start goroutines and pass ctx to each

	zmqPublisher, err := NewZMQPublisher()
	if err != nil {
		log.Fatalf("Failed to create ZMQPublisher: %v", err)
	}

	zmqSubscriber, err := NewZMQSubscriber()
	if err != nil {
		log.Fatalf("Failed to create ZMQSubscriber: %v", err)
	}

	zmqSubscriber.On("message", func(messages []string) error {
		log.Printf("Received messages: %v\n", messages)
		// Here you would implement the logic to handle incoming messages
		return nil
	})

	server := NewServer(fmt.Sprintf(":%d", serverPort), zmqPublisher, zmqSubscriber)
	server.SetupRouter(func(router *gin.Engine, ctx *ServerCtx) {
		// cors
		router.Use(cors.New(cors.Config{
			AllowOrigins:     []string{"*"},
			AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
			AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
			ExposeHeaders:    []string{"Content-Length"},
			AllowCredentials: true,
			MaxAge:           12 * time.Hour,
		}))

		// web socket endpoint
		router.GET("/ws", func(c *gin.Context) {})

		// REST API endpoints
		restRouter := router.Group("/api")
		restRouter.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"zmq_publisher_ready":  ctx.zmqPub.IsReady(),
				"zmq_subscriber_ready": ctx.zmqSub.IsReady(),
			})
		})

		// RPC endpoints
		rpcRouter := router.Group("/rpc")
		rpcRouter.POST("/publish-zmq-message", func(c *gin.Context) {
			var request struct {
				Topic   string `json:"topic" binding:"required"`
				Event   string
				Payload json.RawMessage `json:"payload"`
			}

			if err := c.ShouldBindJSON(&request); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			if err := ctx.zmqPub.Publish(request.Topic, request.Event, request.Payload); err != nil {
				log.Printf("Failed to publish message: %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to publish message"})
				return
			}

			// Handle publish ZMQ message
			c.JSON(200, gin.H{"status": "message published"})
		})
	})

	go zmqPublisher.Start(ctx, &wg)
	go zmqSubscriber.Start(ctx, &wg)
	go server.Start(ctx, &wg)

	// init signal channel for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)

	<-sigCh
	log.Println("Received shutdown signal...")

	cancel()  // cancel context to stop all goroutines
	wg.Wait() // wait for all goroutines to finish

	log.Println("All goroutines finished. Exiting...")
}

// ZMQPublisher
type ZMQPublisher struct {
	ready bool
	sock  *zmq4.Socket
}

func NewZMQPublisher() (*ZMQPublisher, error) {
	sock, err := zmq4.NewSocket(zmq4.PUB)
	if err != nil {
		return nil, err
	}

	return &ZMQPublisher{
		ready: false,
		sock:  sock,
	}, nil
}

func (p *ZMQPublisher) Start(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	defer p.Shutdown()

	// find a free port for the publisher
	freePort, err := network.FindFreePort()
	if err != nil {
		log.Fatalf("Failed to find a free port: %v", err)
	}

	addr := fmt.Sprintf("tcp://localhost:%d", freePort)

	if err := p.sock.Bind(addr); err != nil {
		log.Fatalf("Failed to bind ZMQ socket: %v", err)
	}

	p.ready = true

	log.Printf("ZMQPublisher started and bound to %s\n", addr)

	// Create a memory mapper to store the bound port
	zmqPubBoundPortMme := mmap.NewMemoryMapper(ZMQ_PUB_MMEM_NAME, mmap.Uint16Size)
	defer zmqPubBoundPortMme.Dispose()
	beIntConverter := mmap.NewIntConverter(binary.BigEndian)

	pubBoundPortBytes, err := beIntConverter.ToUint16Bytes(freePort)

	if err != nil {
		log.Fatalf("Failed to convert bound port to bytes: %v", err)
	}

	if err := zmqPubBoundPortMme.Write(pubBoundPortBytes); err != nil {
		log.Fatalf("Failed to write bound port to shared memory: %v", err)
	}

	log.Printf("ZMQPublisher bound port %d written to shared memory at %s\n", freePort, zmqPubBoundPortMme.MemPath())

	<-ctx.Done()
	log.Println("Background worker stopping due to context cancellation...")
}

func (p *ZMQPublisher) Shutdown() {
	log.Println("Shutting down ZMQPublisher...")
	if err := p.sock.Close(); err != nil {
		log.Printf("Error closing ZMQ publisher socket: %v", err)
	}
	log.Println("ZMQPublisher shutdown complete.")
	p.ready = false
}

func (p *ZMQPublisher) Publish(topic string, event string, payload json.RawMessage) error {
	log.Printf("Publishing message to topic: %s, event: %s, payload: %s\n", topic, event, payload)
	// Here you would implement the logic to publish the message to ZMQ
	return nil
}

func (p *ZMQPublisher) IsReady() bool {
	return p.ready
}

// ZMQSubscriber

type ZMQSubscriber struct {
	ready       bool
	sock        *zmq4.Socket
	listeners   map[string]Listener
	OnReceive   chan []string
	OnConnected chan struct{}
	OnShutdown  chan struct{}
}

func NewZMQSubscriber() (*ZMQSubscriber, error) {
	sock, err := zmq4.NewSocket(zmq4.SUB)
	if err != nil {
		return nil, err
	}

	return &ZMQSubscriber{
		ready:       false,
		sock:        sock,
		listeners:   make(map[string]Listener),
		OnReceive:   make(chan []string), // Buffered channel for received messages
		OnConnected: make(chan struct{}), // Channel to signal connection established
		OnShutdown:  make(chan struct{}),
	}, nil

}

func (s *ZMQSubscriber) Start(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	defer s.Shutdown()

	log.Println("Starting ZMQSubscriber...")

	zmqPubBoundPortMme := mmap.NewMemoryMapper(ZMQ_EXTERNAL_PUB_MMEM_NAME, mmap.Uint16Size)
	defer zmqPubBoundPortMme.Dispose()

	// retry logic
	retryConfig := util.RetryConfig{
		MaxAttempts:  50,
		InitialDelay: 2000 * time.Millisecond,
		MaxDelay:     2 * time.Second,
	}

	pubBoundPortBytes, err := util.Retry(ctx, retryConfig, zmqPubBoundPortMme.Read)

	if err != nil {
		log.Fatalf("Failed to read bound port from shared memory: %v", err)
	}

	pubBoundPort, err := mmap.NewIntConverter(binary.BigEndian).FromUint16Bytes(pubBoundPortBytes)

	if err != nil {
		log.Fatalf("Failed to convert bound port bytes to uint16: %v", err)
	}
	log.Printf("ZMQSubscriber will connect to publisher on port: %d\n", pubBoundPort)

	addr := fmt.Sprintf("tcp://localhost:%d", pubBoundPort)
	if err := s.sock.Connect(addr); err != nil {
		log.Fatalf("Failed to connect ZMQ subscriber socket: %v", err)
	}

	if err := s.sock.SetSubscribe(""); err != nil {
		log.Fatalf("Failed to set ZMQ subscriber socket to subscribe to all topics: %v", err)
	}

	log.Printf("ZMQSubscriber connected to publisher at %s\n", addr)
	s.ready = true
	close(s.OnConnected) // Signal that the subscriber is ready

	listeningStopped := make(chan struct{})
	go func() {
		defer close(listeningStopped)
		log.Println("ZMQSubscriber is now listening for messages...")
		s.Listen(ctx)
	}()

	<-ctx.Done()
	log.Println("Context cancelled, ZMQSubscriber stopping...")

	<-listeningStopped // Wait for the listening goroutine to finish
}

func (s *ZMQSubscriber) Listen(ctx context.Context) {
	log.Println("ZMQSubscriber is listening for messages...")
	for {
		select {
		case <-ctx.Done():
			{
				log.Println("ZMQSubscriber stopping due to context cancellation...")
				return
			}
		default:
			{
				msgs, err := s.sock.RecvMessage(0)
				if err != nil {
					log.Printf("Error receiving message: %v", err)
					continue
				}

				if len(os.ModeSetgid.Type().String()) == 0 {
					log.Println("Received empty message, skipping...")
					continue
				}

				topic := msgs[0]
				log.Printf("Received message on topic '%s': %v\n", topic, msgs)

				listener, has := s.listeners[topic]
				if !has {
					log.Printf("No listener found for topic '%s', skipping...\n", topic)
				}

				if err := listener(msgs); err != nil {
					log.Printf("Error in listener for topic '%s': %v", topic, err)
				}

				msgListener, has := s.listeners["message"]
				if !has {
					log.Println("No listener found for 'message' topic, skipping...")
					continue
				}

				if err := msgListener(msgs); err != nil {
					log.Printf("Error in 'message' listener: %v", err)
				}
			}
		}
	}
}

type Listener func(messages []string) error

func (s *ZMQSubscriber) On(topic string, listener Listener) error {
	s.listeners[topic] = listener
	return nil
}

func (s *ZMQSubscriber) Shutdown() {
	log.Println("Shutting down ZMQSubscriber...")
	if err := s.sock.Close(); err != nil {
		log.Printf("Error closing ZMQ subscriber socket: %v", err)
	}
	log.Println("ZMQSubscriber shutdown complete.")
	close(s.OnShutdown)
}

func (s *ZMQSubscriber) IsReady() bool {
	return s.ready
}

// Server

type Server struct {
	Addr       string
	HttpServer *http.Server
	router     *gin.Engine
	zmqPub     *ZMQPublisher
	zmqSub     *ZMQSubscriber
}

func NewServer(addr string, zmqPub *ZMQPublisher, zmqSub *ZMQSubscriber) *Server {
	router := gin.Default()

	return &Server{
		Addr:       addr,
		router:     router,
		zmqPub:     zmqPub,
		zmqSub:     zmqSub,
		HttpServer: &http.Server{Addr: addr, Handler: router},
	}
}

func (s *Server) Start(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	defer s.Shutdown()

	go func() {
		log.Printf("Server is running at http://localhost%s\n", s.HttpServer.Addr)
		if err := s.HttpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("Context cancelled, stopping server...")
}

func (s *Server) Shutdown() {
	log.Println("Shutting down server...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	if err := s.HttpServer.Shutdown(shutdownCtx); err != nil {
		log.Println("Server Shutdown:", err)
	}
	log.Println("Server gracefully stopped.")
	<-shutdownCtx.Done()
}

type ServerCtx struct {
	zmqPub *ZMQPublisher
	zmqSub *ZMQSubscriber
}

func newServerCtx(zmqPub *ZMQPublisher, zmqSub *ZMQSubscriber) *ServerCtx {
	return &ServerCtx{
		zmqPub: zmqPub,
		zmqSub: zmqSub,
	}
}

func (s *Server) SetupRouter(routeSetter func(router *gin.Engine, ctx *ServerCtx)) error {
	ctx := newServerCtx(s.zmqPub, s.zmqSub)
	if routeSetter == nil {
		return fmt.Errorf("routeSetter cannot be nil")
	}

	routeSetter(s.router, ctx)

	return nil
}
