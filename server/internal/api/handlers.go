package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
	"rvcinemaview/internal/media"
	"rvcinemaview/internal/storage"
	"rvcinemaview/internal/streaming"
)

const Version = "0.1.0"

type Handler struct {
	storage          *storage.SQLiteStorage
	logger           zerolog.Logger
	scanner          ScannerInterface
	streamer         *streaming.Handler
	thumbnailService *media.ThumbnailService
	libraryPath      string
	libraryName      string
}

type ScannerInterface interface {
	ScanPath(path, name string) error
	IsScanning() bool
}

func NewHandler(store *storage.SQLiteStorage, logger zerolog.Logger, libraryPath, libraryName string) *Handler {
	return &Handler{
		storage:     store,
		logger:      logger,
		streamer:    streaming.NewHandler(),
		libraryPath: libraryPath,
		libraryName: libraryName,
	}
}

func (h *Handler) SetThumbnailService(service *media.ThumbnailService) {
	h.thumbnailService = service
}

func (h *Handler) SetScanner(scanner ScannerInterface) {
	h.scanner = scanner
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	resp := HealthResponse{
		Status:  "ok",
		Version: Version,
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) ScanLibrary(w http.ResponseWriter, r *http.Request) {
	if h.scanner == nil {
		writeError(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "Scanner not initialized")
		return
	}

	if h.scanner.IsScanning() {
		writeJSON(w, http.StatusOK, ScanResponse{
			Status:  "in_progress",
			Message: "Scan already in progress",
		})
		return
	}

	if h.libraryPath == "" {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "No library path configured")
		return
	}

	go func() {
		if err := h.scanner.ScanPath(h.libraryPath, h.libraryName); err != nil {
			h.logger.Error().Err(err).Msg("scan failed")
		}
	}()

	writeJSON(w, http.StatusAccepted, ScanResponse{
		Status:  "started",
		Message: "Library scan started",
	})
}

func (h *Handler) GetMedia(w http.ResponseWriter, r *http.Request) {
	mediaID := chi.URLParam(r, "id")

	media, err := h.storage.GetMediaItem(mediaID)
	if err != nil {
		h.logger.Error().Err(err).Str("id", mediaID).Msg("failed to get media")
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get media")
		return
	}

	if media == nil {
		writeError(w, http.StatusNotFound, "MEDIA_NOT_FOUND", "Media not found")
		return
	}

	writeJSON(w, http.StatusOK, MediaResponse{
		Media:     media,
		StreamURL: "/api/v1/media/" + mediaID + "/stream",
	})
}

func (h *Handler) StreamMedia(w http.ResponseWriter, r *http.Request) {
	mediaID := chi.URLParam(r, "id")

	media, err := h.storage.GetMediaItem(mediaID)
	if err != nil {
		h.logger.Error().Err(err).Str("id", mediaID).Msg("failed to get media for streaming")
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get media")
		return
	}

	if media == nil {
		writeError(w, http.StatusNotFound, "MEDIA_NOT_FOUND", "Media not found")
		return
	}

	h.streamer.ServeFile(w, r, media.Path)
}

func (h *Handler) GetThumbnail(w http.ResponseWriter, r *http.Request) {
	mediaID := chi.URLParam(r, "id")

	h.logger.Info().Str("id", mediaID).Msg("thumbnail requested")

	if h.thumbnailService == nil {
		h.logger.Warn().Msg("thumbnail service is nil")
		writeError(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "Thumbnail service not available")
		return
	}

	data, err := h.thumbnailService.GetThumbnail(mediaID)
	if err != nil {
		h.logger.Warn().Err(err).Str("id", mediaID).Msg("failed to get thumbnail")
		writeError(w, http.StatusNotFound, "THUMBNAIL_NOT_FOUND", "Thumbnail not available")
		return
	}

	h.logger.Info().Str("id", mediaID).Int("size", len(data)).Msg("thumbnail served")

	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Cache-Control", "public, max-age=86400") // Cache for 24 hours
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, ErrorResponse{
		Error: ErrorDetail{
			Code:    code,
			Message: message,
		},
	})
}

// Playback handlers

func (h *Handler) SavePlaybackPosition(w http.ResponseWriter, r *http.Request) {
	mediaID := chi.URLParam(r, "id")

	// Check if media exists
	media, err := h.storage.GetMediaItem(mediaID)
	if err != nil {
		h.logger.Error().Err(err).Str("id", mediaID).Msg("failed to get media for playback")
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get media")
		return
	}

	if media == nil {
		writeError(w, http.StatusNotFound, "MEDIA_NOT_FOUND", "Media not found")
		return
	}

	var req SavePlaybackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid request body")
		return
	}

	// Validate
	if req.Duration <= 0 {
		writeError(w, http.StatusBadRequest, "BAD_REQUEST", "Duration must be positive")
		return
	}

	if req.Position < 0 {
		req.Position = 0
	}

	if req.Position > req.Duration {
		req.Position = req.Duration
	}

	// Calculate progress
	progress := float64(req.Position) / float64(req.Duration)

	state := &storage.PlaybackState{
		MediaID:  mediaID,
		Position: req.Position,
		Duration: req.Duration,
		Progress: progress,
	}

	if err := h.storage.SavePlaybackState(state); err != nil {
		h.logger.Error().Err(err).Str("id", mediaID).Msg("failed to save playback state")
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to save position")
		return
	}

	h.logger.Debug().
		Str("media_id", mediaID).
		Int64("position", req.Position).
		Float64("progress", progress).
		Msg("playback position saved")

	writeJSON(w, http.StatusOK, PlaybackResponse{
		MediaID:  mediaID,
		Position: req.Position,
		Duration: req.Duration,
		Progress: progress,
	})
}

func (h *Handler) GetPlaybackPosition(w http.ResponseWriter, r *http.Request) {
	mediaID := chi.URLParam(r, "id")

	state, err := h.storage.GetPlaybackState(mediaID)
	if err != nil {
		h.logger.Error().Err(err).Str("id", mediaID).Msg("failed to get playback state")
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get position")
		return
	}

	if state == nil {
		// No saved position, return zeros
		writeJSON(w, http.StatusOK, PlaybackResponse{
			MediaID:  mediaID,
			Position: 0,
			Duration: 0,
			Progress: 0,
		})
		return
	}

	writeJSON(w, http.StatusOK, PlaybackResponse{
		MediaID:  state.MediaID,
		Position: state.Position,
		Duration: state.Duration,
		Progress: state.Progress,
	})
}

func (h *Handler) GetContinueWatching(w http.ResponseWriter, r *http.Request) {
	items, err := h.storage.GetContinueWatching(20) // Limit to 20 items
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to get continue watching")
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get continue watching")
		return
	}

	if items == nil {
		items = []storage.ContinueWatchingItem{}
	}

	writeJSON(w, http.StatusOK, ContinueWatchingResponse{
		Items: items,
	})
}

// GetLibraryTree returns the complete library structure in one response
func (h *Handler) GetLibraryTree(w http.ResponseWriter, r *http.Request) {
	// Get all root folders
	rootFolders, err := h.storage.GetRootFolders()
	if err != nil {
		h.logger.Error().Err(err).Msg("failed to get root folders")
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Failed to get library")
		return
	}

	// Get root-level media (media in the library root directory)
	rootMedia, err := h.storage.GetRootMedia()
	if err != nil {
		h.logger.Warn().Err(err).Msg("failed to get root media")
		rootMedia = []storage.MediaItem{}
	}

	// Build tree recursively
	var folderNodes []FolderNode
	for _, folder := range rootFolders {
		node := h.buildFolderNode(folder)
		folderNodes = append(folderNodes, node)
	}

	// If there's exactly one root folder and no root media,
	// return the contents of that folder directly (unwrap it)
	// This provides a better UX - user sees content immediately
	if len(folderNodes) == 1 && len(rootMedia) == 0 {
		singleFolder := folderNodes[0]
		writeJSON(w, http.StatusOK, LibraryTreeResponse{
			Name:    h.libraryName,
			Folders: singleFolder.SubFolders,
			Media:   singleFolder.Media,
		})
		return
	}

	if folderNodes == nil {
		folderNodes = []FolderNode{}
	}

	writeJSON(w, http.StatusOK, LibraryTreeResponse{
		Name:    h.libraryName,
		Folders: folderNodes,
		Media:   rootMedia,
	})
}

func (h *Handler) buildFolderNode(folder storage.Folder) FolderNode {
	node := FolderNode{
		ID:   folder.ID,
		Name: folder.Name,
	}

	// Get subfolders
	subFolders, err := h.storage.GetSubFolders(folder.ID)
	if err == nil && len(subFolders) > 0 {
		for _, sub := range subFolders {
			subNode := h.buildFolderNode(sub)
			node.SubFolders = append(node.SubFolders, subNode)
		}
	}

	// Get media items
	mediaItems, err := h.storage.GetMediaItemsByFolder(folder.ID)
	if err == nil && len(mediaItems) > 0 {
		node.Media = mediaItems
	}

	return node
}
