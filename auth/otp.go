package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log"
	"naevis/infra"
	"naevis/utils"
	"net/http"
	"net/smtp"
	"os"
	"strings"
	"time"

	"github.com/julienschmidt/httprouter"
	"go.mongodb.org/mongo-driver/bson"
)

const OTPExpiry = 10 * time.Minute

func hashPlainSHA256(s string) string {
	h := sha256.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}

// ========================
// OTP / EMAIL
// ========================

func GenerateOTP(length int) (string, error) {
	if length <= 0 {
		return "", errors.New("invalid length")
	}
	const digits = "0123456789"
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	for i := 0; i < length; i++ {
		b[i] = digits[int(b[i])%len(digits)]
	}
	return string(b), nil
}

func SendEmailOTP(toEmail, otp string) error {
	go func() {
		from := os.Getenv("SMTP_USER")
		pass := os.Getenv("SMTP_PASS")
		host := os.Getenv("SMTP_HOST")
		port := os.Getenv("SMTP_PORT")
		if from == "" || pass == "" || host == "" || port == "" {
			log.Printf("warn: SMTP not configured")
			return
		}
		msg := []byte("Subject: Email Verification\n\nYour OTP is: " + otp + "\nIt expires in 10 minutes.\n")
		auth := smtp.PlainAuth("", from, pass, host)
		if err := smtp.SendMail(host+":"+port, auth, from, []string{toEmail}, msg); err != nil {
			log.Printf("warn: failed to send OTP email: %v", err)
		}
	}()
	return nil
}

type RequestOTPInput struct {
	Email string `json:"email"`
}

type VerifyOTPInput struct {
	Email string `json:"email"`
	OTP   string `json:"otp"`
}

func RequestOTPHandler(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		var input RequestOTPInput
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()

		if err := dec.Decode(&input); err != nil || input.Email == "" {
			utils.RespondWithError(w, http.StatusBadRequest, "Invalid input")
			return
		}

		email := strings.ToLower(strings.TrimSpace(input.Email))

		otp, err := GenerateOTP(6)
		if err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to generate OTP")
			return
		}

		hashed := hashPlainSHA256(otp)
		key := "otp:" + email

		if err = app.Cache.SetWithExpiry(ctx, key, []byte(hashed), OTPExpiry); err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to store OTP")
			return
		}

		if err = SendEmailOTP(email, otp); err != nil {
			_ = app.Cache.Del(ctx, key)
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to send OTP")
			return
		}

		utils.RespondWithJSON(w, http.StatusOK, map[string]string{
			"message": "OTP sent if the email exists",
		})
	}
}

func VerifyOTPHandler(app *infra.Deps) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		var input VerifyOTPInput
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()

		if err := dec.Decode(&input); err != nil || input.Email == "" || input.OTP == "" {
			utils.RespondWithError(w, http.StatusBadRequest, "Invalid input")
			return
		}

		email := strings.ToLower(strings.TrimSpace(input.Email))
		key := "otp:" + email

		stored, err := app.Cache.Get(ctx, key)
		if err != nil || len(stored) == 0 {
			utils.RespondWithError(w, http.StatusUnauthorized, "Invalid or expired OTP")
			return
		}

		expected := hashPlainSHA256(input.OTP)
		if subtle.ConstantTimeCompare([]byte(stored), []byte(expected)) != 1 {
			utils.RespondWithError(w, http.StatusUnauthorized, "Invalid or expired OTP")
			return
		}

		if err = app.DB.Update(
			ctx,
			UsersCollection,
			bson.M{"email": email},
			bson.M{"$set": bson.M{"email_verified": true}},
		); err != nil {
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to verify user")
			return
		}

		_ = app.Cache.Del(ctx, key)

		utils.RespondWithJSON(w, http.StatusOK, map[string]string{
			"message": "User verified successfully",
		})
	}
}
