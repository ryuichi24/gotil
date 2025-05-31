package main

import (
	"fmt"
	"log"
	"os"

	zmq "github.com/pebbe/zmq4"
)

func main() {
	// Get topic filter from command line argument
	topicFilter := ""
	if len(os.Args) > 1 {
		topicFilter = os.Args[1]
	}

	// Create a new SUB socket
	subscriber, err := zmq.NewSocket(zmq.SUB)
	if err != nil {
		log.Fatalf("Failed to create subscriber socket: %v", err)
	}
	defer subscriber.Close()

	// Connect to the publisher
	err = subscriber.Connect("tcp://localhost:5555")
	if err != nil {
		log.Fatalf("Failed to connect to publisher: %v", err)
	}

	// Subscribe to specific topic or all topics
	if topicFilter != "" {
		err = subscriber.SetSubscribe(topicFilter)
		fmt.Printf("Subscribing to topic: %s\n", topicFilter)
	} else {
		err = subscriber.SetSubscribe("")
		fmt.Println("Subscribing to all topics")
	}
	if err != nil {
		log.Fatalf("Failed to subscribe: %v", err)
	}

	fmt.Println("Subscriber started, waiting for messages...")

	// Receive messages
	for {
		// Receive topic first
		topic, err := subscriber.Recv(0)
		if err != nil {
			log.Printf("Failed to receive topic: %v", err)
			continue
		}

		// Receive message
		message, err := subscriber.Recv(0)
		if err != nil {
			log.Printf("Failed to receive message: %v", err)
			continue
		}

		fmt.Printf("Received on topic '%s': %s\n", topic, message)
	}
}
