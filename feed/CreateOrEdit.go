package feed

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"naevis/config/mqevent"
	"naevis/infra"
	"naevis/models"
	"naevis/userdata"
	"naevis/utils"
	"regexp"
	"strings"
	"time"
)

// PostAction defines if this is create or edit
type PostAction string

const (
	ActionCreate PostAction = "create"
	ActionEdit   PostAction = "edit"
)

// PostPayload now matches new frontend structure
type PostPayload struct {
	PostID      string     `json:"postid,omitempty"`
	Type        string     `json:"type,omitempty"`
	Text        string     `json:"text,omitempty"`
	Title       string     `json:"title,omitempty"`
	Description string     `json:"description,omitempty"`
	Tags        []string   `json:"tags,omitempty"`
	Caption     string     `json:"caption,omitempty"`
	Images      []MediaRef `json:"images,omitempty"`
	Video       *MediaRef  `json:"video,omitempty"`
	Thumbnail   *MediaRef  `json:"thumbnail,omitempty"`
}

// MediaRef supports nested resolutions inside video
type MediaRef struct {
	Filename    string `json:"filename"`
	Extn        string `json:"extn"`
	Key         string `json:"key"`
	Resolutions []int  `json:"resolutions,omitempty"`
}

// -------------------- POST HANDLERS --------------------

// editExistingPost updates an existing post using Database interface
func editExistingPost(ctx context.Context, claims *models.Claims, payload PostPayload, app *infra.Deps) (models.FeedPost, error) {
	var post models.FeedPost
	if payload.PostID == "" {
		return post, errors.New("missing postid")
	}

	update := map[string]any{}
	if payload.Text != "" {
		update["text"] = payload.Text
	}
	if payload.Title != "" {
		update["title"] = payload.Title
	}
	if payload.Description != "" {
		update["description"] = payload.Description
	}
	if len(payload.Tags) > 0 {
		update["tags"] = payload.Tags
	}
	if len(update) == 0 {
		return post, errors.New("nothing to update")
	}

	// Database interface call
	if err := app.DB.FindOneAndUpdate(ctx, feedpostsCollection,
		map[string]any{"postid": payload.PostID, "userid": claims.UserID},
		map[string]any{"$set": update},
		&post,
	); err != nil {
		return post, err
	}

	/* -------- Publish PostUpdated Event -------- */
	updatePayload := mqevent.PostUpdatedPayload{
		PostID:     post.PostID,
		UserID:     claims.UserID,
		OccurredAt: time.Now(),
	}

	updateBytes, err := json.Marshal(updatePayload)
	if err == nil {
		publishCtx, cancel := context.WithTimeout(
			context.Background(),
			3*time.Second,
		)
		defer cancel()

		_ = app.MQ.Publish(
			publishCtx,
			mqevent.PostUpdated,
			updateBytes,
		)
	}

	return post, nil
}

// insertNewPost inserts a new post using Database interface
func insertNewPost(ctx context.Context, claims *models.Claims, payload PostPayload, app *infra.Deps) (models.FeedPost, error) {
	post := models.FeedPost{
		PostID:      utils.GenerateRandomString(12),
		Username:    claims.Username,
		UserID:      claims.UserID,
		Text:        payload.Text,
		Title:       payload.Title,
		Description: payload.Description,
		Tags:        payload.Tags,
		Timestamp:   time.Now().Format(time.RFC3339),
		Likes:       0,
		Type:        payload.Type,
		Subtitles:   make(map[string]string),
	}

	switch payload.Type {
	case "image":
		if len(payload.Images) == 0 {
			return post, errors.New("no images attached")
		}
		if len(payload.Images) > 6 {
			return post, errors.New("cannot attach more than 6 images")
		}
		for _, img := range payload.Images {
			post.MediaURL = append(post.MediaURL, img.Filename)
			post.Media = append(post.Media, img.Filename+img.Extn)
		}
		post.Caption = payload.Caption

	case "video":
		if payload.Video == nil {
			return post, errors.New("missing video file")
		}
		post.MediaURL = []string{payload.Video.Filename}
		post.Media = []string{payload.Video.Filename + payload.Video.Extn}
		if payload.Thumbnail != nil {
			post.Thumbnail = payload.Thumbnail.Filename + payload.Thumbnail.Extn
		}
		if len(payload.Video.Resolutions) > 0 {
			post.Resolutions = payload.Video.Resolutions
		}

	case "text":
		// text-only post, nothing extra

	default:
		return post, errors.New("unsupported post type")
	}

	// Database interface insert
	if err := app.DB.InsertOne(ctx, feedpostsCollection, post); err != nil {
		return post, err
	}

	userdata.SetUserData("feedpost", post.PostID, claims.UserID, "", "", app)

	/* -------- Publish PostCreated Event -------- */
	createPayload := mqevent.PostCreatedPayload{
		PostID:     post.PostID,
		UserID:     claims.UserID,
		Username:   claims.Username,
		PostType:   post.Type,
		OccurredAt: time.Now(),
	}

	createBytes, err := json.Marshal(createPayload)
	if err == nil {
		publishCtx, cancel := context.WithTimeout(
			context.Background(),
			3*time.Second,
		)
		defer cancel()

		_ = app.MQ.Publish(
			publishCtx,
			mqevent.PostCreated,
			createBytes,
		)
	}

	return post, nil
}

// CreateOrEditPost creates or edits a post depending on action
func CreateOrEditPost(ctx context.Context, claims *models.Claims, payload PostPayload, action PostAction, app *infra.Deps) (models.FeedPost, error) {
	var post models.FeedPost

	payload, err := preparePostPayload(payload)
	if err != nil {
		return post, err
	}

	switch action {
	case ActionCreate:
		return insertNewPost(ctx, claims, payload, app)
	case ActionEdit:
		return editExistingPost(ctx, claims, payload, app)
	default:
		return post, errors.New("unsupported action")
	}
}

// preparePostPayload sanitizes and validates incoming post data
func preparePostPayload(payload PostPayload) (PostPayload, error) {
	if payload.Type == "" {
		payload.Type = "text"
	}
	payload.Type = utils.SanitizeText(payload.Type)
	payload.Text = utils.SanitizeText(payload.Text)

	if len([]rune(payload.Text)) > 500 {
		return payload, errors.New("post text exceeds 500 characters")
	}

	validPostTypes := map[string]bool{"text": true, "image": true, "video": true}
	if !validPostTypes[payload.Type] {
		return payload, errors.New("invalid post type")
	}

	if err := checkTextContent(payload.Text); err != nil {
		return payload, err
	}

	payload.Tags = sanitizeTags(payload.Tags)
	return payload, nil
}

// sanitizeTags removes duplicates and cleans tags
func sanitizeTags(tags []string) []string {
	seen := make(map[string]bool)
	clean := make([]string, 0, len(tags))
	for _, tag := range tags {
		t := utils.SanitizeText(tag)
		if t != "" && !seen[t] {
			seen[t] = true
			clean = append(clean, t)
		}
	}
	return clean
}

// checkTextContent scans for mentions, hashtags, URLs, and banned words
func checkTextContent(text string) error {
	mentions := extractMentions(text)
	hashtags := extractHashtags(text)
	urls := extractURLs(text)

	banned := []string{"spamword1", "offensiveword", "bannedtopic"}
	lowered := strings.ToLower(text)
	for _, bad := range banned {
		if strings.Contains(lowered, bad) {
			return errors.New("post contains prohibited content")
		}
	}

	if len(mentions) > 0 {
		log.Printf("mentions found: %v", mentions)
	}
	if len(hashtags) > 0 {
		log.Printf("hashtags found: %v", hashtags)
	}
	if len(urls) > 0 {
		log.Printf("urls found: %v", urls)
	}
	return nil
}

// -------------------- REGEX HELPERS --------------------

var (
	mentionRegex = regexp.MustCompile(`@([a-zA-Z0-9_]{1,15})`)
	hashtagRegex = regexp.MustCompile(`#(\w+)`)
	urlRegex     = regexp.MustCompile(`https?://[^\s]+`)
)

func extractMentions(text string) []string {
	matches := mentionRegex.FindAllStringSubmatch(text, -1)
	out := []string{}
	for _, m := range matches {
		if len(m) > 1 {
			out = append(out, m[1])
		}
	}
	return out
}

func extractHashtags(text string) []string {
	matches := hashtagRegex.FindAllStringSubmatch(text, -1)
	out := []string{}
	for _, m := range matches {
		if len(m) > 1 {
			out = append(out, m[1])
		}
	}
	return out
}

func extractURLs(text string) []string {
	return urlRegex.FindAllString(text, -1)
}
