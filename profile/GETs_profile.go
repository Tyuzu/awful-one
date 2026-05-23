package profile

import (
	"encoding/json"
	"net/http"
	"slices"
	"time"

	"naevis/infra"
	"naevis/models"
	"naevis/utils"

	"github.com/julienschmidt/httprouter"
)

/* -------------------------------------------------------
   Get Own Profile
------------------------------------------------------- */

func GetProfile(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctx := r.Context()
		requestingUserID := utils.GetUserIDFromRequest(r)

		user, err := findUser(ctx, map[string]any{"userid": requestingUserID}, app.DB)
		if err != nil || user == nil {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		userFollow, err := GetUserFollowData(ctx, user.UserID, app.DB)
		if err == nil && userFollow.UserID != "" {
			user.FollowersCount = len(userFollow.Followers)
			user.FollowingCount = len(userFollow.Follows)
		}

		user.Online, _ = isOnline(ctx, user.UserID, app.Cache)

		profileJSON, err := json.Marshal(user)
		if err != nil {
			http.Error(w, "Encoding failed", http.StatusInternalServerError)
			return
		}

		// Best-effort cache write (5 min TTL)
		_ = CacheProfile(ctx, app.Cache, user.Username, string(profileJSON), 5*time.Minute)

		w.Header().Set("Content-Type", "application/json")
		w.Write(profileJSON)
	}
}

/* -------------------------------------------------------
   Get Another User's Profile
------------------------------------------------------- */

func GetUserProfile(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx := r.Context()

		claims, err := validateJWT(r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		username := ps.ByName("username")

		user, err := findUser(ctx, map[string]any{"username": username}, app.DB)
		if err != nil || user == nil {
			http.Error(w, "User not found", http.StatusNotFound)
			return
		}

		userFollow, _ := GetUserFollowData(ctx, user.UserID, app.DB)

		isFollowing := false
		if userFollow.UserID != "" {
			isFollowing = slices.Contains(userFollow.Followers, claims.UserID)
		}

		online, _ := isOnline(ctx, user.UserID, app.Cache)

		response := models.UserProfileResponse{
			UserID:         user.UserID,
			Username:       user.Username,
			Email:          user.Email,
			Name:           user.Name,
			Bio:            user.Bio,
			Avatar:         user.Avatar,
			Banner:         user.Banner,
			FollowersCount: len(userFollow.Followers),
			FollowingCount: len(userFollow.Follows),
			IsFollowing:    isFollowing,
			Online:         online,
			LastLogin:      user.LastLogin,
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}
}
