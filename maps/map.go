package maps

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"sync"

	// adjust to real import path for your rate limiter

	"github.com/julienschmidt/httprouter"
)

// Marker represents a single map marker
type Marker struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
	X    int    `json:"x"` // pixel X on native map grid (0..mapWidth)
	Y    int    `json:"y"` // pixel Y on native map grid (0..mapHeight)
}

// LockedArea represents a blocked/unlocked zone
type LockedArea struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	X         int    `json:"x"`
	Y         int    `json:"y"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	Condition string `json:"condition"` // e.g. "mission>=5"
	DependsOn string `json:"dependsOn"` // e.g. "ls"
}

// MapConfig includes map setup for a particular entity
type MapConfig struct {
	Entity      string       `json:"entity"`
	MapImage    string       `json:"mapImage"`
	MapWidth    int          `json:"mapWidth"`
	MapHeight   int          `json:"mapHeight"`
	LockedAreas []LockedArea `json:"lockedAreas"`
}

// --- Player progress (thread-safe) ---
var (
	playerMissionsMu sync.Mutex
	// missions completed per entity (fake store; replace with DB)
	playerMissionsCompleted = map[string]int{
		"ls": 3, // sample starting values; change as needed
		"sf": 0,
		"lv": 0,
	}
)

// --- Static configs & markers (native 1200x600 coordinates) ---
const nativeMapWidth = 1200
const nativeMapHeight = 600

var mapConfigs = map[string]MapConfig{
	"ls": {
		Entity:    "ls",
		MapImage:  "/assets/ls-map.jpg",
		MapWidth:  nativeMapWidth,
		MapHeight: nativeMapHeight,
		LockedAreas: []LockedArea{
			{ID: "sf", Label: "San Fierro (Locked)", X: 300, Y: 100, Width: 200, Height: 150, Condition: "mission>=5", DependsOn: "ls"},
			{ID: "lv_hint", Label: "Las Venturas access locked (see SF)", X: 850, Y: 50, Width: 200, Height: 120, Condition: "", DependsOn: "sf"},
		},
	},
	"sf": {
		Entity:    "sf",
		MapImage:  "/assets/sf-map.jpg",
		MapWidth:  nativeMapWidth,
		MapHeight: nativeMapHeight,
		LockedAreas: []LockedArea{
			{ID: "lv", Label: "Las Venturas (Locked)", X: 600, Y: 200, Width: 220, Height: 160, Condition: "mission>=10", DependsOn: "sf"},
		},
	},
	"lv": {
		Entity:      "lv",
		MapImage:    "/assets/lv-map.jpg",
		MapWidth:    nativeMapWidth,
		MapHeight:   nativeMapHeight,
		LockedAreas: []LockedArea{},
	},
}

var mapMarkers = map[string][]Marker{
	"ls": {
		{ID: "m1", Name: "CJ's House", Type: "house", X: 120, Y: 140},
		{ID: "m2", Name: "Big Smoke Mission", Type: "mission", X: 250, Y: 200},
		{ID: "m3", Name: "Gun Shop", Type: "shop", X: 400, Y: 300},
		{ID: "m4", Name: "Rival Gang", Type: "enemy", X: 350, Y: 250},
	},
	"sf": {
		{ID: "m5", Name: "Garage", Type: "shop", X: 150, Y: 200},
		{ID: "m6", Name: "Woozie Mission", Type: "mission", X: 300, Y: 250},
	},
	"lv": {
		{ID: "m7", Name: "Casino", Type: "shop", X: 200, Y: 180},
		{ID: "m8", Name: "Heist Mission", Type: "mission", X: 350, Y: 260},
	},
}

// --- Handlers ---

// GetMapConfig returns entity-specific map config with active locks filtered by player's progress.
// The response includes a snapshot of player progress under "playerProgress".
func GetMapConfig(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	entity := ps.ByName("entity")
	cfg, ok := mapConfigs[entity]
	if !ok {
		http.Error(w, "entity not found", http.StatusNotFound)
		return
	}

	// get snapshot of player progress
	playerMissionsMu.Lock()
	progressCopy := make(map[string]int, len(playerMissionsCompleted))
	for k, v := range playerMissionsCompleted {
		progressCopy[k] = v
	}
	playerMissionsMu.Unlock()

	// filter locked areas
	var activeLocks []LockedArea
	for _, area := range cfg.LockedAreas {
		if !isUnlockedForEntity(area, progressCopy, entity) {
			activeLocks = append(activeLocks, area)
		}
	}
	cfg.LockedAreas = activeLocks

	// response with config + progress snapshot
	resp := struct {
		Config         MapConfig      `json:"config"`
		PlayerProgress map[string]int `json:"playerProgress"`
	}{
		Config:         cfg,
		PlayerProgress: progressCopy,
	}

	writeJSON(w, resp)
}

// GetMapMarkers returns markers for the entity (native pixel coords)
func GetMapMarkers(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	entity := ps.ByName("entity")
	markers, ok := mapMarkers[entity]
	if !ok {
		http.Error(w, "entity not found", http.StatusNotFound)
		return
	}
	writeJSON(w, markers)
}

// UpdatePlayerProgress increments the missions completed for an entity
// POST /api/v1/player/progress?entity=ls
func UpdatePlayerProgress(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// read entity from query param (default to "ls")
	entity := r.URL.Query().Get("entity")
	if entity == "" {
		entity = "ls"
	}
	// validate entity exists
	if _, ok := mapConfigs[entity]; !ok {
		http.Error(w, "entity not found", http.StatusBadRequest)
		return
	}

	// increment progress safely
	playerMissionsMu.Lock()
	playerMissionsCompleted[entity]++
	newVal := playerMissionsCompleted[entity]
	playerMissionsMu.Unlock()

	writeJSON(w, map[string]interface{}{"entity": entity, "missionsCompleted": newVal})
}

// GetPlayerProgress returns player progress. If ?entity= is provided, return that entity's count,
// otherwise return the whole progress map.
func GetPlayerProgress(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	entity := r.URL.Query().Get("entity")

	playerMissionsMu.Lock()
	defer playerMissionsMu.Unlock()

	if entity != "" {
		if _, ok := playerMissionsCompleted[entity]; !ok {
			http.Error(w, "entity not found", http.StatusBadRequest)
			return
		}
		writeJSON(w, map[string]interface{}{"entity": entity, "missionsCompleted": playerMissionsCompleted[entity]})
		return
	}

	// return full snapshot
	copyMap := make(map[string]int, len(playerMissionsCompleted))
	for k, v := range playerMissionsCompleted {
		copyMap[k] = v
	}
	writeJSON(w, copyMap)
}

// --- Helpers ---

// isUnlockedForEntity checks dependency and condition for a locked area relative to an entity and progress map.
func isUnlockedForEntity(area LockedArea, progress map[string]int, currentEntity string) bool {
	checkEntity := area.DependsOn
	if checkEntity == "" {
		checkEntity = currentEntity
	}

	// If checkEntity not in progress map assume 0
	val := progress[checkEntity]

	// If area depends on another entity but that entity has zero progress -> locked
	if area.DependsOn != "" && progress[area.DependsOn] == 0 {
		return false
	}

	// Condition parsing: support "mission>=N"
	if strings.HasPrefix(area.Condition, "mission>=") {
		nStr := strings.TrimPrefix(area.Condition, "mission>=")
		if n, err := strconv.Atoi(nStr); err == nil {
			return val >= n
		}
		// treat parse error as locked
		return false
	}

	// no condition implies unlocked as long as dependency satisfied
	return true
}

func writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(data)
}
