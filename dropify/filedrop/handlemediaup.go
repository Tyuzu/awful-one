// dropify/filedrop/handlemediaup.go

package filedrop

import (
	"naevis/dropify/filemgr"
	"net/http"
)

func HandleMediaUpload(r *http.Request, postType string, entitytype filemgr.EntityType) (paths, names []string, resolutions []int, err error) {
	switch postType {
	case "image":
		names, err = saveUploadedFiles(r, "images", "photo", entitytype)
	case "video":
		var result *MediaResult
		result, err = saveUploadedVideoFile(r, "video", entitytype)
		if err == nil {
			resolutions, paths, names = result.Resolutions, result.Paths, result.IDs
		}
	case "audio":
		var result *MediaResult
		result, err = saveUploadedAudioFile(r, "audio", entitytype)
		if err == nil {
			resolutions, paths, names = result.Resolutions, result.Paths, result.IDs
		}
	}
	return
}

func saveUploadedVideoFile(r *http.Request, formKey string, entitytype filemgr.EntityType) (*MediaResult, error) {
	return ProcessMediaUpload(r, formKey, Video, entitytype)
}

func saveUploadedAudioFile(r *http.Request, formKey string, entitytype filemgr.EntityType) (*MediaResult, error) {
	return ProcessMediaUpload(r, formKey, Audio, entitytype)
}
