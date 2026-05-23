package filemgr

import (
	"mime/multipart"
)

// --- Picture fields map ---
var pictureFieldMap = map[string]PictureType{
	"banner":  PicBanner,
	"photo":   PicPhoto,
	"avatar":  PicPhoto,
	"seating": PicSeating,
}

// --- File upload wrapper ---
func handleFileUpload(
	form *multipart.Form,
	field string,
	entity EntityType,
	picType PictureType,
) (string, error) {

	return SaveFormFile(
		form,
		field,
		entity,
		picType,
		true,
	)
}
