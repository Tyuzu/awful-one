package services

import (
	"fmt"
	"mime/multipart"
	"naevis/dropify/filedrop"
	"naevis/dropify/filemgr"
	"net/http"
	"strings"
)

// FileService provides file operation abstractions
type FileService struct{}

// NewFileService creates a new FileService instance
func NewFileService() *FileService {
	return &FileService{}
}

// Attachment represents a processed file attachment
type Attachment struct {
	Filename    string `json:"filename"`
	Extension   string `json:"extension"`
	Key         string `json:"key"`
	Resolutions []int  `json:"resolutions,omitempty"`
}

// ProcessUploadedFiles processes multipart uploads
func (fs *FileService) ProcessUploadedFiles(
	r *http.Request,
	entityType string,
	entityId string,
) ([]Attachment, error) {

	_ = entityId // reserved for future use

	if r.MultipartForm == nil || len(r.MultipartForm.File) == 0 {
		return nil, fmt.Errorf("no files provided")
	}

	entity := filemgr.EntityType(strings.ToLower(entityType))

	var attachments []Attachment

	for fieldKey, files := range r.MultipartForm.File {

		keyLower := strings.ToLower(strings.TrimSpace(fieldKey))

		for _, fileHeader := range files {

			atts, err := fs.processSingleFile(
				r,
				fileHeader,
				keyLower,
				entity,
			)

			if err != nil {
				return nil, fmt.Errorf(
					"failed to process file %s: %w",
					fileHeader.Filename,
					err,
				)
			}

			attachments = append(attachments, atts...)
		}
	}

	return attachments, nil
}

// processSingleFile routes uploads by field key
func (fs *FileService) processSingleFile(
	r *http.Request,
	fileHeader *multipart.FileHeader,
	fieldKey string,
	entity filemgr.EntityType,
) ([]Attachment, error) {

	// Feed media special handling
	if fieldKey == "feed" {
		return fs.processFeedFile(r, fileHeader, fieldKey)
	}

	// Regular uploads
	return fs.processRegularFile(fileHeader, fieldKey, entity)
}

// processFeedFile handles feed media uploads
func (fs *FileService) processFeedFile(
	r *http.Request,
	fileHeader *multipart.FileHeader,
	fieldKey string,
) ([]Attachment, error) {

	postType := strings.ToLower(strings.TrimSpace(r.FormValue("postType")))

	if postType == "" {
		postType = "video"
	}

	picType := postTypeToImageType(postType)

	// Video / audio processing
	if postType == "video" || postType == "audio" {

		savedPath, uniqueID, ext, err := filedrop.SaveUploadedFile(
			fileHeader,
			filemgr.EntityFeed,
			picType,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to save file: %w", err)
		}

		uploadDir := filemgr.ResolvePath(filemgr.EntityFeed, picType)

		// Video
		if postType == "video" {

			resolutions, _, err := filedrop.ProcessVideo(
				r,
				savedPath,
				uploadDir,
				uniqueID,
				filemgr.EntityFeed,
			)

			if err != nil {
				return nil, fmt.Errorf("video processing failed: %w", err)
			}

			return []Attachment{
				{
					Filename:    uniqueID,
					Extension:   ext,
					Key:         fieldKey,
					Resolutions: resolutions,
				},
			}, nil
		}

		// Audio
		resolutions, _ := filedrop.ProcessAudio(
			savedPath,
			uploadDir,
			uniqueID,
			filemgr.EntityFeed,
		)

		return []Attachment{
			{
				Filename:    uniqueID,
				Extension:   ext,
				Key:         fieldKey,
				Resolutions: resolutions,
			},
		}, nil
	}

	// Posters/images
	return fs.processRegularFile(
		fileHeader,
		fieldKey,
		filemgr.EntityFeed,
	)
}

// processRegularFile handles standard uploads
func (fs *FileService) processRegularFile(
	fileHeader *multipart.FileHeader,
	fieldKey string,
	entity filemgr.EntityType,
) ([]Attachment, error) {

	file, err := fileHeader.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	picType, ok := map[string]filemgr.PictureType{
		"banner":   filemgr.PicBanner,
		"photo":    filemgr.PicPhoto,
		"avatar":   filemgr.PicPhoto,
		"seating":  filemgr.PicSeating,
		"poster":   filemgr.PicPoster,
		"thumb":    filemgr.PicThumb,
		"document": filemgr.PicDocument,
		"audio":    filemgr.PicAudio,
		"video":    filemgr.PicVideo,
		"song":     filemgr.PicSong,
	}[fieldKey]

	if !ok {
		picType = filemgr.PicPhoto
	}

	savedName, ext, err := filemgr.SaveFileForEntity(
		file,
		fileHeader,
		entity,
		picType,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to save file: %w", err)
	}

	return []Attachment{
		{
			Filename:  savedName + ext,
			Extension: ext,
			Key:       fieldKey,
		},
	}, nil
}

// postTypeToImageType maps feed media types
func postTypeToImageType(postType string) filemgr.PictureType {

	switch strings.ToLower(postType) {

	case "audio":
		return filemgr.PicAudio

	case "video":
		return filemgr.PicVideo

	case "poster":
		return filemgr.PicPoster

	case "banner":
		return filemgr.PicBanner

	case "document":
		return filemgr.PicDocument

	case "song":
		return filemgr.PicSong

	default:
		return filemgr.PicPhoto
	}
}
