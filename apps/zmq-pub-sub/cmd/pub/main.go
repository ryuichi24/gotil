package main

import (
	"fmt"
	"log"
	"time"

	zmq "github.com/pebbe/zmq4"
	"github.com/ryuichi24/zmq-pub-sub/pkg/network"
)

func main() {
	// Create a new PUB socket
	publisher, err := zmq.NewSocket(zmq.PUB)
	if err != nil {
		log.Fatalf("Failed to create publisher socket: %v", err)
	}
	defer publisher.Close()

	freePort, err := network.FindFreePort()
	if err != nil {
		log.Fatalf("Failed to find free port: %v", err)
	}

	addr := fmt.Sprintf("tcp://localhost:%d", freePort)
	// Bind the socket to a port
	err = publisher.Bind(addr)
	if err != nil {
		log.Fatalf("Failed to bind publisher socket: %v", err)
	}

	fmt.Printf("Publisher started on %s\n", addr)
	time.Sleep(time.Second) // Give subscribers time to connect

	// Topics to publish to
	topics := []string{"news", "weather", "sports"}
	topicIndex := 0

	// Send messages
	for i := 1; ; i++ {
		// Rotate through topics
		topic := topics[topicIndex]
		topicIndex = (topicIndex + 1) % len(topics)

		// Create message with topic
		message := fmt.Sprintf("%s: Message %d", topic, i)

		// the first frame is reserved for the topic
		frames := []string{topic, message}

		_, err := publisher.SendMessage(frames)
		if err != nil {
			log.Printf("Failed to send topic: %v", err)
			continue
		}

		fmt.Printf("Published to %s: %s\n", topic, message)
		time.Sleep(time.Second)
	}
}
