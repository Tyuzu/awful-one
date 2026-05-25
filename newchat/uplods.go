package newchat

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"naevis/infra"
	"naevis/utils"

	"github.com/julienschmidt/httprouter"
)

// ------------------------- DB helpers -------------------------

func UpdatexMessage(userID string, id string, newContent string, app *infra.Deps) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := map[string]any{
		"messageid": id,
		"senderid":  userID,
	}
	update := map[string]any{
		"$set": map[string]any{"content": newContent},
	}

	err := app.DB.UpdateOne(ctx, messagesCollection, filter, update)
	if err != nil {
		if strings.Contains(err.Error(), "no documents") {
			return errors.New("message not found or unauthorized")
		}
		return err
	}
	return nil
}

func DeletexMessage(userID string, id string, app *infra.Deps) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := map[string]any{
		"messageid": id,
		"senderid":  userID,
	}

	_, err := app.DB.DeleteOne(ctx, messagesCollection, filter)
	if err != nil {
		if strings.Contains(err.Error(), "no documents") {
			return errors.New("message not found or unauthorized")
		}
		return err
	}
	return nil
}

func findMessageRoom(id string, app *infra.Deps) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var msg Message
	err := app.DB.FindOne(ctx, messagesCollection, map[string]any{"messageid": id}, &msg)
	if err != nil {
		return "", err
	}
	return msg.Room, nil
}

// ------------------------- HTTP handlers -------------------------

type editPayload struct {
	ID      string `json:"id"`
	Content string `json:"content"`
}

func EditMessageHandler(hub *Hub, app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		var payload editPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		userID := utils.GetUserIDFromRequest(r)
		if userID == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		if err := UpdatexMessage(userID, payload.ID, payload.Content, app); err != nil {
			if strings.Contains(err.Error(), "not found") {
				http.Error(w, "not found", http.StatusNotFound)
			} else {
				http.Error(w, "forbidden", http.StatusForbidden)
			}
			return
		}

		room, err := findMessageRoom(payload.ID, app)
		if err != nil {
			http.Error(w, "room not found", http.StatusNotFound)
			return
		}

		out := outboundPayload{
			Action:    "edit",
			ID:        payload.ID,
			Content:   payload.Content,
			Timestamp: time.Now().Unix(),
		}
		if data, err := json.Marshal(out); err == nil {
			hub.broadcast <- broadcastMsg{Room: room, Data: data}
		}

		w.WriteHeader(http.StatusOK)
	}
}

func DeleteMessageHandler(hub *Hub, app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		var payload struct {
			ID string `json:"id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		userID := utils.GetUserIDFromRequest(r)
		if userID == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		room, err := findMessageRoom(payload.ID, app)
		if err != nil {
			http.Error(w, "room not found", http.StatusNotFound)
			return
		}

		if err := DeletexMessage(userID, payload.ID, app); err != nil {
			if strings.Contains(err.Error(), "not found") {
				http.Error(w, "not found", http.StatusNotFound)
			} else {
				http.Error(w, "forbidden", http.StatusForbidden)
			}
			return
		}

		out := outboundPayload{
			Action: "delete",
			ID:     payload.ID,
		}
		if data, err := json.Marshal(out); err == nil {
			hub.broadcast <- broadcastMsg{Room: room, Data: data}
		}

		w.WriteHeader(http.StatusOK)
	}
}

func UploadHandler(hub *Hub, app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		userID := utils.GetUserIDFromRequest(r)
		if userID == "" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		var payload struct {
			Chat  string       `json:"chat"`
			Files []Attachment `json:"files"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if payload.Chat == "" {
			http.Error(w, "chat missing", http.StatusBadRequest)
			return
		}

		ts := time.Now().Unix()
		msg := Message{
			MessageID: utils.GenerateRandomDigitString(16),
			Room:      payload.Chat,
			SenderID:  userID,
			Content:   "",
			Files:     payload.Files,
			Timestamp: ts,
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := app.DB.InsertOne(ctx, messagesCollection, msg); err != nil {
			log.Println("db error:", err)
			http.Error(w, "db error", http.StatusInternalServerError)
			return
		}

		out := outboundPayload{
			Action:    "chat",
			ID:        msg.MessageID,
			Room:      msg.Room,
			SenderID:  msg.SenderID,
			Content:   msg.Content,
			Files:     msg.Files,
			Timestamp: msg.Timestamp,
		}

		data, _ := json.Marshal(out)
		hub.broadcast <- broadcastMsg{Room: msg.Room, Data: data}

		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	}
}
