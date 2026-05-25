package models

import "github.com/golang-jwt/jwt/v5"

// Claims defines the JWT claims structure
type Claims struct {
	Username string   `json:"username"`
	UserID   string   `json:"userId"`
	Role     []string `json:"role"`
	jwt.RegisteredClaims
}

// "session_version": user.SessionVersion + 1
