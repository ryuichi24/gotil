package network

import (
	"net"
)

// GetFreePort asks the kernel for a free open port that is ready to use.
func FindFreePort() (int, error) {
	// Listen on a random port
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()

	// Retrieve the port assigned
	addr := listener.Addr().(*net.TCPAddr)
	return addr.Port, nil
}
