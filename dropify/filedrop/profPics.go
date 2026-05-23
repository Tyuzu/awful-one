// dropify/filedrop/profPics.go

package filedrop

import (
	"fmt"
	"naevis/dropify/filemgr"
	"naevis/models"
	"net/http"
)

// SaveProfilePictureFile saves a profile picture and returns the file information
// The actual database update should be handled by the backend service via events
func SaveProfilePictureFile(r *http.Request, claims *models.Claims) (map[string]string, error) {
	result := make(map[string]string)

	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		return nil, fmt.Errorf("error parsing form data: %w", err)
	}

	file, header, err := r.FormFile("avatar_picture")
	if err != nil {
		return nil, fmt.Errorf("avatar upload failed: %w", err)
	}
	defer file.Close()

	origName, thumbName, err := filemgr.SaveImageWithThumb(file, header, filemgr.EntityUser, filemgr.PicPhoto, 100, claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("save image with thumb failed: %w", err)
	}

	result["avatar"] = origName
	result["profile_thumb"] = thumbName

	return result, nil
}
