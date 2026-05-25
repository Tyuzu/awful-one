package config

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	StripeWebhookSecret string   `json:"STRIPE_WEBHOOK_SECRET"`
	HTTPPort            string   `json:"PORT"`
	AllowedOrigins      []string `json:"ALLOWED_ORIGINS"`
	RTMPIngestURL       string   `json:"RTMP_INGEST_URL"`
	TURNServers         string   `json:"TURN_SERVERS"`
	CDNBaseURL          string   `json:"CDN_BASE_URL"`
}

var defaultAllowedOrigins = []string{
	"http://localhost:5173",
	"https://indium.netlify.app",
}

func InitConfig() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found; using system environment")
	}
	configURL := mustGetEnv("CONFIG_URL")

	cfg := mustFetchConfig(configURL)

	cfg.HTTPPort = normalizePort(cfg.HTTPPort)

	if len(cfg.AllowedOrigins) == 0 {
		cfg.AllowedOrigins = defaultAllowedOrigins
	}

	return cfg
}

func mustGetEnv(key string) string {
	value := os.Getenv(key)

	if value == "" {
		panic(fmt.Sprintf("%s is not set", key))
	}

	return value
}

func mustFetchConfig(url string) *Config {
	client := http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(url)
	must(err)

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		panic(fmt.Sprintf("unexpected status: %s", resp.Status))
	}

	body, err := io.ReadAll(resp.Body)
	must(err)

	var cfg Config

	must(json.Unmarshal(body, &cfg))

	return &cfg
}

func normalizePort(port string) string {
	switch {
	case port == "":
		return ":4000"

	case strings.HasPrefix(port, ":"):
		return port

	default:
		return ":" + port
	}
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
