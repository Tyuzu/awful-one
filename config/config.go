package config

import (
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	STRIPE_WEBHOOK_SECRET string
	HTTPPort              string
	AllowedOrigins        []string
	RTMPIngestURL         string
	TURNServers           string
	CDNBaseURL            string
}

func InitConfig() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found; using system environment")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = ":4000"
	} else if port[0] != ':' {
		port = ":" + port
	}

	allowedOrigins := parseAllowedOrigins(os.Getenv("ALLOWED_ORIGINS"))

	return &Config{
		HTTPPort:              port,
		AllowedOrigins:        allowedOrigins,
		STRIPE_WEBHOOK_SECRET: os.Getenv("STRIPE_WEBHOOK_SECRET"),
		RTMPIngestURL:         os.Getenv("RTMP_INGEST_URL"),
		TURNServers:           os.Getenv("TURN_SERVERS"),
		CDNBaseURL:            os.Getenv("CDN_BASE_URL"),
	}
}

func parseAllowedOrigins(env string) []string {
	if env == "" {
		return []string{"http://localhost:5173", "https://indium.netlify.app"}
	}
	parts := strings.Split(env, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
