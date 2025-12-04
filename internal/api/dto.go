package api

import "rvcinemaview/internal/storage"

type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

type MediaResponse struct {
	Media     *storage.MediaItem `json:"media"`
	StreamURL string             `json:"stream_url"`
}

type ScanResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Playback DTOs

type SavePlaybackRequest struct {
	Position int64 `json:"position"` // Seconds
	Duration int64 `json:"duration"` // Seconds
}

type PlaybackResponse struct {
	MediaID  string  `json:"media_id"`
	Position int64   `json:"position"`
	Duration int64   `json:"duration"`
	Progress float64 `json:"progress"`
}

type ContinueWatchingResponse struct {
	Items []storage.ContinueWatchingItem `json:"items"`
}

// Library tree - complete structure in one response

type LibraryTreeResponse struct {
	Name    string              `json:"name"`
	Folders []FolderNode        `json:"folders"`
	Media   []storage.MediaItem `json:"media,omitempty"`
}

type FolderNode struct {
	ID         string              `json:"id"`
	Name       string              `json:"name"`
	SubFolders []FolderNode        `json:"sub_folders,omitempty"`
	Media      []storage.MediaItem `json:"media,omitempty"`
}
