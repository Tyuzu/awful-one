package discord

import (
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/julienschmidt/httprouter"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(*http.Request) bool { return true },
}

func WebSocketHandler(hubs *HubManager) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		room := ps.ByName("room")
		user := r.Context().Value("userID").(string)

		conn, _ := upgrader.Upgrade(w, r, nil)

		client := &Client{
			ID:     user + ":" + room,
			UserID: user,
			RoomID: room,
			Conn:   conn,
			Send:   make(chan Event, 64),
		}

		hub := hubs.ForRoom(room)
		hub.register <- client
		hub.Join(client.ID, room)

		go client.WritePump()
		go client.ReadPump(hub)
	}
}
