package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/ryuichi24/shared-memory-manager/internal/mmap"
	port "github.com/ryuichi24/shared-memory-manager/internal/network"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("Usage: %s <shared_memory_name>\n", os.Args[0])
		os.Exit(1)
	}

	sharedMemoryName := os.Args[1]

	freePort, err := port.FindFreePort()
	if err != nil {
		log.Fatalf("Failed to find a free port: %v", err)
		os.Exit(1)
	}

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

	// setup signal handling for cleanup
	sigCh := make(chan os.Signal, 1)
	doneCh := make(chan bool, 1)

	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		fmt.Println()
		fmt.Println("Received signal:", sig)
		doneCh <- true
	}()

	// Write the port number in little-endian format
	binary.LittleEndian.PutUint32(data, uint32(freePort))
	log.Printf("Port number %d written to shared memory.\n", freePort)
	fmt.Println("Press Ctrl+C to exit.")

	<-doneCh
	fmt.Println("Exiting writer program.")
}
