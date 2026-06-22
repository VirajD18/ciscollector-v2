package main

import (
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}
// WebSocketHandler upgrades an HTTP connection and registers the client
func (app *App) WebSocketHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("Failed to upgrade websocket:", err)
		return
	}

	client := app.WSHub.AddClient(conn)
	defer app.WSHub.RemoveClient(client)

	fmt.Println("WS client connected")

	// Keep the connection alive (frontend can send pings or ignore messages)
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}