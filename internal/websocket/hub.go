package websocket

import (
	"encoding/json"
	"sync"

	"github.com/google/uuid"
)

type WsMessage struct {
	EventType string      `json:"event_type"`
	Payload   interface{} `json:"payload"`
}

type internalMessage struct {
	ProjectID uuid.UUID
	Message   WsMessage
}

// Tambahkan interface agar bisa di-inject ke usecase
type Hub interface {
	Run()
	BroadcastToProject(projectID uuid.UUID, eventType string, payload interface{})
	Register(client *Client)
	Unregister(client *Client)
}

type hub struct {
	rooms      map[uuid.UUID]map[*Client]bool
	broadcast  chan internalMessage
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

func NewHub() Hub {
	return &hub{
		rooms:      make(map[uuid.UUID]map[*Client]bool),
		broadcast:  make(chan internalMessage, 256), // buffered agar tidak blocking
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

func (h *hub) Register(client *Client) {
	h.register <- client
}

func (h *hub) Unregister(client *Client) {
	h.unregister <- client
}

func (h *hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			if h.rooms[client.projectID] == nil {
				h.rooms[client.projectID] = make(map[*Client]bool)
			}
			h.rooms[client.projectID][client] = true
			h.mu.Unlock()

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.rooms[client.projectID][client]; ok {
				delete(h.rooms[client.projectID], client)
				close(client.send)
				if len(h.rooms[client.projectID]) == 0 {
					delete(h.rooms, client.projectID)
				}
			}
			h.mu.Unlock()

		case msg := <-h.broadcast:
			h.mu.Lock() // Ganti RLock -> Lock karena ada operasi delete
			if clients, ok := h.rooms[msg.ProjectID]; ok {
				jsonMsg, _ := json.Marshal(msg.Message)
				for client := range clients {
					select {
					case client.send <- jsonMsg:
					default:
						close(client.send)
						delete(clients, client)
					}
				}
				if len(clients) == 0 {
					delete(h.rooms, msg.ProjectID)
				}
			}
			h.mu.Unlock()
		}
	}
}

func (h *hub) BroadcastToProject(projectID uuid.UUID, eventType string, payload interface{}) {
	h.broadcast <- internalMessage{
		ProjectID: projectID,
		Message: WsMessage{
			EventType: eventType,
			Payload:   payload,
		},
	}
}
