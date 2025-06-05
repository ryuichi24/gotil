package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pebbe/zmq4"
	"github.com/ryuichi24/zmq-ws-comm/pkg/mmap"
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

	freePort, err := network.FindFreePort()
	if err != nil {
		log.Fatalf("Failed to find a free port: %v", err)
	}
	log.Printf("Found free port: %d", freePort)

	buffer := new(bytes.Buffer)
	// port is presented in 16 bits, so we use uint16
	if err := binary.Write(buffer, binary.LittleEndian, uint16(freePort)); err != nil {
		log.Fatalf("Failed to write port to buffer: %v", err)
	}
	memWriter := NewUint16MemWriter(binary.LittleEndian)
	memReader := NewUint16MemReader(binary.LittleEndian)

	memMapper := NewMemMapper("example_shared_memory", 2, memWriter, memReader)
	defer memMapper.Dispose()
	if err := memMapper.Write(buffer.Bytes()); err != nil {
		log.Fatalf("Failed to write port to shared memory: %v", err)
	}
	zmqPubAddr := fmt.Sprintf("tcp://localhost:%d", freePort)
	// Find a bound port of the zmq publisher
	memMapper2 := NewMemMapper("example_shared_memory2", 2, memWriter, memReader)

	retryConfig := util.RetryConfig{
		MaxAttempts:  5,
		InitialDelay: 2000 * time.Millisecond,
		MaxDelay:     2 * time.Second,
	}
	portBuffer, err := util.Retry(ctx, retryConfig, memMapper2.Read)
	if err != nil {
		log.Fatalf("Failed to read port from shared memory: %v", err)
	}
	fmt.Printf("Port read from shared memory: %d\n", portBuffer)
	pubPort := binary.LittleEndian.Uint16(portBuffer)

	log.Printf("Port read from shared memory: %d", pubPort)
	zmqSubAddr := fmt.Sprintf("tcp://localhost:%d", pubPort)

	// ZeroMQ manager setup
	zmqManager, err := NewZeroMQManager(zmqSubAddr, zmqPubAddr)
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

func WritePortToSharedMemory(port int) error {
	sharedMemoryName := "example_shared_memory"

	tempDir := os.TempDir()
	sharedMemoryPath := filepath.Join(tempDir, sharedMemoryName)
	log.Println("Using shared memory file:", sharedMemoryPath)

	// Create or open the file with read-write permissions
	file, err := os.OpenFile(sharedMemoryPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		log.Fatalf("Failed to open or create file: %v", err)
	}
	defer file.Close()

	// Ensure the file is 4 bytes long
	if err := file.Truncate(4); err != nil {
		log.Fatalf("Failed to truncate file: %v", err)
	}

	// Memory-map the file
	data, err := mmap.MmapFileToWrite(file, 4)
	if err != nil {
		fmt.Println("Failed to memory-map the file:", err)
		os.Exit(1)
	}

	// dispose
	defer mmap.MunmapFile(data)

	// Write the port number in little-endian format
	binary.LittleEndian.PutUint32(data, uint32(port))
	log.Printf("Port number %d written to shared memory.\n", port)
	fmt.Println("Press Ctrl+C to exit.")

	return nil
}

type MemWriter interface {
	Write(mappedMem []byte, data []byte) error
}

type Uint16MemWriter struct {
	Order binary.ByteOrder
}

func NewUint16MemWriter(order binary.ByteOrder) *Uint16MemWriter {
	return &Uint16MemWriter{
		Order: order,
	}
}

func (w *Uint16MemWriter) Write(mappedMem []byte, data []byte) error {
	if len(data) > len(mappedMem) {
		return fmt.Errorf("data length %d exceeds available space %d", len(data), len(mappedMem)-2)
	}
	if len(data) < 2 {
		return fmt.Errorf("data length %d is less than 2 bytes", len(data))
	}
	// make sure the data length is exactly 2 bytes
	uint16Data := w.Order.Uint16(data)
	w.Order.PutUint16(mappedMem, uint16Data)
	return nil
}

type MemReader interface {
	Read(mappedMem []byte) ([]byte, error)
}

type Uint16MemReader struct {
	Order binary.ByteOrder
}

func NewUint16MemReader(order binary.ByteOrder) *Uint16MemReader {
	return &Uint16MemReader{
		Order: order,
	}
}

func (r *Uint16MemReader) Read(mappedMem []byte) ([]byte, error) {
	buffer := make([]byte, 2)
	r.Order.PutUint16(buffer, r.Order.Uint16(mappedMem))

	if len(buffer) == 0 {
		return nil, fmt.Errorf("mapped memory is too small: %d bytes", len(buffer))
	}

	return buffer, nil
}

type MemMapper struct {
	MemName string
	maxSize int
	// Write the port number in little-endian format
	// binary.LittleEndian.PutUint32(mappedMem, uint32(port))
	writer MemWriter
	reader MemReader
}

func NewMemMapper(memName string, size int, writer MemWriter, reader MemReader) *MemMapper {
	return &MemMapper{
		MemName: memName,
		maxSize: size,
		writer:  writer,
		reader:  reader,
	}
}

func (mm *MemMapper) Write(data []byte) error {
	if mm.maxSize < len(data) {
		return fmt.Errorf("data length %d exceeds maximum size %d", len(data), mm.maxSize)
	}

	tempDir := os.TempDir()
	sharedMemoryPath := filepath.Join(tempDir, mm.MemName)
	log.Println("Using shared memory file:", sharedMemoryPath)

	// Create or open the file with read-write permissions
	file, err := os.OpenFile(sharedMemoryPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		log.Fatalf("Failed to open or create file: %v", err)
	}
	defer file.Close()

	// Ensure the file is 4 bytes long
	if err := file.Truncate(int64(mm.maxSize)); err != nil {
		log.Fatalf("Failed to truncate file: %v", err)
	}

	// Memory-map the file
	mappedMem, err := mmap.MmapFileToWrite(file, mm.maxSize)
	if err != nil {
		fmt.Println("Failed to memory-map the file:", err)
		os.Exit(1)
	}

	// dispose
	defer mmap.MunmapFile(mappedMem)

	if err := mm.writer.Write(mappedMem, data); err != nil {
		return fmt.Errorf("failed to write data to shared memory: %w", err)
	}

	log.Printf("%d bytes are written to the shared memory.\n", len(data))

	return nil
}

func (mm *MemMapper) Read() ([]byte, error) {
	tempDir := os.TempDir()
	sharedMemoryPath := filepath.Join(tempDir, mm.MemName)
	file, err := os.OpenFile(sharedMemoryPath, os.O_RDONLY, 0600)
	if err != nil {
		log.Printf("Failed to open shared memory file: %v", err)
		return nil, err
	}
	defer file.Close()
	mappedMem, err := mmap.MmapFileToRead(file, mm.maxSize)
	if err != nil {
		log.Printf("Failed to memory-map the file: %v", err)
		return nil, err
	}
	defer mmap.MunmapFile(mappedMem)
	if len(mappedMem) < 2 {
		log.Printf("Mapped memory is too small: %d bytes", len(mappedMem))
		return nil, fmt.Errorf("mapped memory is too small: %d bytes", len(mappedMem))
	}

	data, err := mm.reader.Read(mappedMem)
	if err != nil {
		log.Printf("Failed to read data from shared memory: %v", err)
		return nil, fmt.Errorf("failed to read data from shared memory: %w", err)
	}
	log.Printf("%d bytes are read from the shared memory.\n", len(data))

	return data, nil
}

func (mm *MemMapper) Dispose() error {
	tempDir := os.TempDir()
	sharedMemoryPath := filepath.Join(tempDir, mm.MemName)
	if err := os.Remove(sharedMemoryPath); err != nil {
		log.Printf("Failed to remove shared memory file: %v", err)
		return fmt.Errorf("failed to remove shared memory file: %w", err)
	}
	log.Printf("Shared memory file %s removed successfully.", sharedMemoryPath)
	return nil
}
