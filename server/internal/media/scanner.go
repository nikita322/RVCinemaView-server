package media

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"rvcinemaview/internal/storage"
)

type Scanner struct {
	storage  *storage.SQLiteStorage
	logger   zerolog.Logger
	scanning bool
	mu       sync.Mutex
}

func NewScanner(store *storage.SQLiteStorage, logger zerolog.Logger) *Scanner {
	return &Scanner{
		storage: store,
		logger:  logger,
	}
}

func (s *Scanner) IsScanning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.scanning
}

// ScanPath scans a single library path with the given display name
func (s *Scanner) ScanPath(libraryPath, libraryName string) error {
	s.mu.Lock()
	if s.scanning {
		s.mu.Unlock()
		return nil
	}
	s.scanning = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.scanning = false
		s.mu.Unlock()
	}()

	if libraryPath == "" {
		s.logger.Warn().Msg("no library path configured")
		return nil
	}

	info, err := os.Stat(libraryPath)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return nil
	}

	libraryPath = filepath.Clean(libraryPath)
	s.logger.Info().
		Str("path", libraryPath).
		Str("name", libraryName).
		Msg("scanning library")

	// Cleanup deleted files first
	if err := s.CleanupDeletedFiles(); err != nil {
		s.logger.Warn().Err(err).Msg("cleanup failed, continuing with scan")
	}

	// Scan the library directory directly - subfolders become root folders
	return s.scanLibraryRoot(libraryPath, libraryName)
}

// scanLibraryRoot scans the root library directory
// Subfolders of the library become "root" folders (parent_id = NULL)
// Media files in the root have empty folder_id and are returned at root level
func (s *Scanner) scanLibraryRoot(libraryPath, libraryName string) error {
	entries, err := os.ReadDir(libraryPath)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		fullPath := filepath.Join(libraryPath, entry.Name())

		if entry.IsDir() {
			// Skip hidden directories
			if strings.HasPrefix(entry.Name(), ".") {
				continue
			}

			// Create folder as root folder (parent_id = NULL)
			folderID := generateID(fullPath)
			folder := &storage.Folder{
				ID:        folderID,
				Name:      entry.Name(),
				Path:      fullPath,
				ParentID:  nil, // Root level folder
				CreatedAt: time.Now(),
			}

			if err := s.storage.CreateFolder(folder); err != nil {
				s.logger.Error().Err(err).Str("path", fullPath).Msg("failed to create folder")
				continue
			}

			// Recursively scan subfolder
			if err := s.scanDirectory(fullPath, folderID); err != nil {
				s.logger.Error().Err(err).Str("path", fullPath).Msg("failed to scan subfolder")
			}

			continue
		}

		// Check if it's a supported video file in the library root
		if !IsSupportedVideo(entry.Name()) {
			continue
		}

		// Get file info
		info, err := entry.Info()
		if err != nil {
			s.logger.Error().Err(err).Str("path", fullPath).Msg("failed to get file info")
			continue
		}

		// Create media item with empty folder_id (root-level media)
		mediaID := generateID(fullPath)
		title := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))

		mediaItem := &storage.MediaItem{
			ID:         mediaID,
			FolderID:   "", // Empty = root level
			Title:      title,
			Path:       fullPath,
			Size:       info.Size(),
			ModifiedAt: info.ModTime(),
			CreatedAt:  time.Now(),
		}

		if err := s.storage.CreateMediaItem(mediaItem); err != nil {
			s.logger.Error().Err(err).Str("path", fullPath).Msg("failed to create media item")
			continue
		}

		s.logger.Debug().Str("title", title).Int64("size", info.Size()).Msg("added root media item")
	}

	return nil
}

func (s *Scanner) scanDirectory(dirPath string, parentID string) error {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return err
	}

	var mediaCount int

	for _, entry := range entries {
		fullPath := filepath.Join(dirPath, entry.Name())

		if entry.IsDir() {
			// Skip hidden directories
			if strings.HasPrefix(entry.Name(), ".") {
				continue
			}

			// Create subfolder
			folderID := generateID(fullPath)
			folder := &storage.Folder{
				ID:        folderID,
				Name:      entry.Name(),
				Path:      fullPath,
				ParentID:  &parentID,
				CreatedAt: time.Now(),
			}

			if err := s.storage.CreateFolder(folder); err != nil {
				s.logger.Error().Err(err).Str("path", fullPath).Msg("failed to create folder")
				continue
			}

			// Recursively scan subfolder
			if err := s.scanDirectory(fullPath, folderID); err != nil {
				s.logger.Error().Err(err).Str("path", fullPath).Msg("failed to scan subfolder")
			}

			continue
		}

		// Check if it's a supported video file
		if !IsSupportedVideo(entry.Name()) {
			continue
		}

		// Get file info
		info, err := entry.Info()
		if err != nil {
			s.logger.Error().Err(err).Str("path", fullPath).Msg("failed to get file info")
			continue
		}

		// Create media item
		mediaID := generateID(fullPath)
		title := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))

		mediaItem := &storage.MediaItem{
			ID:         mediaID,
			FolderID:   parentID,
			Title:      title,
			Path:       fullPath,
			Size:       info.Size(),
			ModifiedAt: info.ModTime(),
			CreatedAt:  time.Now(),
		}

		if err := s.storage.CreateMediaItem(mediaItem); err != nil {
			s.logger.Error().Err(err).Str("path", fullPath).Msg("failed to create media item")
			continue
		}

		mediaCount++
		s.logger.Debug().
			Str("title", title).
			Int64("size", info.Size()).
			Msg("added media item")
	}

	// Update folder item count
	if mediaCount > 0 {
		if err := s.storage.UpdateFolderItemCount(parentID, mediaCount); err != nil {
			s.logger.Error().Err(err).Msg("failed to update folder item count")
		}
	}

	return nil
}

func generateID(path string) string {
	hash := sha256.Sum256([]byte(path))
	return hex.EncodeToString(hash[:8])
}

// CleanupDeletedFiles removes database entries for files that no longer exist
func (s *Scanner) CleanupDeletedFiles() error {
	// Cleanup media items
	mediaPaths, err := s.storage.GetAllMediaPaths()
	if err != nil {
		return err
	}

	deletedMedia := 0
	for id, path := range mediaPaths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			if err := s.storage.DeleteMediaItem(id); err != nil {
				s.logger.Error().Err(err).Str("path", path).Msg("failed to delete media item")
			} else {
				deletedMedia++
				s.logger.Debug().Str("path", path).Msg("deleted missing media item")
			}
		}
	}

	// Cleanup folders
	folderPaths, err := s.storage.GetAllFolderPaths()
	if err != nil {
		return err
	}

	deletedFolders := 0
	for id, path := range folderPaths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			if err := s.storage.DeleteFolder(id); err != nil {
				s.logger.Error().Err(err).Str("path", path).Msg("failed to delete folder")
			} else {
				deletedFolders++
				s.logger.Debug().Str("path", path).Msg("deleted missing folder")
			}
		}
	}

	if deletedMedia > 0 || deletedFolders > 0 {
		s.logger.Info().
			Int("media", deletedMedia).
			Int("folders", deletedFolders).
			Msg("cleanup completed")
	}

	return nil
}
