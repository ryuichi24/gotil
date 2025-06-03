package port

import "net"

// FindFreePort finds and returns a free TCP port on the system.
// It returns the port number and any error encountered.
func FindFreePort() (int, error) {
	// Listen on port 0, which tells the OS to assign a free port
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()

	// Extract the port from the listener's address
	addr := listener.Addr().(*net.TCPAddr)
	return addr.Port, nil
}
