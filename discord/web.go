package discord

import (
	"encoding/json"
	"naevis/infra"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

type CreateMessageRequest struct {
	Content string `json:"content"`
	Nonce   string `json:"nonce"`
}

func CreateMessageHTTP(
	app *infra.Deps,
) httprouter.Handle {

	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx := r.Context()

		roomID := ps.ByName("room")
		userID := ctx.Value("userID").(string)

		var req CreateMessageRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}

		if req.Content == "" || req.Nonce == "" {
			http.Error(w, "content and nonce required", http.StatusBadRequest)
			return
		}

		msg, err := CreateMessage(
			ctx,
			app.DB,
			app.Cache,
			app.MQ,
			roomID,
			userID,
			req.Content,
			req.Nonce,
		)
		if err != nil {
			http.Error(w, "failed to create message", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(msg)
	}
}
