package media

import (
	"path/filepath"
	"strings"
)

var supportedVideoExtensions = map[string]bool{
	".mp4":  true,
	".m4v":  true,
	".mkv":  true,
	".avi":  true,
	".webm": true,
	".mov":  true,
	".wmv":  true,
	".flv":  true,
}

func IsSupportedVideo(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return supportedVideoExtensions[ext]
}

func GetContentType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))

	switch ext {
	case ".mp4", ".m4v":
		return "video/mp4"
	case ".mkv":
		return "video/x-matroska"
	case ".avi":
		return "video/x-msvideo"
	case ".webm":
		return "video/webm"
	case ".mov":
		return "video/quicktime"
	case ".wmv":
		return "video/x-ms-wmv"
	case ".flv":
		return "video/x-flv"
	default:
		return "application/octet-stream"
	}
}
