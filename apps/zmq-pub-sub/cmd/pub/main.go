package main

import (
	"fmt"
	"log"
	"time"

	zmq "github.com/pebbe/zmq4"
)

func main() {
	// Create a new PUB socket
	publisher, err := zmq.NewSocket(zmq.PUB)
	if err != nil {
		log.Fatalf("Failed to create publisher socket: %v", err)
	}
	defer publisher.Close()

	// Bind the socket to a port
	err = publisher.Bind("tcp://*:5555")
	if err != nil {
		log.Fatalf("Failed to bind publisher socket: %v", err)
	}

	fmt.Println("Publisher started, waiting for subscribers...")
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
