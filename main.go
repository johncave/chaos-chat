package main

import (
	"log"
	"net/http"

	"math/rand"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Message represents a chat or typing event
type Message struct {
	Type string `json:"type"`
	User string `json:"user"`
	Text string `json:"text"`
}

type Client struct {
	conn *websocket.Conn
	user string
}

type Hub struct {
	clients    map[*Client]bool
	broadcast  chan Message
	register   chan *Client
	unregister chan *Client
	mu         sync.Mutex
}

var hub = &Hub{
	clients:    make(map[*Client]bool),
	broadcast:  make(chan Message),
	register:   make(chan *Client),
	unregister: make(chan *Client),
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
		case client := <-h.unregister:
			h.mu.Lock()
			delete(h.clients, client)
			h.mu.Unlock()
		case msg := <-h.broadcast:
			h.mu.Lock()
			for c := range h.clients {
				c.conn.WriteJSON(msg)
			}
			h.mu.Unlock()
		}
	}
}

func randomUser() string {
	nouns := []string{"Cat", "Dog", "Fox", "Bear", "Wolf", "Lion", "Tiger", "Owl", "Hawk", "Duck"}
	adjs := []string{"Red", "Blue", "Green", "Yellow", "Fast", "Slow", "Happy", "Sad", "Big", "Small"}
	rand.Seed(time.Now().UnixNano())
	return adjs[rand.Intn(len(adjs))] + nouns[rand.Intn(len(nouns))]
}

func wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("Upgrade error:", err)
		return
	}
	client := &Client{conn: conn, user: randomUser()}
	// Send initial welcome message with assigned username
	conn.WriteJSON(Message{Type: "welcome", User: client.user})
	hub.register <- client
	defer func() {
		hub.unregister <- client
		conn.Close()
	}()
	for {
		var msg Message
		err := conn.ReadJSON(&msg)
		if err != nil {
			break
		}
		switch msg.Type {
		case "message":
			hub.broadcast <- Message{Type: "message", User: client.user, Text: msg.Text}
		case "typing":
			hub.broadcast <- Message{Type: "typing", User: client.user, Text: msg.Text}
		}
	}
}

func main() {
	go hub.run()

	http.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/favicon.ico")
	})

	http.HandleFunc("/ws", wsHandler)
	http.Handle("/", http.FileServer(http.Dir("./static")))
	log.Println("Server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
