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
	mongoClient   *mongo.Client
	pgClient      *gorm.DB
	chatColl      *mongo.Collection
	chatUsersColl *mongo.Collection // New collection to cache usernames
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
	SenderName string             `bson:"sender_name" json:"sender_name"` // Added to decouple from User DB
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
	conn     *websocket.Conn
	userID   string
	userName string // Added to keep track of the user's name during the session
	send     chan []byte
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
	
	// Initialize Collections
	db := mongoClient.Database("chat_db")
	chatColl = db.Collection("messages")
	chatUsersColl = db.Collection("users") // Initialize user cache collection
	
	fmt.Println("Connected to MongoDB!")

	// PostgreSQL
	pgURI := os.Getenv("DB_URI")
	if pgURI == "" {
		pgURI = "postgres://admin:password@localhost:5432/core_db"
	}
	pgDb, err := gorm.Open(postgres.Open(pgURI), &gorm.Config{})
	if err != nil {
		log.Println("Warning: PostgreSQL connection error (Chat will rely on MongoDB cache): ", err)
	} else {
		pgClient = pgDb
		fmt.Println("Connected to PostgreSQL!")
	}
}

// REST: Get history between two users
func historyHandler(w http.ResponseWriter, r *http.Request) {
	// Add CORS headers for direct frontend access if not using Gateway for this
	w.Header().Set("Access-Control-Allow-Origin", "*")
	
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

func contactsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "missing user_id", http.StatusBadRequest)
		return
	}

	// Find all messages where the user is either the sender or receiver
	filter := bson.M{
		"$or": []bson.M{
			bson.M{"sender_id": userID},
			bson.M{"receiver_id": userID},
		},
	}

	cursor, err := chatColl.Find(context.TODO(), filter)
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

	// Extract unique partners
	partnerMap := make(map[string]string) // Map ID to Name
	for _, msg := range messages {
		if msg.SenderID != userID {
			partnerMap[msg.SenderID] = msg.SenderName
		}
		if msg.ReceiverID != userID {
			// If they were only a receiver, we might not have their name in the message struct yet
			if _, exists := partnerMap[msg.ReceiverID]; !exists {
				partnerMap[msg.ReceiverID] = "" 
			}
		}
	}

	// Build the response array, resolving missing names via the cache
	type Contact struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	var contacts []Contact

	for pID, pName := range partnerMap {
		if pName == "" {
			var cachedUser struct {
				Name string `bson:"name"`
			}
			err := chatUsersColl.FindOne(context.TODO(), bson.M{"_id": pID}).Decode(&cachedUser)
			if err == nil {
				pName = cachedUser.Name
			} else {
				pName = "Unknown User" // Fallback
			}
		}
		contacts = append(contacts, Contact{ID: pID, Name: pName})
	}

	if contacts == nil {
		contacts = []Contact{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(contacts)
}

// WebSocket handler
func wsHandler(w http.ResponseWriter, r *http.Request) {
	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "user_id is required", http.StatusBadRequest)
		return
	}

	var userName string
	var user User

	// 1. Try to verify user exists in PostgreSQL
	if pgClient != nil {
		result := pgClient.Table("users").First(&user, "id = ?", userID)
		if result.Error == nil {
			userName = user.Name

			// UPSERT to MongoDB to cache the username
			opts := options.Update().SetUpsert(true)
			update := bson.M{"$set": bson.M{"name": userName}}
			_, err := chatUsersColl.UpdateByID(context.TODO(), userID, update, opts)
			if err != nil {
				log.Println("MongoDB user cache update error:", err)
			}
		}
	}

	// 2. Fallback: If PG is down or user wasn't found, check MongoDB Cache
	if userName == "" {
		var cachedUser struct {
			Name string `bson:"name"`
		}
		err := chatUsersColl.FindOne(context.TODO(), bson.M{"_id": userID}).Decode(&cachedUser)
		if err != nil {
			http.Error(w, "User not found in Core DB or Chat Cache", http.StatusUnauthorized)
			return
		}
		userName = cachedUser.Name
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("ws upgrade error:", err)
		return
	}

	client := &Client{
		conn:     conn,
		userID:   userID,
		userName: userName, // Store the cached name in the client session
		send:     make(chan []byte, 256),
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
			msg.SenderName = client.userName // Attach the saved username directly to the payload
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
	http.HandleFunc("/api/v1/chat/history", historyHandler)
	http.HandleFunc("/api/v1/chat/contacts", contactsHandler)
	
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