package utils

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"

	"naevis/globals"
	"naevis/models"
)

// ----------------------
// Parsing Helpers
// ----------------------

func ParseFloat(s string) float64 {
	val, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return val
}

func ParseInt(s string) int {
	val, _ := strconv.Atoi(strings.TrimSpace(s))
	return val
}

func ParseDate(s string) *time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return nil
	}
	return &t
}

func GetUUID() string {
	return uuid.New().String()
}

func SanitizeText(s string) string {
	return strings.TrimSpace(s)
}

// ----------------------
// Pagination / Query Helpers
// ----------------------

type QueryOptions struct {
	Page      int
	Limit     int
	Published *bool
	Search    string
	Genre     string
}

func ParseQueryOptions(r *http.Request) QueryOptions {
	q := r.URL.Query()

	page := 1
	if p, err := strconv.Atoi(q.Get("page")); err == nil && p > 0 {
		page = p
	}

	limit := 10
	if l, err := strconv.Atoi(q.Get("limit")); err == nil && l > 0 {
		limit = l
	}

	var published *bool
	if pubStr := q.Get("published"); pubStr != "" {
		val := pubStr == "true"
		published = &val
	}

	return QueryOptions{
		Page:      page,
		Limit:     limit,
		Published: published,
		Search:    q.Get("search"),
		Genre:     q.Get("genre"),
	}
}

func ContainsIgnoreCase(str, substr string) bool {
	return strings.Contains(strings.ToLower(str), strings.ToLower(substr))
}

// ----------------------
// HTTP JSON Helpers
// ----------------------

func RespondWithJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
	}
}

func RespondWithError(w http.ResponseWriter, code int, msg string) {
	RespondWithJSON(w, code, map[string]string{"error": msg})
}

type M map[string]interface{}

func ToJSON(v interface{}) []byte {
	data, _ := json.Marshal(v)
	return data
}

// ----------------------
// Sorting / Filtering Helpers
// ----------------------

// RegexFilter creates a case-insensitive regex filter compatible with db.Database
func RegexFilter(field, value string) map[string]any {
	if value == "" {
		return map[string]any{}
	}
	return map[string]any{
		field: map[string]any{
			"$regex":   regexp.QuoteMeta(value),
			"$options": "i",
		},
	}
}

// ParseSort returns a sort map usable in db.FindManyOptions
func ParseSort(
	param string,
	defaultSort bson.D,
	sortMap map[string]bson.D,
) bson.D {

	if s, ok := sortMap[param]; ok {
		return s
	}

	return defaultSort
}

// ParsePagination extracts skip and limit values safely
func ParsePagination(r *http.Request, defaultLimit, maxLimit int) (skip, limit int) {
	page := 1
	limit = defaultLimit

	if v := r.URL.Query().Get("page"); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p > 0 {
			page = p
		}
	}

	if v := r.URL.Query().Get("limit"); v != "" {
		if l, err := strconv.Atoi(v); err == nil && l > 0 && l <= maxLimit {
			limit = l
		}
	}

	skip = (page - 1) * limit
	return
}

// ----------------------
// JWT Helpers
// ----------------------

func ExtractBearerToken(header string) string {
	if len(header) > 7 && strings.HasPrefix(header, "Bearer ") {
		return header[7:]
	}
	return ""
}

func ParseToken(tokenString string) (*models.Claims, error) {
	claims := &models.Claims{}
	_, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		return globals.JwtSecret, nil
	})
	if err != nil || claims.UserID == "" {
		return nil, fmt.Errorf("unauthorized: %w", err)
	}
	return claims, nil
}

func ValidateJWT(tokenString string) (*models.Claims, error) {
	tokenString = ExtractBearerToken(tokenString)
	if tokenString == "" {
		return nil, fmt.Errorf("invalid token")
	}
	return ParseToken(tokenString)
}

// ----------------------
// User Context Helpers
// ----------------------

func GetUserIDFromRequest(r *http.Request) string {
	ctx := r.Context()
	userID, ok := ctx.Value(globals.UserIDKey).(string)
	if !ok || userID == "" {
		return ""
	}
	return userID
}

func GetUsernameFromRequest(r *http.Request) string {
	tokenString := r.Header.Get("Authorization")
	claims, err := ValidateJWT(tokenString)
	if err != nil {
		return ""
	}
	return claims.Username
}
