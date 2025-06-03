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
)

func main() {
	if len(os.Args) < 2 {
		programName := os.Args[0]
		fmt.Println("Error: Missing shared memory name argument.")
		fmt.Printf("Usage: %s <shared_memory_name>\n", programName)
		os.Exit(1)
	}
	sharedMemoryName := os.Args[1]

	tempDir := os.TempDir()
	sharedMemoryDir := filepath.Join(tempDir, sharedMemoryName)
	log.Println("Using shared memory file:", sharedMemoryDir)

	file, err := os.OpenFile((sharedMemoryDir),
		// Open the file in read-only mode
		os.O_RDONLY,
		// Set newly created file permissions to read-only for the owner
		0600)

	if err != nil {
		fmt.Println("Failed to open file:", err)
		os.Exit(1)
	}

	defer file.Close()

	data, err := mmap.MmapFileToRead(file, 4)
	if err != nil {
		fmt.Println("Failed to memory-map the file:", err)
		os.Exit(1)
	}

	// dispose
	defer mmap.MunmapFile(data)

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

	port := binary.LittleEndian.Uint32(data)
	fmt.Println("Port number read from shared memory:", port)
	fmt.Println("Press Ctrl+C to exit.")

	<-done
	fmt.Println("Exiting.")
}
