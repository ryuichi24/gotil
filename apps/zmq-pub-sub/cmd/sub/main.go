package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	zmq "github.com/pebbe/zmq4"
)

func main() {
	// Define port flag
	serverPortPtr := flag.Int("port", 8080, "Port number to run the server on")

	// Parse flags
	flag.Parse()

	// Validate port range
	serverPort := *serverPortPtr
	if serverPort < 1 || serverPort > 65535 {
		log.Fatalf("Invalid port number: %d. Port must be between 1 and 65535.", serverPort)
	}

	// Get topic as a required positional argument
	args := flag.Args()
	if len(args) < 1 {
		log.Fatalf("Usage: %s [options] <topic>", os.Args[0])
	}
	topicFilter := args[0]

	// Create a new SUB socket
	subscriber, err := zmq.NewSocket(zmq.SUB)
	if err != nil {
		log.Fatalf("Failed to create subscriber socket: %v", err)
	}
	defer subscriber.Close()

	// Connect to the publisher
	addr := fmt.Sprintf("tcp://localhost:%d", serverPort)
	err = subscriber.Connect(addr)
	if err != nil {
		log.Fatalf("Failed to connect to publisher: %v", err)
	}

	// Subscribe to the topic
	err = subscriber.SetSubscribe(topicFilter)
	if err != nil {
		log.Fatalf("Failed to subscribe: %v", err)
	}
	fmt.Printf("Subscribing to topic: %s\n", topicFilter)
	fmt.Println("Subscriber started on", addr)

	// Receive messages
	for {
		msgs, err := subscriber.RecvMessage(0)
		if err != nil {
			log.Printf("Failed to receive topic: %v", err)
			continue
		}

		log.Printf("Received message: %v", msgs)
	}
}
