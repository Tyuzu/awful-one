package settings

import (
	"encoding/json"
	"net/http"

	"naevis/globals"
	"naevis/infra"

	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/bson"
)

/* -------------------------
   Models
------------------------- */

type UserSettings struct {
	UserID        string `json:"userID,omitempty" bson:"userID"`
	Theme         string `json:"theme" bson:"theme"`
	Notifications bool   `json:"notifications" bson:"notifications"`
	PrivacyMode   bool   `json:"privacy_mode" bson:"privacy_mode"`
	AutoLogout    bool   `json:"auto_logout" bson:"auto_logout"`
	Language      string `json:"language" bson:"language"`
	TimeZone      string `json:"time_zone" bson:"time_zone"`
	DailyReminder string `json:"daily_reminder" bson:"daily_reminder"`
}

/* -------------------------
   Defaults
------------------------- */

func getDefaultSettings(userID string) UserSettings {
	return UserSettings{
		UserID:        userID,
		Theme:         "light",
		Notifications: true,
		PrivacyMode:   false,
		AutoLogout:    false,
		Language:      "english",
		TimeZone:      "UTC",
		DailyReminder: "09:00",
	}
}

/* -------------------------
   Helpers
------------------------- */

func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

/* -------------------------
   Get User Settings
------------------------- */

func GetUserSettings(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctx := r.Context()
		userID := ctx.Value(globals.UserIDKey).(string)

		var settings UserSettings
		err := app.DB.FindOne(
			ctx,
			settingsCollection,
			bson.M{"userID": userID},
			&settings,
		)

		if err != nil {
			settings = getDefaultSettings(userID)
			_ = app.DB.Insert(ctx, settingsCollection, settings)
		}

		settingsArray := []map[string]any{
			{"type": "theme", "value": settings.Theme, "description": "Choose theme mode"},
			{"type": "notifications", "value": settings.Notifications, "description": "Enable notifications"},
			{"type": "privacy_mode", "value": settings.PrivacyMode, "description": "Enable privacy mode"},
			{"type": "auto_logout", "value": settings.AutoLogout, "description": "Enable auto logout"},
			{"type": "language", "value": settings.Language, "description": "Select language"},
			{"type": "time_zone", "value": settings.TimeZone, "description": "Select time zone"},
			{"type": "daily_reminder", "value": settings.DailyReminder, "description": "Set daily reminder"},
		}

		respondJSON(w, http.StatusOK, settingsArray)
	}
}

/* -------------------------
   Update User Setting
------------------------- */

type UpdateSettingPayload struct {
	Value any `json:"value"`
}

func UpdateUserSetting(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		ctx := r.Context()
		userID := ctx.Value(globals.UserIDKey).(string)
		settingType := ps.ByName("type")

		valid := map[string]bool{
			"theme":          true,
			"notifications":  true,
			"privacy_mode":   true,
			"auto_logout":    true,
			"language":       true,
			"time_zone":      true,
			"daily_reminder": true,
		}
		if !valid[settingType] {
			respondJSON(w, http.StatusBadRequest, map[string]string{
				"error": "Invalid setting type",
			})
			return
		}

		var payload UpdateSettingPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			respondJSON(w, http.StatusBadRequest, map[string]string{
				"error": "Invalid input",
			})
			return
		}

		filter := bson.M{"userID": userID}
		update := bson.M{
			settingType: payload.Value,
		}

		if err := app.DB.Update(ctx, settingsCollection, filter, update); err != nil {
			// create if missing
			settings := getDefaultSettings(userID)
			switch settingType {
			case "theme":
				settings.Theme, _ = payload.Value.(string)
			case "notifications":
				settings.Notifications, _ = payload.Value.(bool)
			case "privacy_mode":
				settings.PrivacyMode, _ = payload.Value.(bool)
			case "auto_logout":
				settings.AutoLogout, _ = payload.Value.(bool)
			case "language":
				settings.Language, _ = payload.Value.(string)
			case "time_zone":
				settings.TimeZone, _ = payload.Value.(string)
			case "daily_reminder":
				settings.DailyReminder, _ = payload.Value.(string)
			}
			_ = app.DB.Insert(ctx, settingsCollection, settings)
		}

		respondJSON(w, http.StatusOK, map[string]any{
			"status":  "success",
			"message": "Setting updated successfully",
			"type":    settingType,
			"value":   payload.Value,
		})
	}
}

/* -------------------------
   Init User Settings
------------------------- */

func InitUserSettings(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctx := r.Context()
		userID := ctx.Value(globals.UserIDKey).(string)

		var existing UserSettings
		err := app.DB.FindOne(
			ctx,
			settingsCollection,
			bson.M{"userID": userID},
			&existing,
		)

		if err == nil {
			respondJSON(w, http.StatusOK, false)
			return
		}

		settings := getDefaultSettings(userID)
		if err := app.DB.Insert(ctx, settingsCollection, settings); err != nil {
			respondJSON(w, http.StatusInternalServerError, map[string]string{
				"error": "Failed to initialize settings",
			})
			return
		}

		respondJSON(w, http.StatusOK, true)
	}
}
