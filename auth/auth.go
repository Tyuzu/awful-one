package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"naevis/infra"
	"naevis/models"
	"naevis/utils"

	"github.com/golang-jwt/jwt/v5"
	"github.com/julienschmidt/httprouter"
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

			// Generic duplicate check
			if strings.Contains(strings.ToLower(err.Error()), "duplicate") ||
				strings.Contains(strings.ToLower(err.Error()), "exists") {

				utils.RespondWithError(w, http.StatusConflict, "User already exists")
				return
			}

			utils.RespondWithError(w, http.StatusInternalServerError, "Registration failed")
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
			map[string]any{
				"username": creds.Username,
			},
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
			map[string]any{
				"userid": user.UserID,
			},
			map[string]any{
				"refresh_token":  hashRefreshToken(refreshToken),
				"refresh_expiry": time.Now().Add(RefreshTokenTTL),
				"refresh_ua":     uaHash(r),
				"refresh_ip":     ipPrefix(ip),
				"last_login":     time.Now(),
				"online":         true,
				"updated_at":     time.Now(),
			},
		)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Session error")
			return
		}

		/* ---------------- Set Refresh Cookie ---------------- */

		setRefreshCookie(w, refreshToken)

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
				map[string]any{
					"refresh_token": hashed,
				},
				&user,
			)

			_ = app.DB.Update(
				ctx,
				UsersCollection,
				map[string]any{
					"refresh_token": hashed,
				},
				map[string]any{
					"refresh_token":  nil,
					"refresh_expiry": nil,
					"online":         false,
					"updated_at":     time.Now(),
				},
			)
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
			map[string]any{
				"userid": claims.UserID,
			},
			map[string]any{
				"refresh_token":  nil,
				"refresh_prev":   nil,
				"refresh_expiry": nil,
				"refresh_ua":     nil,
				"refresh_ip":     nil,
				"online":         false,
				"updated_at":     time.Now(),
			},
		)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Logout failed")
			return
		}

		clearRefreshCookie(w)

		utils.RespondWithJSON(w, http.StatusOK, map[string]any{
			"message": "All sessions revoked",
		})
	}
}
