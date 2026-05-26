package handlers

import (
	"encoding/json"
	"family-manager/backend/db"
	"family-manager/backend/middleware"
	"family-manager/backend/models"
	"family-manager/backend/utils"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Разрешаем все источники для разработки
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type Hub struct {
	clients    map[*Client]bool
	broadcast  chan *MessageBroadcast
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

type Client struct {
	conn     *websocket.Conn
	familyID int
	userID   int
	username string
	send     chan []byte
	hub      *Hub
}

type MessageBroadcast struct {
	FamilyID int
	Message  models.Message
}

var hub = &Hub{
	clients:    make(map[*Client]bool),
	broadcast:  make(chan *MessageBroadcast),
	register:   make(chan *Client),
	unregister: make(chan *Client),
}

func StartHub() {
	log.Println("Starting chat hub...")
	for {
		select {
		case client := <-hub.register:
			hub.mu.Lock()
			hub.clients[client] = true
			hub.mu.Unlock()
			log.Printf("Client registered. Total clients: %d", len(hub.clients))

		case client := <-hub.unregister:
			hub.mu.Lock()
			if _, ok := hub.clients[client]; ok {
				delete(hub.clients, client)
				close(client.send)
			}
			hub.mu.Unlock()
			log.Printf("Client unregistered. Total clients: %d", len(hub.clients))

		case msg := <-hub.broadcast:
			hub.mu.RLock()
			for client := range hub.clients {
				if client.familyID == msg.FamilyID {
					select {
					case client.send <- encodeMessage(msg.Message):
					default:
						close(client.send)
						delete(hub.clients, client)
					}
				}
			}
			hub.mu.RUnlock()
		}
	}
}

func encodeMessage(msg models.Message) []byte {
	data, _ := json.Marshal(msg)
	return data
}

func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) readPump() {
	defer func() {
		hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(512)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		var msg models.Message
		err := c.conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Сохраняем сообщение в БД
		msg.UserID = c.userID
		msg.FamilyID = c.familyID
		msg.Username = c.username

		err = db.DB.QueryRow(`
            INSERT INTO messages (family_id, user_id, username, message)
            VALUES ($1, $2, $3, $4) 
            RETURNING id, created_at
        `, c.familyID, c.userID, c.username, msg.Message).Scan(&msg.ID, &msg.CreatedAt)

		if err != nil {
			log.Printf("Error saving message: %v", err)
			continue
		}

		// Отправляем сообщение всем в семье
		hub.broadcast <- &MessageBroadcast{
			FamilyID: c.familyID,
			Message:  msg,
		}
	}
}

func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	log.Println("WebSocket connection attempt")

	// Получаем токен из query параметра
	token := r.URL.Query().Get("token")
	if token == "" {
		// Пробуем из заголовка
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" {
			token = strings.TrimPrefix(authHeader, "Bearer ")
		}
	}

	if token == "" {
		log.Println("No token provided")
		utils.SendJSONError(w, "No authorization token", http.StatusUnauthorized)
		return
	}

	// Валидируем токен
	claims, err := middleware.ValidateToken(token)
	if err != nil {
		log.Printf("Token validation error: %v", err)
		utils.SendJSONError(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	log.Printf("Token validated - UserID: %d, FamilyID: %d, Role: %s",
		claims.UserID, claims.FamilyID, claims.Role)

	// Получаем username
	var username string
	err = db.DB.QueryRow("SELECT username FROM users WHERE id = $1", claims.UserID).Scan(&username)
	if err != nil {
		log.Printf("Error getting username: %v", err)
		utils.SendJSONError(w, "User not found", http.StatusNotFound)
		return
	}

	// Upgrade connection
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	// Создаем клиента
	client := &Client{
		conn:     conn,
		familyID: claims.FamilyID,
		userID:   claims.UserID,
		username: username,
		send:     make(chan []byte, 256),
		hub:      hub,
	}

	// Регистрируем клиента
	hub.register <- client

	// Запускаем горутины для клиента
	go client.writePump()
	go client.readPump()

	log.Printf("WebSocket client connected: User=%s, Family=%d", username, claims.FamilyID)
}

// GetMessages - получение истории сообщений (REST API)
func GetMessages(w http.ResponseWriter, r *http.Request) {
	familyID := r.Context().Value("family_id").(int)

	log.Printf("Getting messages for family %d", familyID)

	rows, err := db.DB.Query(`
        SELECT m.id, m.user_id, u.username, m.message, m.created_at
        FROM messages m
        JOIN users u ON m.user_id = u.id
        WHERE m.family_id = $1
        ORDER BY m.created_at DESC LIMIT 100
    `, familyID)

	if err != nil {
		log.Printf("Database error: %v", err)
		utils.SendJSONError(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var messages []models.Message
	for rows.Next() {
		var msg models.Message
		err := rows.Scan(&msg.ID, &msg.UserID, &msg.Username, &msg.Message, &msg.CreatedAt)
		if err != nil {
			log.Printf("Error scanning message: %v", err)
			continue
		}
		messages = append(messages, msg)
	}

	// Reverse to get chronological order (oldest first)
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	log.Printf("Returning %d messages", len(messages))
	utils.SendJSONResponse(w, messages, http.StatusOK)
}
