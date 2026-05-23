package profile

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"naevis/infra/cache"
	"naevis/infra/db"
	"naevis/models"
	"naevis/utils"
)

/* -------------------------------------------------------
   Helpers
------------------------------------------------------- */

// validateJWT extracts + validates JWT from header
func validateJWT(r *http.Request) (*models.Claims, error) {
	token := r.Header.Get("Authorization")
	if token == "" {
		return nil, errors.New("no auth header")
	}
	return utils.ValidateJWT(token)
}

/* -------------------------------------------------------
   Helpers
------------------------------------------------------- */

// findUser returns a user by filter, or nil if not found
func findUser(ctx context.Context, filter map[string]any, database db.Database) (*models.User, error) {
	var user models.User
	_ = database.FindOne(ctx, usersCollection, filter, &user)
	// ignore errors; return nil if user not found
	if user.UserID == "" {
		return nil, nil
	}
	return &user, nil
}

// isOnline checks if a user is online via cache
func isOnline(ctx context.Context, userid string, cache cache.Cache) (bool, error) {
	return cache.Exists(ctx, "online:"+userid)
}

/* -------------------------------------------------------
   User profile endpoints
------------------------------------------------------- */

// RespondWithUserProfile writes user profile as JSON to the response
func RespondWithUserProfile(w http.ResponseWriter, userid string, database db.Database) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var userProfile models.User
	_ = database.FindOne(ctx, usersCollection, map[string]any{"userid": userid}, &userProfile)

	if userProfile.UserID == "" {
		http.Error(w, "User not found", http.StatusNotFound)
		return nil
	}

	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(userProfile)
}

/* -------------------------------------------------------
   Follow data + caching utilities
------------------------------------------------------- */

// GetUserFollowData returns followers and follows for a user
func GetUserFollowData(ctx context.Context, userID string, database db.Database) (models.UserFollow, error) {
	var uf models.UserFollow
	_ = database.FindOne(ctx, "followings", map[string]any{"userid": userID}, &uf)

	// return empty if not found
	if uf.UserID == "" {
		return models.UserFollow{
			Followers: []string{},
			Follows:   []string{},
		}, nil
	}

	return uf, nil
}

// CacheProfile stores the serialized profile in cache
func CacheProfile(ctx context.Context, c cache.Cache, username string, data string, ttl time.Duration) error {
	return c.Set(ctx, "profile:"+username, []byte(data), ttl)
}

// GetCachedProfile fetches the cached profile
func GetCachedProfile(ctx context.Context, c cache.Cache, username string) (string, error) {
	data, err := c.Get(ctx, "profile:"+username)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// InvalidateCachedProfile deletes the cached profile
func InvalidateCachedProfile(ctx context.Context, c cache.Cache, username string) error {
	return c.Del(ctx, "profile:"+username)
}

// UpdateCachedUsername invalidates cached data keyed by user ID
func UpdateCachedUsername(ctx context.Context, c cache.Cache, userid string) error {
	return c.Del(ctx, fmt.Sprintf("users:%s", userid))
}
