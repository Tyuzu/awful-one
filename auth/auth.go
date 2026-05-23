package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"naevis/config/mqevent"
	"naevis/infra"
	"naevis/models"
	"naevis/utils"

	"github.com/golang-jwt/jwt/v5"
	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

/* ============================================================
   REGISTER
============================================================ */

func Register(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		var input struct {
			Username string `json:"username"`
			Password string `json:"password"`
			Email    string `json:"email"`
		}

		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, "Invalid input")
			return
		}

		input.Username = strings.TrimSpace(input.Username)
		input.Password = strings.TrimSpace(input.Password)
		input.Email = strings.ToLower(strings.TrimSpace(input.Email))

		if !validateUsername(input.Username) ||
			!validateEmail(input.Email) ||
			!validatePassword(input.Password) {

			utils.RespondWithError(w, http.StatusBadRequest, "Invalid credentials")
			return
		}

		hashedPassword, err := bcrypt.GenerateFromPassword(
			[]byte(input.Password),
			bcrypt.DefaultCost,
		)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Password error")
			return
		}

		now := time.Now()

		user := models.User{
			UserID:        "u" + utils.GenerateRandomString(10),
			Username:      input.Username,
			Email:         input.Email,
			Password:      string(hashedPassword),
			Role:          []string{"user"},
			CreatedAt:     now,
			UpdatedAt:     now,
			EmailVerified: false,
			IsVerified:    false,
			Online:        false,
		}

		if err := app.DB.Insert(ctx, UsersCollection, user); err != nil {
			if mongo.IsDuplicateKeyError(err) {
				utils.RespondWithError(w, http.StatusConflict, "User already exists")
				return
			}

			utils.RespondWithError(w, http.StatusInternalServerError, "Registration failed")
			return
		}

		/* ---------------- Event Payload ---------------- */

		payload := mqevent.UserRegisteredPayload{
			UserID:    user.UserID,
			Username:  user.Username,
			Email:     user.Email,
			CreatedAt: user.CreatedAt,
		}

		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Event serialization failed")
			return
		}

		/* ---------------- Publish Event ---------------- */

		publishCtx, publishCancel := context.WithTimeout(
			context.Background(),
			3*time.Second,
		)
		defer publishCancel()

		if err := app.MQ.Publish(
			publishCtx,
			mqevent.UserRegistered,
			payloadBytes,
		); err != nil {

			utils.RespondWithError(w, http.StatusInternalServerError, "Event publish failed")
			return
		}

		utils.RespondWithJSON(w, http.StatusCreated, map[string]any{
			"message": "User registered successfully",
			"userid":  user.UserID,
		})
	}
}

/* ============================================================
   LOGIN
============================================================ */

func Login(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		var creds struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}

		if err := json.NewDecoder(r.Body).Decode(&creds); err != nil {
			utils.RespondWithError(w, http.StatusBadRequest, "Invalid input")
			return
		}

		creds.Username = strings.TrimSpace(creds.Username)
		creds.Password = strings.TrimSpace(creds.Password)

		ip := clientIP(r)
		failKey := fmt.Sprintf("auth:fail:%s:%s", creds.Username, ipPrefix(ip))

		val, err := app.Cache.Get(ctx, failKey)

		var cnt int64

		if err == nil && len(val) > 0 {
			cnt, _ = strconv.ParseInt(string(val), 10, 64)
		}

		if cnt >= maxFailedAttempts {
			utils.RespondWithError(w, http.StatusTooManyRequests, "Too many attempts")
			return
		}

		var user models.User

		if err := app.DB.FindOne(
			ctx,
			UsersCollection,
			bson.M{"username": creds.Username},
			&user,
		); err != nil {

			cnt, _ = app.Cache.Incr(ctx, failKey)

			app.Cache.Set(
				ctx,
				failKey,
				[]byte(strconv.FormatInt(cnt, 10)),
				lockoutDuration,
			)

			utils.RespondWithError(w, http.StatusUnauthorized, "Invalid credentials")
			return
		}

		if bcrypt.CompareHashAndPassword(
			[]byte(user.Password),
			[]byte(creds.Password),
		) != nil {

			cnt, _ = app.Cache.Incr(ctx, failKey)

			app.Cache.Set(
				ctx,
				failKey,
				[]byte(strconv.FormatInt(cnt, 10)),
				lockoutDuration,
			)

			utils.RespondWithError(w, http.StatusUnauthorized, "Invalid credentials")
			return
		}

		/* ---------------- Clear Fail Counter ---------------- */

		app.Cache.Del(ctx, failKey)

		/* ---------------- JWT Claims ---------------- */

		claims := &models.Claims{
			UserID:   user.UserID,
			Username: user.Username,
			Role:     user.Role,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(
					time.Now().Add(AccessTokenTTL),
				),
				IssuedAt: jwt.NewNumericDate(time.Now()),
			},
		}

		accessToken, err := createAccessToken(claims)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Token error")
			return
		}

		refreshToken, err := generateRefreshToken()
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Token error")
			return
		}

		/* ---------------- Persist Session ---------------- */

		err = app.DB.Update(
			ctx,
			UsersCollection,
			bson.M{"userid": user.UserID},
			bson.M{
				"$set": bson.M{
					"refresh_token":  hashRefreshToken(refreshToken),
					"refresh_expiry": time.Now().Add(RefreshTokenTTL),
					"refresh_ua":     uaHash(r),
					"refresh_ip":     ipPrefix(ip),
					"last_login":     time.Now(),
					"online":         true,
					"updated_at":     time.Now(),
				},
			},
		)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Session error")
			return
		}

		/* ---------------- Set Refresh Cookie ---------------- */

		setRefreshCookie(w, refreshToken)

		/* ---------------- Publish Login Event ---------------- */

		loginPayload := mqevent.UserLoggedInPayload{
			UserID:     user.UserID,
			Username:   user.Username,
			OccurredAt: time.Now(),
			IP:         ipPrefix(ip),
		}

		loginBytes, err := json.Marshal(loginPayload)
		if err == nil {
			publishCtx, cancel := context.WithTimeout(
				context.Background(),
				3*time.Second,
			)

			defer cancel()

			_ = app.MQ.Publish(
				publishCtx,
				mqevent.UserLoggedIn,
				loginBytes,
			)
		}

		utils.RespondWithJSON(w, http.StatusOK, map[string]any{
			"message": "Login successful",
			"data": map[string]string{
				"token":  accessToken,
				"userid": user.UserID,
			},
		})
	}
}

/* ============================================================
   LOGOUT
============================================================ */

func LogoutUser(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		if r.Header.Get("X-Refresh-Intent") != "1" {
			utils.RespondWithError(w, http.StatusForbidden, "CSRF blocked")
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		cookie, err := r.Cookie("refresh_token")

		if err == nil && cookie.Value != "" {
			hashed := hashRefreshToken(cookie.Value)

			var user models.User
			_ = app.DB.FindOne(
				ctx,
				UsersCollection,
				bson.M{"refresh_token": hashed},
				&user,
			)

			_ = app.DB.Update(
				ctx,
				UsersCollection,
				bson.M{"refresh_token": hashed},
				bson.M{
					"$unset": bson.M{
						"refresh_token":  "",
						"refresh_expiry": "",
					},
					"$set": bson.M{
						"online":     false,
						"updated_at": time.Now(),
					},
				},
			)

			/* -------- Publish Logout Event -------- */
			if user.UserID != "" {
				logoutPayload := mqevent.UserLoggedOutPayload{
					UserID:     user.UserID,
					OccurredAt: time.Now(),
				}

				logoutBytes, err := json.Marshal(logoutPayload)
				if err == nil {
					publishCtx, cancel := context.WithTimeout(
						context.Background(),
						3*time.Second,
					)
					defer cancel()

					_ = app.MQ.Publish(
						publishCtx,
						mqevent.UserLoggedOut,
						logoutBytes,
					)
				}
			}
		}

		clearRefreshCookie(w)

		utils.RespondWithJSON(w, http.StatusOK, map[string]any{
			"message": "Logged out",
			"data":    nil,
		})
	}
}

/* ============================================================
   LOGOUT ALL
============================================================ */

func LogoutAllSessions(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		authHeader := r.Header.Get("Authorization")

		if authHeader == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		tokenString := authHeader

		claims, err := utils.ValidateJWT(tokenString)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		err = app.DB.Update(
			ctx,
			UsersCollection,
			bson.M{"userid": claims.UserID},
			bson.M{
				"$unset": bson.M{
					"refresh_token":  "",
					"refresh_prev":   "",
					"refresh_expiry": "",
					"refresh_ua":     "",
					"refresh_ip":     "",
				},
				"$set": bson.M{
					"online":     false,
					"updated_at": time.Now(),
				},
			},
		)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Logout failed")
			return
		}

		/* -------- Publish Logout Event -------- */
		logoutPayload := mqevent.UserLoggedOutPayload{
			UserID:     claims.UserID,
			OccurredAt: time.Now(),
		}

		logoutBytes, err := json.Marshal(logoutPayload)
		if err == nil {
			publishCtx, cancel := context.WithTimeout(
				context.Background(),
				3*time.Second,
			)
			defer cancel()

			_ = app.MQ.Publish(
				publishCtx,
				mqevent.UserLoggedOut,
				logoutBytes,
			)
		}

		clearRefreshCookie(w)

		utils.RespondWithJSON(w, http.StatusOK, map[string]any{
			"message": "All sessions revoked",
		})
	}
}
