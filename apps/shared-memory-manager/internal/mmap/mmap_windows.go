//go:build windows
// +build windows

package mmap

import (
	"os"
	"syscall"
	"unsafe"
)

func MmapFile(file *os.File, size int) ([]byte, error) {
	handle, err := syscall.CreateFileMapping(syscall.Handle(file.Fd()), nil, syscall.PAGE_READONLY, 0, uint32(size), nil)
	if err != nil {
		return nil, err
	}
	defer syscall.CloseHandle(handle)

	addr, err := syscall.MapViewOfFile(handle, syscall.FILE_MAP_READ, 0, 0, uintptr(size))
	if err != nil {
		return nil, err
	}

	// Convert to byte slice
	data := (*[1 << 30]byte)(unsafe.Pointer(addr))[:size:size]
	return data, nil
}

func MunmapFile(data []byte) error {
	return syscall.UnmapViewOfFile(uintptr(unsafe.Pointer(&data[0])))
}
