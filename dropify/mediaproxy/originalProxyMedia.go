package mediaproxy

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"image"
	"image/jpeg"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/disintegration/imaging"
	"github.com/julienschmidt/httprouter"

	_ "image/gif"
	_ "image/png"
)

const (
	CacheDir          = "./mediacache/media"
	CacheMaxAge       = 72 * time.Hour
	ClientTimeout     = 12 * time.Second
	MaxPixelsAllowed  = 4096 * 4096       // 16M
	StreamThresholdPx = 8 * 1024 * 1024   // stream if >8MP
	MaxImageBytes     = 200 * 1024 * 1024 // 200MB cap
	ProbeLimit        = 64 * 1024
	MaxFetchers       = 8
	MaxEncoders       = 6
)

var (
	httpClient = &http.Client{Timeout: ClientTimeout}

	privatePrefixes = []string{
		"localhost", "127.", "10.", "192.168.", "172.",
	}

	DomainBlocklist = map[string]bool{}
	DomainAllowlist = map[string]bool{}

	fetchSem = make(chan struct{}, MaxFetchers)
	encSem   = make(chan struct{}, MaxEncoders)
)

func ProxyHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	target, err := normalizeTarget(ps.ByName("url"))
	if err != nil {
		http.Error(w, "invalid url", http.StatusBadRequest)
		return
	}

	u, _ := url.Parse(target)
	if !isAllowedHost(u.Hostname()) {
		http.Error(w, "blocked host", http.StatusForbidden)
		return
	}

	_ = os.MkdirAll(CacheDir, 0755)

	cacheKey := target + "?" + r.URL.RawQuery
	cachePath := filepath.Join(CacheDir, hashURL(cacheKey))

	// serve cache
	if fi, err := os.Stat(cachePath); err == nil && time.Since(fi.ModTime()) < CacheMaxAge {
		http.ServeFile(w, r, cachePath)
		return
	}

	// params
	wParam, _ := strconv.Atoi(r.URL.Query().Get("w"))
	hParam, _ := strconv.Atoi(r.URL.Query().Get("h"))
	qParam, _ := strconv.Atoi(r.URL.Query().Get("q"))
	if qParam <= 0 {
		qParam = 80
	}

	format := strings.ToLower(r.URL.Query().Get("format"))
	if format != "webp" && format != "avif" {
		format = "jpeg"
	}

	// fetch
	resp, err := fetch(target)
	if err != nil {
		http.Error(w, "fetch failed", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		http.Error(w, "upstream error", http.StatusBadGateway)
		return
	}

	contentType := resp.Header.Get("Content-Type")

	// non-image passthrough
	if !strings.HasPrefix(contentType, "image/") {
		if err := streamToCache(resp.Body, cachePath); err != nil {
			http.Error(w, "stream fail", http.StatusBadGateway)
			return
		}
		w.Header().Set("Cache-Control", "public, max-age=86400")
		http.ServeFile(w, r, cachePath)
		return
	}

	// probe
	head, err := io.ReadAll(io.LimitReader(resp.Body, ProbeLimit))
	if err != nil && err != io.EOF {
		http.Error(w, "probe fail", http.StatusBadGateway)
		return
	}

	cfg, _, err := image.DecodeConfig(bytes.NewReader(head))
	if err != nil {
		// fallback stream
		streamFallback(w, r, cachePath, contentType, head, resp.Body)
		return
	}

	totalPx := int64(cfg.Width) * int64(cfg.Height)
	if totalPx > StreamThresholdPx || totalPx > MaxPixelsAllowed {
		streamFallback(w, r, cachePath, contentType, head, resp.Body)
		return
	}

	// read full image (bounded)
	full, err := io.ReadAll(io.MultiReader(bytes.NewReader(head), resp.Body))
	if err != nil || int64(len(full)) > MaxImageBytes {
		streamFallback(w, r, cachePath, contentType, head, resp.Body)
		return
	}

	img, _, err := image.Decode(bytes.NewReader(full))
	if err != nil {
		http.Error(w, "decode error", http.StatusBadGateway)
		return
	}

	// resize
	if wParam > 0 || hParam > 0 {
		img = imaging.Resize(img, wParam, hParam, imaging.Lanczos)
	}

	// encode
	data, ct, err := encode(img, format, qParam)
	if err != nil {
		http.Error(w, "encode fail", http.StatusBadGateway)
		return
	}

	if err := saveAtomically(cachePath, data); err != nil {
		w.Header().Set("Content-Type", ct)
		w.Write(data)
		return
	}

	w.Header().Set("Content-Type", ct)
	w.Header().Set("Cache-Control", "public, max-age=86400")
	http.ServeFile(w, r, cachePath)
}

func streamFallback(w http.ResponseWriter, r *http.Request, path, ct string, head []byte, body io.Reader) {
	_ = streamToCache(io.MultiReader(bytes.NewReader(head), body), path)
	w.Header().Set("Content-Type", ct)
	w.Header().Set("Cache-Control", "public, max-age=86400")
	http.ServeFile(w, r, path)
}

func fetch(target string) (*http.Response, error) {
	fetchSem <- struct{}{}
	defer func() { <-fetchSem }()

	req, err := http.NewRequest("GET", target, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "MediaProxy/1.0")
	req.Header.Set("Accept", "*/*")

	return httpClient.Do(req)
}

func normalizeTarget(raw string) (string, error) {
	raw = strings.TrimPrefix(raw, "/")

	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		return raw, nil
	}

	// support original scheme-less format
	raw = strings.Replace(raw, "/", "://", 1)

	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return "", errors.New("invalid url")
	}
	return raw, nil
}

func isPrivateHost(host string) bool {
	host = strings.ToLower(host)
	if host == "localhost" || host == "::1" {
		return true
	}
	for _, p := range privatePrefixes {
		if strings.HasPrefix(host, p) {
			return true
		}
	}
	return false
}

func isAllowedHost(host string) bool {
	if isPrivateHost(host) {
		return false
	}
	if len(DomainAllowlist) > 0 {
		return DomainAllowlist[host]
	}
	return !DomainBlocklist[host]
}

func hashURL(u string) string {
	h := sha1.New()
	h.Write([]byte(u))
	return hex.EncodeToString(h.Sum(nil))
}

func streamToCache(r io.Reader, path string) error {
	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	_, err = io.Copy(f, r)
	f.Close()
	if err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return os.Rename(tmp, path)
}

func saveAtomically(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func encode(img image.Image, _ string, quality int) ([]byte, string, error) {
	encSem <- struct{}{}
	defer func() { <-encSem }()

	var buf bytes.Buffer

	// future formats can be added here safely
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
		return nil, "", err
	}
	return buf.Bytes(), "image/jpeg", nil
}
