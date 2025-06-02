package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/ryuichi24/shared-memory-manager/internal/mmap"
)

const (
	SHARED_MEMO_NAME = "qt_shared_memory"
)

func main() {
	fileName := SHARED_MEMO_NAME
	tempDir := os.TempDir()

	fileName = filepath.Join(tempDir, fileName)
	log.Println("Using shared memory file:", fileName)

	// Open the file
	file, err := os.OpenFile(fileName, os.O_RDONLY, 0600)
	if err != nil {
		fmt.Println("Failed to open file:", err)
		return
	}
	defer file.Close()

	// Memory-map the file (cross-platform)
	data, err := mmap.MmapFile(file, 4)
	if err != nil {
		fmt.Println("Failed to memory-map the file:", err)
		return
	}
	defer func() {
		mmap.MunmapFile(data)
		fmt.Println("Memory unmapped.")
	}()

	// Setup signal handling for cleanup
	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigs
		fmt.Println()
		fmt.Println("Received signal:", sig)
		done <- true
	}()

	// Read the port number (little-endian)
	port := uint32(data[0]) | uint32(data[1])<<8 | uint32(data[2])<<16 | uint32(data[3])<<24
	fmt.Println("Port number read from shared memory:", port)
	fmt.Println("Press Ctrl+C to exit.")

	<-done
	fmt.Println("Exiting.")
}
