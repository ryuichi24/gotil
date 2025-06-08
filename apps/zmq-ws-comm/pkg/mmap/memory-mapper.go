package mmap

import (
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
)

const (
	// Integer sizes (in bytes)
	Int8Size  = 1
	Uint8Size = 1
	Int16Size = 2
	// 0 to 65535 (e.g. port numbers)
	Uint16Size = 2
	Int32Size  = 4
	Uint32Size = 4
	Int64Size  = 8
	Uint64Size = 8

	// Floating-point sizes (in bytes)
	Float32Size = 4
	Float64Size = 8

	// Complex number sizes (in bytes)
	// Complex64 = float32 real + float32 imag = 8 bytes
	// Complex128 = float64 real + float64 imag = 16 bytes
	Complex64Size  = 8
	Complex128Size = 16

	// Byte and rune are aliases
	ByteSize = Uint8Size
	RuneSize = Int32Size
)

type MemoryMapper struct {
	id      string
	size    int
	memPath string
}

func NewMemoryMapper(id string, size int) *MemoryMapper {
	tempDir := os.TempDir()
	sharedMemoryPath := filepath.Join(tempDir, id)
	log.Println("Using shared memory file:", sharedMemoryPath)

	return &MemoryMapper{
		id:      id,
		size:    size,
		memPath: sharedMemoryPath,
	}
}

func (m *MemoryMapper) Write(data []byte) error {
	// Create or open the file with read-write permissions
	file, err := os.OpenFile(m.memPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		log.Fatalf("Failed to open or create file: %v", err)
	}
	defer file.Close()

	if err := file.Truncate(int64(m.size)); err != nil {
		log.Fatalf("Failed to truncate file: %v", err)
	}

	mappedMem, err := MmapFileToWrite(file, m.size)
	if err != nil {
		fmt.Println("Failed to memory-map the file:", err)
		os.Exit(1)
	}

	// dispose
	defer MunmapFile(mappedMem)

	if len(data) > m.size {
		return fmt.Errorf("data size %d exceeds memory size %d", len(data), m.size)
	}

	copy(mappedMem, data)

	return nil
}

func (m *MemoryMapper) Read() ([]byte, error) {
	file, err := os.OpenFile(m.memPath, os.O_RDONLY, 0600)
	if err != nil {
		log.Printf("Failed to open shared memory file: %v", err)
		return nil, err
	}
	defer file.Close()

	mappedMem, err := MmapFileToRead(file, m.size)
	if err != nil {
		log.Printf("Failed to memory-map the file: %v", err)
		return nil, err
	}
	defer MunmapFile(mappedMem)

	buffer := make([]byte, m.size)
	copy(buffer, mappedMem)

	return buffer, nil
}

func (m *MemoryMapper) Dispose() error {
	if err := os.Remove(m.memPath); err != nil {
		log.Printf("Failed to remove shared memory file: %v", err)
		return err
	}
	log.Println("Shared memory file removed:", m.memPath)
	return nil
}

// getters
func (m *MemoryMapper) MemPath() string {
	return m.memPath
}

// IntConverter provides methods to convert between int and byte arrays

type IntConverter struct {
	byteOrder binary.ByteOrder
}

func NewIntConverter(byteOrder binary.ByteOrder) *IntConverter {
	return &IntConverter{
		byteOrder: byteOrder,
	}
}

// convert int to byte array

func (c *IntConverter) ToUint16Bytes(value int) ([]byte, error) {
	if value < 0 || value > math.MaxUint16 {
		return nil, fmt.Errorf("value %d out of range for uint16", value)
	}
	buf := make([]byte, Uint16Size)
	c.byteOrder.PutUint16(buf, uint16(value))
	return buf, nil
}

func (c *IntConverter) ToUint32Bytes(value int) ([]byte, error) {
	if value < 0 || value > math.MaxUint32 {
		return nil, fmt.Errorf("value %d out of range for uint32", value)
	}
	buf := make([]byte, Uint32Size)
	c.byteOrder.PutUint32(buf, uint32(value))
	return buf, nil
}

func (c *IntConverter) ToUint64Bytes(value int) ([]byte, error) {
	if value < 0 {
		return nil, fmt.Errorf("value %d out of range for uint64", value)
	}
	buf := make([]byte, Uint64Size)
	c.byteOrder.PutUint64(buf, uint64(value))
	return buf, nil
}

// convert byte array to int
func (c *IntConverter) FromUint16Bytes(buf []byte) (int, error) {
	if len(buf) != Uint16Size {
		return 0, fmt.Errorf("buffer size %d does not match uint16 size %d", len(buf), Uint16Size)
	}
	val := c.byteOrder.Uint16(buf)
	return int(val), nil
}

func (c *IntConverter) FromUint32Bytes(buf []byte) (int, error) {
	if len(buf) != Uint32Size {
		return 0, fmt.Errorf("buffer size %d does not match uint32 size %d", len(buf), Uint32Size)
	}
	val := c.byteOrder.Uint32(buf)
	return int(val), nil
}

func (c *IntConverter) FromUint64Bytes(buf []byte) (int, error) {
	if len(buf) != Uint64Size {
		return 0, fmt.Errorf("buffer size %d does not match uint64 size %d", len(buf), Uint64Size)
	}
	val := c.byteOrder.Uint64(buf)
	if val > math.MaxInt64 {
		return 0, fmt.Errorf("value %d out of range for int", val)
	}
	return int(val), nil
}

func (c *IntConverter) FromBytes(buf []byte, size int) (int, error) {
	switch size {
	case Uint16Size:
		return c.FromUint16Bytes(buf)
	case Uint32Size:
		return c.FromUint32Bytes(buf)
	case Uint64Size:
		return c.FromUint64Bytes(buf)
	default:
		return 0, fmt.Errorf("unsupported size: %d", size)
	}
}
