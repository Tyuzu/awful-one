package auth

import (
	"context"
	"fmt"
	"naevis/infra"
	"naevis/models"
	"naevis/utils"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/bson"
)

/* ============================================================
   REFRESH TOKEN (STRICT, COOKIE LIFECYCLE HANDLED IN HANDLER)
============================================================ */

// RefreshResult communicates intended cookie side-effects and tokens.
type RefreshResult struct {
	AccessToken string
	NewRefresh  string // non-empty => set this new refresh in cookie
	ClearCookie bool   // true => clear cookie on response
}

// RefreshToken handler: reads cookie, delegates logic, and applies cookie changes exactly once.
func RefreshToken(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		cookie, err := r.Cookie("refresh_token")
		if err != nil || cookie.Value == "" {
			utils.RespondWithError(w, http.StatusUnauthorized, "Missing refresh token")
			return
		}

		result, err := RefreshTokenFromCookie(ctx, cookie.Value, r, app)
		if err != nil {
			if result != nil && result.ClearCookie {
				clearRefreshCookie(w)
			}
			utils.RespondWithError(w, http.StatusUnauthorized, err.Error())
			return
		}

		// Apply cookie changes (single place)
		if result.NewRefresh != "" {
			setRefreshCookie(w, result.NewRefresh)
		}

		utils.RespondWithJSON(w, http.StatusOK, map[string]any{
			"message": "Token refreshed successfully",
			"data": map[string]string{
				"token": result.AccessToken,
			},
		})
	}
}

// RefreshTokenFromCookie performs DB checks/rotation but does NOT modify HTTP response.
// It returns a RefreshResult that the handler should apply to the outgoing response.
func RefreshTokenFromCookie(ctx context.Context, rawToken string, r *http.Request, app *infra.Deps) (*RefreshResult, error) {

	now := time.Now()
	hashed := hashRefreshToken(rawToken)

	// -----------------------
	// Find valid refresh session
	// -----------------------
	var user models.User
	err := app.DB.FindOne(ctx, UsersCollection, bson.M{
		"refresh_expiry": bson.M{"$gt": now},
		"$or": []bson.M{
			{"refresh_token": hashed},
			{"refresh_prev": hashed},
		},
	}, &user)

	if err != nil {
		// Invalid or expired token
		return &RefreshResult{ClearCookie: true}, fmt.Errorf("invalid refresh token")
	}

	// -----------------------
	// Refresh token reuse detection
	// -----------------------
	if user.RefreshPrev == hashed {
		// Invalidate entire session
		_ = app.DB.Update(
			ctx,
			UsersCollection,
			bson.M{"userid": user.UserID},
			bson.M{
				"$set": bson.M{
					"refresh_token":  nil,
					"refresh_prev":   nil,
					"refresh_expiry": nil,
					"refresh_ua":     nil,
					"updated_at":     now,
				},
			},
		)

		return &RefreshResult{ClearCookie: true}, fmt.Errorf("refresh token reuse detected")
	}

	// -----------------------
	// UA binding validation
	// -----------------------
	if user.RefreshUA != uaHash(r) {
		_ = app.DB.Update(
			ctx,
			UsersCollection,
			bson.M{"userid": user.UserID},
			bson.M{
				"$set": bson.M{
					"refresh_token":  nil,
					"refresh_prev":   nil,
					"refresh_expiry": nil,
					"refresh_ua":     nil,
					"updated_at":     now,
				},
			},
		)

		return &RefreshResult{ClearCookie: true}, fmt.Errorf("session invalidated")
	}

	// -----------------------
	// Issue new access token
	// -----------------------
	claims := &models.Claims{
		UserID:   user.UserID,
		Username: user.Username,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(AccessTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	accessToken, err := createAccessToken(claims)
	if err != nil {
		return nil, err
	}

	// -----------------------
	// Rotate refresh token
	// -----------------------
	newRefresh, err := generateRefreshToken()
	if err != nil {
		return nil, err
	}

	err = app.DB.Update(
		ctx,
		UsersCollection,
		bson.M{"userid": user.UserID},
		bson.M{
			"$set": bson.M{
				"refresh_prev":   user.RefreshToken,
				"refresh_token":  hashRefreshToken(newRefresh),
				"refresh_expiry": now.Add(RefreshTokenTTL),
				"refresh_ua":     uaHash(r),
				"updated_at":     now,
			},
		},
	)
	if err != nil {
		return nil, err
	}

	return &RefreshResult{
		AccessToken: accessToken,
		NewRefresh:  newRefresh,
	}, nil
}
