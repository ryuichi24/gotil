package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type Event struct {
	WSockId string          `json:"wSockId"`
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type BCEvent struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type WSocket struct {
	Id   string
	Conn *websocket.Conn
}

func NewWSocket(conn *websocket.Conn) *WSocket {
	return &WSocket{
		Id:   uuid.New().String(),
		Conn: conn,
	}
}

type EventHandler func(evt Event) error

type EventHandlerRegistry struct {
	handlers map[string][]EventHandler
	mutex    sync.RWMutex
}

func NewEventHandlerRegistry() *EventHandlerRegistry {
	return &EventHandlerRegistry{
		handlers: make(map[string][]EventHandler),
	}
}

// Registers an event handler for a specific event type
func (ehr *EventHandlerRegistry) on(eventType string, handler EventHandler) {
	ehr.mutex.Lock()
	defer ehr.mutex.Unlock()

	ehr.handlers[eventType] = append(ehr.handlers[eventType], handler)
	log.Printf("Registered handler for event type: %s", eventType)
}

// Removes all handlers for a specific event type
func (ehr *EventHandlerRegistry) off(eventType string) {
	ehr.mutex.Lock()
	defer ehr.mutex.Unlock()

	delete(ehr.handlers, eventType)
	log.Printf("Unregistered all handlers for event type: %s", eventType)
}

// Returns all handlers for a specific event type
func (ehr *EventHandlerRegistry) GetHandlers(eventType string) []EventHandler {
	ehr.mutex.RLock()
	defer ehr.mutex.RUnlock()

	handlers, exists := ehr.handlers[eventType]
	if !exists {
		return nil
	}

	// Return a copy to prevent concurrent modification
	result := make([]EventHandler, len(handlers))
	copy(result, handlers)
	return result
}

// Returns all registered event types
func (ehr *EventHandlerRegistry) ListEventTypes() []string {
	ehr.mutex.RLock()
	defer ehr.mutex.RUnlock()

	types := make([]string, 0, len(ehr.handlers))
	for eventType := range ehr.handlers {
		types = append(types, eventType)
	}
	return types
}

type WebSocketConnManager struct {
	conns map[string]*websocket.Conn

	connect    chan *WSocket
	disConnect chan *WSocket

	receiveEvt chan Event
	broadcast  chan BCEvent

	mutex         sync.RWMutex
	eventRegistry *EventHandlerRegistry
}

func (wscm *WebSocketConnManager) addConn(wSock *WSocket) {
	wscm.mutex.Lock()
	defer wscm.mutex.Unlock()

	if _, has := wscm.conns[wSock.Id]; !has {
		wscm.conns[wSock.Id] = wSock.Conn
	}
}

func (wscm *WebSocketConnManager) removeConn(wSock *WSocket) {
	wscm.mutex.Lock()
	defer wscm.mutex.Unlock()

	if _, has := wscm.conns[wSock.Id]; has {
		delete(wscm.conns, wSock.Id)
		wSock.Conn.Close()
	}
}

func (wscm *WebSocketConnManager) handleEvt(evt Event) {
	handlers := wscm.eventRegistry.GetHandlers(evt.Type)

	if len(handlers) == 0 {
		log.Printf("No handlers registered for event type: %s", evt.Type)
		return
	}

	// Execute all handlers
	for i, handler := range handlers {
		go func(index int, h EventHandler, event Event) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("Handler %d for event type %s panicked: %v", index, event.Type, r)
				}
			}()

			if err := h(event); err != nil {
				log.Printf("Handler %d for event type %s returned error: %v", index, event.Type, err)
			}
		}(i, handler, evt)
	}
}

func (wscm *WebSocketConnManager) Start() {
	go func() {
		for {
			select {
			case wSock := <-wscm.connect:
				wscm.addConn(wSock)
				log.Printf("New WebSocket connection established: %v", wSock.Conn.RemoteAddr())

			case wSock := <-wscm.disConnect:
				wscm.removeConn(wSock)
				log.Printf("WebSocket connection closed: %v", wSock.Conn.RemoteAddr())

			case evt := <-wscm.receiveEvt:
				log.Printf("Received event: %s from %s", evt.Type, evt.WSockId)
				wscm.handleEvt(evt)

			case bdEvent := <-wscm.broadcast:
				log.Printf("Broadcasting event: %s", bdEvent.Type)
			}
		}
	}()
}

func NewWebSocketConnectionManager() *WebSocketConnManager {
	return &WebSocketConnManager{
		conns:         make(map[string]*websocket.Conn),
		connect:       make(chan *WSocket),
		disConnect:    make(chan *WSocket),
		receiveEvt:    make(chan Event),
		broadcast:     make(chan BCEvent),
		eventRegistry: NewEventHandlerRegistry(),
	}
}

func main() {
	r := gin.Default()

	// apiBaseRouter := r.Group("/api")

	wscm := NewWebSocketConnectionManager()

	wscm.eventRegistry.on("example:event", func(evt Event) error {
		log.Printf("Handling example_event for WSockId: %s", evt.WSockId)
		return nil
	},
	)

	wscm.Start()

	// websocket route
	upgrader := websocket.Upgrader{
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

		wSock := NewWSocket(conn)

		// init a new goroutine to handle "each" connection
		go func() {
			defer func() { wscm.disConnect <- wSock }()

			for {
				_, msg, err := conn.ReadMessage()
				if err != nil {
					c.String(http.StatusInternalServerError, "Failed to read message: %v", err)
					break
				}

				if json.Valid(msg) {
					log.Printf("Received Json message: %s", msg)
					var evt Event

					if err := json.Unmarshal(msg, &evt); err != nil {
						log.Println("JSON unmarshal error:", err)
						break
					}

					evt.WSockId = wSock.Id

					wscm.receiveEvt <- evt
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
