package game

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gofiber/contrib/websocket"
)

type Client struct {
	conn   *websocket.Conn
	userID string
	mu     sync.Mutex
}

type Hub struct {
	clients    map[*Client]bool
	broadcast  chan interface{}
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan interface{}, 100),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			log.Printf("[WS] Client connected: %s (Total: %d)", client.userID, len(h.clients))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				client.conn.Close()
				log.Printf("[WS] Client disconnected: %s (Total: %d)", client.userID, len(h.clients))
			}
			h.mu.Unlock()

		case message := <-h.broadcast:
			jsonMessage, err := json.Marshal(message)
			if err != nil {
				log.Printf("[WS] Marshal error: %v", err)
				continue
			}

			h.mu.RLock()
			for client := range h.clients {
				go client.send(jsonMessage) // Non-blocking send
			}
			h.mu.RUnlock()
		}
	}
}

func (h *Hub) Broadcast(message interface{}) {
	select {
	case h.broadcast <- message:
	default:
		log.Println("[WS] Broadcast channel full, dropping message")
	}
}


func (h *Hub) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

func (c *Client) send(message interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var data []byte
	var err error

	switch v := message.(type) {
	case []byte:
		data = v
	default:
		data, err = json.Marshal(v)
		if err != nil {
			log.Printf("[WS] Send marshal error: %v", err)
			return
		}
	}

	c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
		log.Printf("[WS] Write error for user %s: %v", c.userID, err)
	}
}

func (c *Client) SendInitialState(state *RoundState) {
	if state != nil {
		c.send(map[string]interface{}{
			"type": "initial_state",
			"data": state,
		})
	}
}

func (h *Hub) RegisterClient(conn *websocket.Conn, userID string) {
	client := &Client{
		conn:   conn,
		userID: userID,
	}
	h.register <- client
}

func (h *Hub) UnregisterClient(conn *websocket.Conn) {
	h.mu.RLock()
	for client := range h.clients {
		if client.conn == conn {
			h.mu.RUnlock()
			h.unregister <- client
			return
		}
	}
	h.mu.RUnlock()
}
