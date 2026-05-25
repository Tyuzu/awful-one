package utils

import (
	rndm "math/rand"
	"net/http"

	"github.com/julienschmidt/httprouter"
)

// --- CSRF Token Generation ---

func CSRF(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	csrf := GenerateRandomString(12)
	RespondWithJSON(w, http.StatusOK, map[string]any{
		"ok":         true,
		"csrf_token": csrf,
	})
}

// --- Random String and ID Generators ---

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyz0123456789_ABCDEFGHIJKLMNOPQRSTUVWXYZ")
var digitRunes = []rune("0123456789")

// GenerateRandomString creates a random alphanumeric string of length n.
func GenerateRandomString(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rndm.Intn(len(letterRunes))]
	}
	return string(b)
}

// GenerateRandomDigitString creates a random numeric string of length n.
func GenerateRandomDigitString(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = digitRunes[rndm.Intn(len(digitRunes))]
	}
	return string(b)
}
