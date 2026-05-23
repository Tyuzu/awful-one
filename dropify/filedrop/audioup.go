// dropify/filedrop/audioup.go

package filedrop

import "naevis/dropify/filemgr"

// -------------------- Audio Processing --------------------

// ProcessAudio handles audio file processing (exported for use by services)
func ProcessAudio(savedPath, uploadDir, uniqueID string, entitytype filemgr.EntityType) ([]int, []string) {
	return processAudio(savedPath, uploadDir, uniqueID, entitytype)
}

func processAudio(savedPath, uploadDir, uniqueID string, entitytype filemgr.EntityType) ([]int, []string) {
	_ = entitytype
	resolutions, outputPath := processAudioResolutions(savedPath, uploadDir, uniqueID)
	var paths []string
	if outputPath != "" {
		paths = []string{normalizePath(outputPath)}
	}

	go createSubtitleFile(uniqueID)
	NotifyEvent("postaudio-uploaded", map[string]string{"id": uniqueID})

	return resolutions, paths
}
