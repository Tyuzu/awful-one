package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/julienschmidt/httprouter"
)

// verifyClaims extracts and verifies JWT claims from the request
func (a *App) verifyClaims(r *http.Request) (*CustomClaims, error) {
	token, err := ExtractToken(r)
	if err != nil {
		return nil, err
	}

	claims, err := a.jwtManager.Verify(token)
	if err != nil {
		return nil, err
	}

	return claims, nil
}

func (a *App) createRoom(
	w http.ResponseWriter,
	r *http.Request,
	_ httprouter.Params,
) {
	// Verify JWT token
	_, err := a.verifyClaims(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "unauthorized",
		})
		return
	}

	var input struct {
		Name string `json:"name"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid json",
		})
		return
	}

	input.Name = strings.TrimSpace(input.Name)

	if input.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "room name required",
		})
		return
	}

	a.roomMux.Lock()
	defer a.roomMux.Unlock()

	room := &ChatRoom{
		ID:        a.nextRoomID,
		Name:      input.Name,
		Members:   make(map[int64]bool),
		Messages:  []Message{},
		CreatedAt: time.Now(),
	}

	a.rooms[room.ID] = room
	a.nextRoomID++

	writeJSON(w, http.StatusCreated, room)
}
func (a *App) listRooms(
	w http.ResponseWriter,
	r *http.Request,
	_ httprouter.Params,
) {
	a.roomMux.RLock()
	defer a.roomMux.RUnlock()

	rooms := make([]*ChatRoom, 0, len(a.rooms))

	for _, room := range a.rooms {
		rooms = append(rooms, room)
	}

	writeJSON(w, http.StatusOK, rooms)
}
func (a *App) getRoom(
	w http.ResponseWriter,
	r *http.Request,
	ps httprouter.Params,
) {
	id, err := parseID(ps.ByName("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid room id",
		})
		return
	}

	a.roomMux.RLock()
	defer a.roomMux.RUnlock()

	room, ok := a.rooms[id]
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "room not found",
		})
		return
	}

	writeJSON(w, http.StatusOK, room)
}
func (a *App) joinRoom(
	w http.ResponseWriter,
	r *http.Request,
	ps httprouter.Params,
) {
	// Verify JWT token
	claims, err := a.verifyClaims(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "unauthorized",
		})
		return
	}

	roomID, err := parseID(ps.ByName("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid room id",
		})
		return
	}

	a.roomMux.Lock()
	defer a.roomMux.Unlock()

	room, ok := a.rooms[roomID]
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "room not found",
		})
		return
	}

	room.Members[claims.UserID] = true

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "joined room",
	})
}
func (a *App) leaveRoom(
	w http.ResponseWriter,
	r *http.Request,
	ps httprouter.Params,
) {
	// Verify JWT token
	claims, err := a.verifyClaims(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "unauthorized",
		})
		return
	}

	roomID, err := parseID(ps.ByName("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid room id",
		})
		return
	}

	a.roomMux.Lock()
	defer a.roomMux.Unlock()

	room, ok := a.rooms[roomID]
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "room not found",
		})
		return
	}

	delete(room.Members, claims.UserID)

	writeJSON(w, http.StatusOK, map[string]string{
		"message": "left room",
	})
}
func (a *App) sendMessage(
	w http.ResponseWriter,
	r *http.Request,
	ps httprouter.Params,
) {
	// Verify JWT token
	claims, err := a.verifyClaims(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "unauthorized",
		})
		return
	}

	roomID, err := parseID(ps.ByName("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid room id",
		})
		return
	}

	var input struct {
		Content string `json:"content"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid json",
		})
		return
	}

	input.Content = strings.TrimSpace(input.Content)

	if input.Content == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "message required",
		})
		return
	}

	a.roomMux.Lock()
	defer a.roomMux.Unlock()

	room, ok := a.rooms[roomID]
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "room not found",
		})
		return
	}

	if !room.Members[claims.UserID] {
		writeJSON(w, http.StatusForbidden, map[string]string{
			"error": "user not in room",
		})
		return
	}

	msg := Message{
		ID:        a.nextMsgID,
		RoomID:    roomID,
		UserID:    claims.UserID,
		Content:   input.Content,
		CreatedAt: time.Now(),
	}

	a.nextMsgID++

	room.Messages = append(room.Messages, msg)

	writeJSON(w, http.StatusCreated, msg)
}
func (a *App) getMessages(
	w http.ResponseWriter,
	r *http.Request,
	ps httprouter.Params,
) {
	roomID, err := parseID(ps.ByName("id"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid room id",
		})
		return
	}

	a.roomMux.RLock()
	defer a.roomMux.RUnlock()

	room, ok := a.rooms[roomID]
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "room not found",
		})
		return
	}

	writeJSON(w, http.StatusOK, room.Messages)
}
