package websocket

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/fintech-multipass/backend/internal/domain/entity"
	"github.com/fintech-multipass/backend/internal/infrastructure/logger"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// En producción, validar origen
		return true
	},
}

// Hub mantiene el conjunto de clientes activos y broadcasting de mensajes
type Hub struct {
	// Clientes registrados
	clients map[*Client]bool

	// Clientes por país (para filtrar mensajes)
	clientsByCountry map[uuid.UUID]map[*Client]bool

	// Canal para registrar clientes
	register chan *Client

	// Canal para desregistrar clientes
	unregister chan *Client

	// Canal para broadcasting de mensajes
	broadcast chan *Message

	// Mutex para operaciones thread-safe
	mu sync.RWMutex

	// Logger
	log *logger.Logger
}

// Client representa una conexión WebSocket
type Client struct {
	hub        *Hub
	conn       *websocket.Conn
	send       chan []byte
	userID     uuid.UUID
	countryIDs []uuid.UUID
	role       entity.UserRole
}

// Message representa un mensaje para broadcasting
type Message struct {
	Type       string      `json:"type"`
	Data       interface{} `json:"data"`
	CountryID  *uuid.UUID  `json:"country_id,omitempty"`
	TargetUser *uuid.UUID  `json:"target_user,omitempty"`
	Timestamp  time.Time   `json:"timestamp"`
}

// Tipos de mensajes
const (
	MessageTypeApplicationCreated  = "application_created"
	MessageTypeApplicationUpdated  = "application_updated"
	MessageTypeStatusChanged       = "status_changed"
	MessageTypeNotification        = "notification"
	MessageTypePing                = "ping"
	MessageTypePong                = "pong"
)

// NewHub crea un nuevo Hub
func NewHub(log *logger.Logger) *Hub {
	return &Hub{
		clients:          make(map[*Client]bool),
		clientsByCountry: make(map[uuid.UUID]map[*Client]bool),
		register:         make(chan *Client),
		unregister:       make(chan *Client),
		broadcast:        make(chan *Message),
		log:              log,
	}
}

// Run inicia el loop principal del Hub
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.registerClient(client)

		case client := <-h.unregister:
			h.unregisterClient(client)

		case message := <-h.broadcast:
			h.broadcastMessage(message)
		}
	}
}

func (h *Hub) registerClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.clients[client] = true

	// Registrar por país
	for _, countryID := range client.countryIDs {
		if h.clientsByCountry[countryID] == nil {
			h.clientsByCountry[countryID] = make(map[*Client]bool)
		}
		h.clientsByCountry[countryID][client] = true
	}

	h.log.Info().
		Str("user_id", client.userID.String()).
		Int("total_clients", len(h.clients)).
		Msg("Client connected")
}

func (h *Hub) unregisterClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.clients[client]; ok {
		delete(h.clients, client)
		close(client.send)

		// Desregistrar de países
		for _, countryID := range client.countryIDs {
			if clients, ok := h.clientsByCountry[countryID]; ok {
				delete(clients, client)
				if len(clients) == 0 {
					delete(h.clientsByCountry, countryID)
				}
			}
		}

		h.log.Info().
			Str("user_id", client.userID.String()).
			Int("total_clients", len(h.clients)).
			Msg("Client disconnected")
	}
}

func (h *Hub) broadcastMessage(message *Message) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	data, err := json.Marshal(message)
	if err != nil {
		h.log.Error().Err(err).Msg("Failed to marshal message")
		return
	}

	// Si hay un usuario específico
	if message.TargetUser != nil {
		for client := range h.clients {
			if client.userID == *message.TargetUser {
				select {
				case client.send <- data:
				default:
					// Buffer lleno, cliente lento
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
		return
	}

	// Si hay un país específico
	if message.CountryID != nil {
		if clients, ok := h.clientsByCountry[*message.CountryID]; ok {
			for client := range clients {
				select {
				case client.send <- data:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
		return
	}

	// Broadcast a todos
	for client := range h.clients {
		select {
		case client.send <- data:
		default:
			close(client.send)
			delete(h.clients, client)
		}
	}
}

// Broadcast envía un mensaje a todos los clientes
func (h *Hub) Broadcast(msgType string, data interface{}) {
	h.broadcast <- &Message{
		Type:      msgType,
		Data:      data,
		Timestamp: time.Now(),
	}
}

// BroadcastToCountry envía un mensaje a clientes de un país específico
func (h *Hub) BroadcastToCountry(countryID uuid.UUID, msgType string, data interface{}) {
	h.broadcast <- &Message{
		Type:      msgType,
		Data:      data,
		CountryID: &countryID,
		Timestamp: time.Now(),
	}
}

// SendToUser envía un mensaje a un usuario específico
func (h *Hub) SendToUser(userID uuid.UUID, msgType string, data interface{}) {
	h.broadcast <- &Message{
		Type:       msgType,
		Data:       data,
		TargetUser: &userID,
		Timestamp:  time.Now(),
	}
}

// HandleWebSocket maneja una nueva conexión WebSocket
func HandleWebSocket(hub *Hub, c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		hub.log.Error().Err(err).Msg("Failed to upgrade connection")
		return
	}

	// Obtener información del usuario del contexto (si está autenticado)
	var userID uuid.UUID
	var countryIDs []uuid.UUID
	var role entity.UserRole

	if uid, exists := c.Get("user_id"); exists {
		userID = uid.(uuid.UUID)
	} else {
		userID = uuid.New() // Usuario anónimo
	}

	if cids, exists := c.Get("country_ids"); exists {
		countryIDs = cids.([]uuid.UUID)
	}

	if r, exists := c.Get("user_role"); exists {
		role = r.(entity.UserRole)
	}

	client := &Client{
		hub:        hub,
		conn:       conn,
		send:       make(chan []byte, 256),
		userID:     userID,
		countryIDs: countryIDs,
		role:       role,
	}

	hub.register <- client

	// Iniciar goroutines de lectura y escritura
	go client.writePump()
	go client.readPump()
}

// readPump lee mensajes del WebSocket
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(512)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.hub.log.Error().Err(err).Msg("Unexpected close error")
			}
			break
		}

		// Procesar mensaje del cliente
		var msg Message
		if err := json.Unmarshal(message, &msg); err == nil {
			// Manejar ping
			if msg.Type == MessageTypePing {
				response := Message{
					Type:      MessageTypePong,
					Timestamp: time.Now(),
				}
				data, _ := json.Marshal(response)
				c.send <- data
			}
		}
	}
}

// writePump escribe mensajes al WebSocket
func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Enviar mensajes pendientes en el buffer
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
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

