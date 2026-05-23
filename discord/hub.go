package discord

type Hub struct {
	clients    map[string]*Client
	rooms      map[string]map[string]bool
	register   chan *Client
	unregister chan *Client
	broadcast  chan Event
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[string]*Client),
		rooms:      make(map[string]map[string]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan Event, 512),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case c := <-h.register:
			h.clients[c.ID] = c

		case c := <-h.unregister:
			delete(h.clients, c.ID)
			close(c.Send)

		case evt := <-h.broadcast:
			if subs, ok := h.rooms[evt.RoomID]; ok {
				for id := range subs {
					if c, ok := h.clients[id]; ok {
						select {
						case c.Send <- evt:
						default:
						}
					}
				}
			}
		}
	}
}

func (h *Hub) Join(clientID, roomID string) {
	if _, ok := h.rooms[roomID]; !ok {
		h.rooms[roomID] = map[string]bool{}
	}
	h.rooms[roomID][clientID] = true
}

func (h *Hub) Emit(evt Event) {
	h.broadcast <- evt
}
