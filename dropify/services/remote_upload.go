package services

import (
	"fmt"
	"io"
	"mime/multipart"
	"naevis/dropify/filemgr"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// ProcessRemoteFile downloads and stores a remote image
func (s *FileService) ProcessRemoteFile(
	remoteURL string,
	key string,
	entityType string,
	entityId string,
) ([]Attachment, error) {

	_ = entityId // reserved for future use

	// -------------------------
	// Validate URL
	// -------------------------

	parsed, err := url.Parse(remoteURL)
	if err != nil {
		return nil, fmt.Errorf("invalid remote URL")
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("unsupported URL scheme")
	}

	// -------------------------
	// Download file
	// -------------------------

	resp, err := http.Get(remoteURL)
	if err != nil {
		return nil, fmt.Errorf("failed to download remote file")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("remote server returned %d", resp.StatusCode)
	}

	contentType := strings.ToLower(resp.Header.Get("Content-Type"))

	if !strings.HasPrefix(contentType, "image/") {
		return nil, fmt.Errorf("remote file is not an image")
	}

	// -------------------------
	// Temp file
	// -------------------------

	tmpFile, err := os.CreateTemp("", "remote-upload-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file")
	}
	defer os.Remove(tmpFile.Name())

	// copy body
	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		tmpFile.Close()
		return nil, fmt.Errorf("failed to save remote file")
	}

	if err := tmpFile.Close(); err != nil {
		return nil, err
	}

	// -------------------------
	// Reopen temp file
	// -------------------------

	file, err := os.Open(tmpFile.Name())
	if err != nil {
		return nil, fmt.Errorf("failed to reopen temp file")
	}
	defer file.Close()

	// -------------------------
	// Resolve entity
	// -------------------------

	entity := filemgr.EntityType(strings.ToLower(entityType))

	// -------------------------
	// Resolve picture type
	// -------------------------

	picType, ok := map[string]filemgr.PictureType{
		"banner":  filemgr.PicBanner,
		"photo":   filemgr.PicPhoto,
		"avatar":  filemgr.PicPhoto,
		"seating": filemgr.PicSeating,
	}[key]

	if !ok {
		return nil, fmt.Errorf("invalid picture key")
	}

	// -------------------------
	// Create multipart header
	// -------------------------

	filename := filepath.Base(parsed.Path)

	if filename == "." || filename == "/" || filename == "" {
		filename = "remote.jpg"
	}

	header := &multipart.FileHeader{
		Filename: filename,
		Header: textproto.MIMEHeader{
			"Content-Type": []string{contentType},
		},
		Size: resp.ContentLength,
	}

	// -------------------------
	// Save through existing pipeline
	// -------------------------

	savedName, ext, err := filemgr.SaveFileForEntity(
		file,
		header,
		entity,
		picType,
	)

	if err != nil {
		return nil, err
	}

	return []Attachment{
		{
			Filename:  savedName + ext,
			Extension: ext,
			Key:       key,
		},
	}, nil
}
