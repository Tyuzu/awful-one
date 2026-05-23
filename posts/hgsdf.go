package posts

import (
	"encoding/json"
	"html"
	"naevis/globals"
	"naevis/infra"
	"naevis/models"
	"naevis/utils"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
)

// --- Allowed block types ---
var allowedBlockTypes = map[string]bool{
	"text":  true,
	"image": true,
	"code":  true,
	"video": true,
}

// --- Pick thumbnail ---
func pickThumb(blocks []models.Block) string {
	for _, b := range blocks {
		if b.Type == "image" && b.URL != "" {
			return b.URL
		}
	}
	for _, b := range blocks {
		if b.Type == "video" && b.URL != "" {
			return b.URL
		}
	}
	return ""
}

// --- Sanitize blocks ---
func sanitizeBlocks(raw []models.Block) []models.Block {
	out := make([]models.Block, 0, len(raw))
	for _, b := range raw {
		if !allowedBlockTypes[b.Type] {
			continue
		}

		switch b.Type {
		case "text":
			b.Content = strings.TrimSpace(html.EscapeString(b.Content))
			if b.Content != "" {
				out = append(out, b)
			}
		case "image":
			b.URL = strings.TrimSpace(b.URL)
			b.Alt = strings.TrimSpace(html.EscapeString(b.Alt))
			if b.URL != "" {
				out = append(out, b)
			}
		case "code":
			b.Language = strings.TrimSpace(html.EscapeString(b.Language))
			b.Content = strings.TrimSpace(html.EscapeString(b.Content))
			if b.Content != "" {
				out = append(out, b)
			}
		case "video":
			b.URL = strings.TrimSpace(b.URL)
			b.Caption = strings.TrimSpace(html.EscapeString(b.Caption))
			if b.URL != "" {
				out = append(out, b)
			}
		}
	}
	return out
}

// --- Create / Update ---
func CreateOrUpdatePost(w http.ResponseWriter, r *http.Request, ps httprouter.Params, isEdit bool, app *infra.Deps) {
	ctx := r.Context()
	userID, ok := ctx.Value(globals.UserIDKey).(string)
	if !ok || userID == "" {
		utils.RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}

	var postID string
	if isEdit {
		postID = ps.ByName("id")
		if postID == "" {
			utils.RespondWithError(w, http.StatusBadRequest, "Post ID required")
			return
		}
	}

	if err := r.ParseMultipartForm(20 << 20); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid form data")
		return
	}

	title := strings.TrimSpace(html.EscapeString(r.FormValue("title")))
	category := strings.TrimSpace(r.FormValue("category"))
	subcategory := strings.TrimSpace(r.FormValue("subcategory"))
	referenceID := strings.TrimSpace(r.FormValue("referenceId"))
	blocksRaw := r.FormValue("blocks")

	rawTags := r.MultipartForm.Value["hashtags"]
	hashtags := make([]string, 0, len(rawTags))
	for _, t := range rawTags {
		tt := strings.TrimSpace(html.EscapeString(t))
		if tt != "" {
			hashtags = append(hashtags, tt)
		}
	}

	if title == "" || category == "" || subcategory == "" || blocksRaw == "" {
		utils.RespondWithError(w, http.StatusBadRequest, "Missing required fields")
		return
	}

	var blocksIn []models.Block
	if err := json.Unmarshal([]byte(blocksRaw), &blocksIn); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid blocks JSON")
		return
	}

	blocks := sanitizeBlocks(blocksIn)

	var refPtr *string
	if category == "Review" &&
		(subcategory == "Product" || subcategory == "Place" || subcategory == "Event") {

		if referenceID == "" {
			utils.RespondWithError(w, http.StatusBadRequest, "Reference ID required")
			return
		}
		refPtr = &referenceID
	}

	now := time.Now()

	if isEdit {
		filter := map[string]any{
			"postid":    postID,
			"createdBy": userID,
		}

		update := map[string]any{
			"title":       title,
			"category":    category,
			"subcategory": subcategory,
			"referenceId": refPtr,
			"blocks":      blocks,
			"hashtags":    hashtags,
			"thumb":       pickThumb(blocks),
			"updatedAt":   now,
		}

		if err := app.DB.UpdateOne(ctx, blogPostsCollection, filter, update); err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to update post")
			return
		}

		utils.RespondWithJSON(w, http.StatusOK, map[string]any{"postid": postID})
		return
	}

	newPost := models.BlogPost{
		PostID:      uuid.NewString(),
		Title:       title,
		Category:    category,
		Subcategory: subcategory,
		ReferenceID: refPtr,
		Blocks:      blocks,
		Hashtags:    hashtags,
		Thumb:       pickThumb(blocks),
		CreatedBy:   userID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := app.DB.InsertOne(ctx, blogPostsCollection, newPost); err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to create post")
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, map[string]any{"postid": newPost.PostID})
}

// --- Wrappers ---
func CreatePost(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		CreateOrUpdatePost(w, r, ps, false, app)
	}
}

func UpdatePost(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		CreateOrUpdatePost(w, r, ps, true, app)
	}
}

// --- Delete ---
func DeletePost(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx := r.Context()
		userID := utils.GetUserIDFromRequest(r)
		postID := ps.ByName("id")

		if postID == "" {
			utils.RespondWithError(w, http.StatusBadRequest, "Post ID required")
			return
		}

		filter := map[string]any{
			"postid":    postID,
			"createdBy": userID,
		}

		_, err := app.DB.DeleteOne(ctx, blogPostsCollection, filter)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to delete post")
			return
		}
		// if !deleted {
		// 	utils.RespondWithError(w, http.StatusNotFound, "Post not found or unauthorized")
		// 	return
		// }

		utils.RespondWithJSON(w, http.StatusOK, map[string]any{
			"postid":  postID,
			"deleted": true,
		})
	}
}
