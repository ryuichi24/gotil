package websocket

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	gorillaWS "github.com/gorilla/websocket"
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
func (ehr *EventHandlerRegistry) On(eventType string, handler EventHandler) {
	ehr.mutex.Lock()
	defer ehr.mutex.Unlock()

	ehr.handlers[eventType] = append(ehr.handlers[eventType], handler)
	log.Printf("Registered handler for event type: %s", eventType)
}

// Removes all handlers for a specific event type
func (ehr *EventHandlerRegistry) Off(eventType string) {
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
	Conns map[string]*websocket.Conn

	Connect    chan *WSocket
	DisConnect chan *WSocket

	ReceiveEvt chan Event
	Broadcast  chan BCEvent

	mutex         sync.RWMutex
	EventRegistry *EventHandlerRegistry
}

func (wscm *WebSocketConnManager) addConn(wSock *WSocket) {
	wscm.mutex.Lock()
	defer wscm.mutex.Unlock()

	if _, has := wscm.Conns[wSock.Id]; !has {
		wscm.Conns[wSock.Id] = wSock.Conn
	}
}

func (wscm *WebSocketConnManager) removeConn(wSock *WSocket) {
	wscm.mutex.Lock()
	defer wscm.mutex.Unlock()

	if _, has := wscm.Conns[wSock.Id]; has {
		delete(wscm.Conns, wSock.Id)
		wSock.Conn.Close()
	}
}

func (wscm *WebSocketConnManager) handleEvt(evt Event) {
	handlers := wscm.EventRegistry.GetHandlers(evt.Type)

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

func (wscm *WebSocketConnManager) Start(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	go func() {

		defer wscm.Shutdown()

		for {
			select {
			case wSock := <-wscm.Connect:
				wscm.addConn(wSock)
				log.Printf("New WebSocket connection established: %v", wSock.Conn.RemoteAddr())

			case wSock := <-wscm.DisConnect:
				wscm.removeConn(wSock)
				log.Printf("WebSocket connection closed: %v", wSock.Conn.RemoteAddr())

			case evt := <-wscm.ReceiveEvt:
				log.Printf("Received event: %s from %s", evt.Type, evt.WSockId)
				wscm.handleEvt(evt)

			case bdEvent := <-wscm.Broadcast:
				log.Printf("Broadcasting event: %s", bdEvent.Type)

			case <-ctx.Done():
				log.Println("Closing WebSocket due to context cancellation...")
				return
			}
		}
	}()
}

func (wscm *WebSocketConnManager) Shutdown() {
	log.Println("WebSocketConnManager shutting down")

	wscm.mutex.Lock()
	defer wscm.mutex.Unlock()

	for _, conn := range wscm.Conns {
		conn.Close()
		log.Printf("Closed connection: %v", conn.RemoteAddr())
	}
	// why need to reinit this?
	// wscm.Conns = make(map[string]*websocket.Conn)
	log.Println("WebSocketConnManager shutdown complete")
}

func (wscm *WebSocketConnManager) HandleWSConnection(c *gin.Context) {
	// websocket route
	upgrader := gorillaWS.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow connections from any origin
		},
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		c.String(http.StatusInternalServerError, "Failed to upgrade connection: %v", err)
		return
	}

	wSock := NewWSocket(conn)

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
				var evt Event

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

}

func NewWebSocketConnectionManager() *WebSocketConnManager {
	return &WebSocketConnManager{
		Conns:         make(map[string]*websocket.Conn),
		Connect:       make(chan *WSocket),
		DisConnect:    make(chan *WSocket),
		ReceiveEvt:    make(chan Event),
		Broadcast:     make(chan BCEvent),
		EventRegistry: NewEventHandlerRegistry(),
	}
}
