package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"naevis/globals"
	"naevis/models"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	RefreshTokenTTL = 7 * 24 * time.Hour
	AccessTokenTTL  = 15 * time.Minute

	maxFailedAttempts = 5
	lockoutDuration   = 10 * time.Minute
)

var (
	usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]{3,20}$`)
	emailRegex    = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)
)

/* ============================================================
   Helpers
============================================================ */

func clientIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		return strings.TrimSpace(strings.Split(fwd, ",")[0])
	}
	if rip := r.Header.Get("X-Real-IP"); rip != "" {
		return rip
	}
	host, _, _ := net.SplitHostPort(r.RemoteAddr)
	return host
}

func ipPrefix(ip string) string {
	if strings.Contains(ip, ":") {
		return ip
	}
	parts := strings.Split(ip, ".")
	if len(parts) < 2 {
		return ip
	}
	return parts[0] + "." + parts[1]
}

func uaHash(r *http.Request) string {
	sum := sha256.Sum256([]byte(r.UserAgent()))
	return hex.EncodeToString(sum[:])
}

func hashRefreshToken(token string) string {
	mac := hmac.New(sha256.New, globals.RefreshTokenSecret)
	mac.Write([]byte(token))
	return hex.EncodeToString(mac.Sum(nil))
}

func generateRefreshToken() (string, error) {
	b := make([]byte, 64)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func createAccessToken(claims *models.Claims) (string, error) {
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString(globals.JwtSecret)
}

func setRefreshCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteNoneMode,
		Secure:   true,
		Expires:  time.Now().Add(RefreshTokenTTL),
		MaxAge:   int(RefreshTokenTTL.Seconds()),
	})
}

// func setRefreshCookie(w http.ResponseWriter, token string) {
// 	sameSite, secure := refreshCookieAttrs()

// 	http.SetCookie(w, &http.Cookie{
// 		Name:     "refresh_token",
// 		Value:    token,
// 		Path:     "/",
// 		HttpOnly: true,
// 		Secure:   secure,
// 		SameSite: sameSite,
// 		Expires:  time.Now().Add(RefreshTokenTTL),
// 		MaxAge:   int(RefreshTokenTTL.Seconds()),
// 	})
// }

func clearRefreshCookie(w http.ResponseWriter) {
	sameSite, secure := http.SameSiteNoneMode, true

	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: sameSite,
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
	})
}

/* ============================================================
   Validators
============================================================ */

func validateUsername(u string) bool { return usernameRegex.MatchString(u) }
func validateEmail(e string) bool    { return emailRegex.MatchString(e) }
func validatePassword(p string) bool { return len(p) >= 6 }
