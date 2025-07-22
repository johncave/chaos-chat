package main

import (
	"container/list"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/ollama/ollama/api" // Ensure you have the Ollama API package installed
)

var globalConn *websocket.Conn
var userName string
var mu sync.Mutex // Mutex to protect globalConn

// Message represents a chat message
type Message struct {
	Type string `json:"type"` // Type of message (e.g., "chat", "typing")
	User string `json:"user"` // Username of the sender
	Text string `json:"text"` // Content of the message

}

// Bot represents the chat bot
type Bot struct {
	LastMessages *list.List // Keeps track of the last 10 messages
}

// NewBot initializes a new Bot instance
func NewBot() *Bot {
	return &Bot{
		LastMessages: list.New(),
	}
}

// AddMessage adds a message to the bot's history
func (b *Bot) AddMessage(msg Message) {
	if b.LastMessages.Len() >= 15 {
		b.LastMessages.Remove(b.LastMessages.Front()) // Remove the oldest message
	}
	b.LastMessages.PushBack(msg)
}

// GetLastMessages returns the last 10 messages as a slice
func (b *Bot) GetLastMessages() []Message {
	var messages []Message
	for e := b.LastMessages.Front(); e != nil; e = e.Next() {
		messages = append(messages, e.Value.(Message))
	}
	return messages
}

// AskLLM queries the Ollama LLM for a response
func (b *Bot) AskLLM(prompt string) (string, error) {
	ctx := context.Background()
	messages := []api.Message{
		{
			Role:    "system",
			Content: "You are a bot participating in a chat service called Chaos chat. You will respond to messages with short, chaotic responses.",
		},
	}

	for _, msg := range b.GetLastMessages() {
		messages = append(messages, api.Message{
			Role:    "user",
			Content: fmt.Sprintf("%s: %s", msg.User, msg.Text),
		})
	}

	log.Println("Generating response...")
	responseContent := ""
	responseFunc := func(resp api.ChatResponse) error {

		responseContent += resp.Message.Content
		go SendTypingUpdate(responseContent) // Simulate sending typing updates to the chat server
		return nil
	}

	chatRequest := &api.ChatRequest{
		Model:    "llama3.2:1b", // Replace with the appropriate model name
		Messages: messages,
	}

	client, err := api.ClientFromEnvironment()
	if err != nil {
		log.Fatal(err)
	} // Replace with the actual Ollama API endpoint
	err = client.Chat(ctx, chatRequest, responseFunc)
	if err != nil {
		log.Printf("Error querying LLM: %v", err)
		return "", err
	}

	return responseContent, nil
}

// SendTypingUpdate simulates sending typing updates to the chat server
func SendTypingUpdate(content string) {
	//fmt.Println("Typing update:", content)
	msg := Message{
		Type: "typing",
		User: userName, // Assuming the bot is the sender
		Text: content,
	}

	//fmt.Println("Typing update:", msg)
	sendMessage(msg) // Send the typing update to the WebSocket server
	//log.Println("Typing update sent:", content)
}

// HandleMessage processes incoming messages and generates a response
func (b *Bot) HandleMessage(message Message) {
	msg := Message{
		Type: message.Type,
		User: message.User,
		Text: message.Text,
	}
	b.AddMessage(msg)

	// Prepare the prompt for the LLM
	lastMessages := b.GetLastMessages()
	var prompt strings.Builder
	for _, m := range lastMessages {
		prompt.WriteString(fmt.Sprintf("%s: %s\n", m.User, m.Text))
	}
	//prompt.WriteString(fmt.Sprintf("Bot: %s", content))

	// Get the response from the LLM
	response, err := b.AskLLM(prompt.String())
	if err != nil {
		log.Println("Error querying LLM:", err)
		return
	}

	// Simulate sending the response to the chat server
	//fmt.Println("Bot response:", response)

	msg = Message{
		Type: "message",
		User: userName, // Assuming the bot is the sender
		Text: response,
	}

	sendMessage(msg)
	log.Println("Response Sent", msg.Text)

	msg.User = "YOU"
	b.AddMessage(msg) // Add the bot's response to the history

	time.Sleep(300 * time.Millisecond)
	msg.Type = "typing" // Send the message to the WebSocket server
	msg.Text = ""

	sendMessage(msg) // Send the typing update to the WebSocket server to close any open typing state
}

func main() {
	bot := NewBot()

	go SubscribeToWebSocket(bot, "wss://chaos.myhackathon.app/ws") // Replace with your WebSocket URL

	select {}
	// // Example usage
	// bot.HandleMessage("User1", "Hello, bot!")
	// bot.HandleMessage("User2", "How are you?")
}

func sendMessage(msg Message) {
	mu.Lock()         // Lock the mutex to protect globalConn
	defer mu.Unlock() // Ensure the mutex is unlocked after sending
	if globalConn != nil {
		err := globalConn.WriteJSON(msg)
		if err != nil {
			log.Printf("Error sending message: %v", err)
		} else {
			//log.Println("Message sent", msg.Text)
		}
	} else {
		log.Println("Global connection is nil, cannot send message")
	}
}

// SubscribeToWebSocket listens for incoming messages on a WebSocket
func SubscribeToWebSocket(bot *Bot, wsURL string) {
	u, err := url.Parse(wsURL)
	if err != nil {
		log.Fatalf("Invalid WebSocket URL: %v", err)
	}

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatalf("Failed to connect to WebSocket: %v", err)
	}
	defer conn.Close()

	globalConn = conn // Store the connection globally for use in other functions

	log.Println("Connected to WebSocket server")

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("Error reading WebSocket message: %v", err)
			os.Exit(1)
			break
		}

		//log.Printf("Received message: %s", message)

		// Process the message (assuming JSON format with sender and content)
		//var sender, content string

		var msg Message // Define a struct to parse the incoming message
		err = json.Unmarshal(message, &msg)
		if err != nil {
			log.Printf("Error parsing message: %v", err)
			continue
		}
		// Parse the message here (e.g., using json.Unmarshal)
		// For simplicity, assume sender and content are extracted

		// Handle the message
		if msg.Type == "message" && msg.User != userName {
			bot.HandleMessage(msg)
		} else if msg.Type == "welcome" {
			userName = msg.User // Store the username from the welcome message
		}

		//bot.HandleMessage(sender, content)
	}
}
