package main

import (
	"encoding/json"
	"fmt"
	"log"

	// "net/http"
	"sync"

	"github.com/gorilla/websocket"
)

// WSMessage is broadcast to dashboard WebSocket clients (collector fleet events).
type WSMessage struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload,omitempty"`
}

// Client represents a single WebSocket connection
type Client struct {
	conn *websocket.Conn
	send chan WSMessage
}

// Hub manages all connected clients
type Hub struct {
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	broadcast  chan WSMessage
	mu         sync.Mutex
}

// NewHub creates a new Hub
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan WSMessage, 32),
	}
}

// Run starts the hub's main loop
func (h *Hub) Run() {
	fmt.Println("Starting Run for WS Hub(line 152,ws_hub.go)...")
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
			fmt.Println("Client registered")

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				fmt.Println("Client unregistered")
			}
			h.mu.Unlock()

		case msg := <-h.broadcast:
			h.mu.Lock()
			for client := range h.clients {
				select {
				case client.send <- msg:
				default:
					// client not reading, remove
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mu.Unlock()
		}
	}
}

// AddClient registers a new client and starts its write pump
func (h *Hub) AddClient(conn *websocket.Conn) *Client {
	fmt.Println("(line 88, ws_hub.go) Adding a client...")
	client := &Client{
		conn: conn,
		send: make(chan WSMessage, 16), // buffered channel
	}

	h.register <- client

	go client.WritePump()
	return client
}

// RemoveClient unregisters a client
func (h *Hub) RemoveClient(client *Client) {
	h.unregister <- client
}

// BroadcastMessage sends a message to all clients
func (h *Hub) BroadcastMessage(msg WSMessage) {
	h.broadcast <- msg
}

// WritePump sends messages to the WebSocket connection
func (c *Client) WritePump() {
	defer c.conn.Close()
	for msg := range c.send {
		fmt.Println("Msg in ws_hub.go(line 114) : ", msg)
		data, err := json.Marshal(msg)
		if err != nil {
			log.Println("Failed to marshal WS message:", err)
			continue
		}
		if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
			log.Println("WebSocket write error:", err)
			return
		}
	}
}
