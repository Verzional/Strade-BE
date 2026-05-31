package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	mongoClient *mongo.Client
	pgClient    *gorm.DB
	chatColl    *mongo.Collection
)

// User represents the existing PostgreSQL user schema (read-only)
type User struct {
	ID    string `gorm:"primaryKey;type:uuid"`
	Name  string
	Email string `gorm:"unique"`
}

// Message represents the MongoDB schema for chat
type Message struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	SenderID   string             `bson:"sender_id" json:"sender_id"`
	ReceiverID string             `bson:"receiver_id" json:"receiver_id"`
	Content    string             `bson:"content" json:"content"`
	CreatedAt  time.Time          `bson:"created_at" json:"created_at"`
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // allow all origins for dev
	},
}

type Client struct {
	conn   *websocket.Conn
	userID string
	send   chan []byte
}

var (
	clients    = make(map[string]*Client)
	clientsMux sync.Mutex
)

// Init DB connections
func initDB() {
	// MongoDB
	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://admin:password@localhost:27017/chat_db?authSource=admin"
	}
	clientOptions := options.Client().ApplyURI(mongoURI)
	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		log.Fatal("MongoDB connection error: ", err)
	}
	err = client.Ping(context.TODO(), nil)
	if err != nil {
		log.Fatal("MongoDB ping error: ", err)
	}
	mongoClient = client
	chatColl = mongoClient.Database("chat_db").Collection("messages")
	fmt.Println("Connected to MongoDB!")

	// PostgreSQL
	pgURI := os.Getenv("DB_URI")
	if pgURI == "" {
		pgURI = "postgres://admin:password@localhost:5432/core_db"
	}
	db, err := gorm.Open(postgres.Open(pgURI), &gorm.Config{})
	if err != nil {
		log.Fatal("PostgreSQL connection error: ", err)
	}
	pgClient = db
	fmt.Println("Connected to PostgreSQL!")
}

// REST: Get history between two users
func historyHandler(w http.ResponseWriter, r *http.Request) {
	senderID := r.URL.Query().Get("sender_id")
	receiverID := r.URL.Query().Get("receiver_id")

	if senderID == "" || receiverID == "" {
		http.Error(w, "missing sender_id or receiver_id", http.StatusBadRequest)
		return
	}

	filter := bson.M{
		"$or": []bson.M{
			bson.M{"sender_id": senderID, "receiver_id": receiverID},
			bson.M{"sender_id": receiverID, "receiver_id": senderID},
		},
	}

	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}})
	cursor, err := chatColl.Find(context.TODO(), filter, opts)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer cursor.Close(context.TODO())

	var messages []Message
	if err = cursor.All(context.TODO(), &messages); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if messages == nil {
		messages = []Message{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}

// WebSocket handler
func wsHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "user_id is required", http.StatusBadRequest)
		return
	}

	// Verify user exists in PostgreSQL
	var user User
	result := pgClient.Table("users").First(&user, "id = ?", userID)
	if result.Error != nil {
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("ws upgrade error:", err)
		return
	}

	client := &Client{
		conn:   conn,
		userID: userID,
		send:   make(chan []byte, 256),
	}

	clientsMux.Lock()
	clients[userID] = client
	clientsMux.Unlock()

	// Write pump
	go func() {
		defer conn.Close()
		for msg := range client.send {
			if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		}
	}()

	// Read pump
	defer func() {
		clientsMux.Lock()
		delete(clients, userID)
		clientsMux.Unlock()
		conn.Close()
	}()

	for {
		_, messagePayload, err := conn.ReadMessage()
		if err != nil {
			break
		}

		var msg Message
		err = json.Unmarshal(messagePayload, &msg)
		if err == nil {
			msg.SenderID = userID
			msg.CreatedAt = time.Now()
			msg.ID = primitive.NewObjectID()

			// Save to MongoDB
			_, err = chatColl.InsertOne(context.TODO(), msg)
			if err != nil {
				log.Println("mongodb insert error:", err)
				continue
			}

			// Route to receiver if online
			clientsMux.Lock()
			if receiverClient, ok := clients[msg.ReceiverID]; ok {
				responseMsg, _ := json.Marshal(msg)
				receiverClient.send <- responseMsg
			}
			clientsMux.Unlock()
		}
	}
}

func main() {
	initDB()

	http.HandleFunc("/ws", wsHandler)
	http.HandleFunc("/api/v1/chat/history", historyHandler) // Updated to follow standard structure
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Chat Service Running")
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	fmt.Println("Chat Server is running on port", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
