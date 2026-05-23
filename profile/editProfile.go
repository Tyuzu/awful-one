package profile

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/julienschmidt/httprouter"
	"golang.org/x/crypto/bcrypt"

	"naevis/infra"
	"naevis/infra/cache"
	"naevis/infra/db"
	"naevis/models"
)

/* -------------------------------------------------------
   Edit Profile
------------------------------------------------------- */

func EditProfile(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctx := r.Context()

		// 1. Validate JWT
		claims, err := validateJWT(r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// 2. Parse form data (~10 MB)
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			http.Error(w, "Unable to parse form", http.StatusBadRequest)
			return
		}

		// 3. Invalidate cached profile
		_ = InvalidateCachedProfile(ctx, app.Cache, claims.Username)
		_ = UpdateCachedUsername(ctx, app.Cache, claims.UserID)

		// 4. Build updates map
		updates, err := BuildProfileUpdates(ctx, r, claims, app.Cache)
		if err != nil {
			http.Error(w, "Failed to update profile fields", http.StatusInternalServerError)
			return
		}

		// 5. Apply updates in DB
		if err := ApplyProfileUpdates(ctx, app.DB, claims.UserID, updates); err != nil {
			http.Error(w, "Failed to update profile", http.StatusInternalServerError)
			return
		}

		// 6. Respond with updated profile
		if err := RespondWithUserProfile(w, claims.UserID, app.DB); err != nil {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
	}
}

/* -------------------------------------------------------
   Delete Profile
------------------------------------------------------- */

func DeleteProfile(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctx := r.Context()

		claims, err := validateJWT(r)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Invalidate cached profile
		_ = InvalidateCachedProfile(ctx, app.Cache, claims.Username)
		_ = UpdateCachedUsername(ctx, app.Cache, claims.UserID)

		// Delete user in DB
		if _, err := DeleteUserByID(ctx, app.DB, claims.UserID); err != nil {
			http.Error(w, "Failed to delete profile", http.StatusInternalServerError)
			return
		}

		// Success response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"message": "Profile deleted successfully",
		})
	}
}

/* -------------------------------------------------------
   Helper: Build profile updates
------------------------------------------------------- */

func BuildProfileUpdates(
	ctx context.Context,
	r *http.Request,
	claims *models.Claims,
	c cache.Cache,
) (map[string]any, error) {

	updates := map[string]any{}

	// Username
	if newUsername := r.FormValue("username"); newUsername != "" && newUsername != claims.Username {
		updates["username"] = newUsername
		_ = c.HSet(ctx, "users", claims.UserID, []byte(newUsername))
	}

	// Email, Bio, Name, Phone
	if val := r.FormValue("email"); val != "" {
		updates["email"] = val
	}
	if val := r.FormValue("bio"); val != "" {
		updates["bio"] = val
	}
	if val := r.FormValue("name"); val != "" {
		updates["name"] = val
	}
	if val := r.FormValue("phone"); val != "" {
		updates["phone_number"] = val
	}

	// Password (hashed)
	if val := r.FormValue("password"); val != "" {
		hashed, err := bcrypt.GenerateFromPassword([]byte(val), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		updates["password"] = string(hashed)
	}

	return updates, nil
}

/* -------------------------------------------------------
   Apply updates / Delete user
------------------------------------------------------- */

func ApplyProfileUpdates(
	ctx context.Context,
	database db.Database,
	userID string,
	updates map[string]any,
) error {
	return database.UpdateOne(ctx, usersCollection, map[string]any{"userid": userID}, updates)
}

func DeleteUserByID(
	ctx context.Context,
	database db.Database,
	userID string,
) (int64, error) {
	return database.DeleteOne(ctx, usersCollection, map[string]any{"userid": userID})
}
