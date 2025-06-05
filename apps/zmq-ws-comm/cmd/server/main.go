package main

import (
	"encoding/json"
	"log"
	"net/http"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	gorillaWS "github.com/gorilla/websocket"
	ws "github.com/ryuichi24/zmq-ws-comm/pkg/websocket"
)

func main() {
	r := gin.Default()

	// apiBaseRouter := r.Group("/api")

	wscm := ws.NewWebSocketConnectionManager()

	wscm.EventRegistry.On("example:event", func(evt ws.Event) error {
		log.Printf("Handling example_event for WSockId: %s", evt.WSockId)
		return nil
	},
	)

	wscm.Start()

	// websocket route
	upgrader := gorillaWS.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow connections from any origin
		},
	}

	r.GET("ws", func(c *gin.Context) {
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			c.String(http.StatusInternalServerError, "Failed to upgrade connection: %v", err)
			return
		}

		wSock := ws.NewWSocket(conn)

		// init a new goroutine to handle "each" connection
		go func() {
			defer func() { wscm.DisConnect <- wSock }()

			for {
				_, msg, err := conn.ReadMessage()
				if err != nil {
					c.String(http.StatusInternalServerError, "Failed to read message: %v", err)
					break
				}

				if json.Valid(msg) {
					log.Printf("Received Json message: %s", msg)
					var evt ws.Event

					if err := json.Unmarshal(msg, &evt); err != nil {
						log.Println("JSON unmarshal error:", err)
						break
					}

					evt.WSockId = wSock.Id

					wscm.ReceiveEvt <- evt
					continue
				}

				if utf8.Valid(msg) {
					log.Printf("Received UTF-8 message: %s", msg)
					continue
				}

				log.Printf("Received raw message: %s", msg)
			}
		}()
	})

	log.Println("Server starting on :8080")
	r.Run(":8080")
}
