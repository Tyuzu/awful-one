package activity

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"naevis/config"
	"net/http"
	"strconv"
	"time"
)

var (
	ActivitiesCollection = config.Collections.ActivitiesCollection
	AnalyticsCollection  = config.Collections.AnalyticsCollection

	analyticsIdemTTL = 24 * time.Hour
	defaultPageSize  = 20
	maxPageSize      = 100
)

func parseCursor(r *http.Request) (time.Time, int) {
	q := r.URL.Query()

	limit := defaultPageSize
	if v := q.Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			if n > maxPageSize {
				n = maxPageSize
			}
			limit = n
		}
	}

	var cursor time.Time
	if v := q.Get("cursor"); v != "" {
		if ts, err := strconv.ParseInt(v, 10, 64); err == nil {
			cursor = time.UnixMilli(ts)
		}
	}

	return cursor, limit
}
func analyticsIdempotencyKey(ev map[string]any) string {
	raw, _ := json.Marshal(ev)
	sum := sha256.Sum256(raw)
	return "analytics:idemp:" + hex.EncodeToString(sum[:])
}
