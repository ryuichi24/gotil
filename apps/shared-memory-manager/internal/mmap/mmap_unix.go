//go:build !windows
// +build !windows

package mmap

import (
	"os"
	"syscall"
)

func MmapFile(file *os.File, size int) ([]byte, error) {
	data, err := syscall.Mmap(int(file.Fd()), 0, size, syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func MunmapFile(data []byte) error {
	return syscall.Munmap(data)
}
