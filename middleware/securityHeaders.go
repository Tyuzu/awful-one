package middleware

import (
	"net/http"
	"path"
	"strings"
)

// SecurityHeaders applies security headers with different policies
// for static assets and dynamic/API responses.
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()

		urlPath := strings.ToLower(r.URL.Path)
		ext := strings.ToLower(path.Ext(urlPath))

		// Static asset detection
		staticExts := map[string]bool{
			".png":   true,
			".jpg":   true,
			".jpeg":  true,
			".webp":  true,
			".gif":   true,
			".svg":   true,
			".ico":   true,
			".mp4":   true,
			".mp3":   true,
			".wav":   true,
			".webm":  true,
			".ttf":   true,
			".woff":  true,
			".woff2": true,
		}

		isStatic := staticExts[ext]

		// ---------------------------------------------------------------------
		// Common headers
		// ---------------------------------------------------------------------

		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		// HTTPS only
		if r.TLS != nil {
			h.Set(
				"Strict-Transport-Security",
				"max-age=31536000; includeSubDomains; preload",
			)
		}

		// ---------------------------------------------------------------------
		// Static assets
		// ---------------------------------------------------------------------

		if isStatic {
			h.Set("Cache-Control", "public, max-age=31536000, immutable")

			// Optional iframe support for same-origin embeds
			h.Set("X-Frame-Options", "SAMEORIGIN")

			// Minimal CSP for assets
			h.Set("Content-Security-Policy", "default-src 'self'")

			// Allow external embedding/CDN usage
			h.Del("Cross-Origin-Opener-Policy")
			h.Del("Cross-Origin-Resource-Policy")
			h.Del("Pragma")
			h.Del("Expires")

			next.ServeHTTP(w, r)
			return
		}

		// ---------------------------------------------------------------------
		// Dynamic / API / HTML responses
		// ---------------------------------------------------------------------

		h.Set("X-Frame-Options", "DENY")

		h.Set(
			"Content-Security-Policy",
			"default-src 'self'; "+
				"object-src 'none'; "+
				"base-uri 'self'; "+
				"frame-ancestors 'none'; "+
				"form-action 'self'; "+
				"media-src 'self'; "+
				"block-all-mixed-content;",
		)

		h.Set("Cross-Origin-Opener-Policy", "same-origin")
		h.Set("Cross-Origin-Resource-Policy", "same-origin")

		// Prevent caching of authenticated/private responses
		h.Set("Cache-Control", "no-store, no-cache, must-revalidate, private")
		h.Set("Pragma", "no-cache")
		h.Set("Expires", "0")

		next.ServeHTTP(w, r)
	})
}
