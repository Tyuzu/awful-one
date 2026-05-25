package config

import "time"

const (
	UploadDir       = "static/uploads"
	TempUploadDir   = "static/temp"
	ChunkUploadSize = 2 * 1024 * 1024
	CleanupAge      = 1 * time.Hour
	ChunkBuffer     = 4 * 1024 * 1024
)
