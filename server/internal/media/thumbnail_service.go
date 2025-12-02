package media

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"rvcinemaview/internal/cache"
	"rvcinemaview/internal/storage"
)

// ThumbnailService manages thumbnail generation and caching
type ThumbnailService struct {
	generator   *ThumbnailGenerator
	metadata    *MetadataExtractor
	storage     *storage.SQLiteStorage
	cache       *cache.LRUCache
	logger      zerolog.Logger
	processing  map[string]bool
	processingMu sync.Mutex
}

// NewThumbnailService creates a new thumbnail service
func NewThumbnailService(
	generator *ThumbnailGenerator,
	metadata *MetadataExtractor,
	store *storage.SQLiteStorage,
	cacheCapacity int,
	cacheMaxSize int64,
	logger zerolog.Logger,
) *ThumbnailService {
	return &ThumbnailService{
		generator:  generator,
		metadata:   metadata,
		storage:    store,
		cache:      cache.NewLRUCache(cacheCapacity, cacheMaxSize),
		logger:     logger,
		processing: make(map[string]bool),
	}
}

// GetThumbnail returns thumbnail data from cache or generates it
func (s *ThumbnailService) GetThumbnail(mediaID string) ([]byte, error) {
	// Check cache first
	if data, ok := s.cache.Get(mediaID); ok {
		s.logger.Debug().Str("id", mediaID).Msg("thumbnail from cache")
		return data, nil
	}

	// Check if file exists on disk
	thumbnailPath := s.generator.GetPath(mediaID)
	if data, err := os.ReadFile(thumbnailPath); err == nil {
		s.logger.Debug().Str("id", mediaID).Str("path", thumbnailPath).Msg("thumbnail from disk")
		s.cache.Set(mediaID, data)
		return data, nil
	}

	// Get media item to generate thumbnail
	media, err := s.storage.GetMediaItem(mediaID)
	if err != nil {
		s.logger.Error().Err(err).Str("id", mediaID).Msg("failed to get media item")
		return nil, err
	}
	if media == nil {
		s.logger.Warn().Str("id", mediaID).Msg("media item not found")
		return nil, nil
	}

	s.logger.Info().Str("id", mediaID).Str("path", media.Path).Msg("generating thumbnail on demand")

	// Check if generator is available
	if !s.generator.IsAvailable() {
		s.logger.Warn().Msg("ffmpeg not available for thumbnail generation")
		return nil, fmt.Errorf("ffmpeg not available")
	}

	// Generate thumbnail synchronously if not exists
	duration := int64(0)
	if media.Duration != nil {
		duration = *media.Duration
	}

	thumbnailPath, err = s.generator.Generate(media.Path, mediaID, duration)
	if err != nil {
		s.logger.Error().Err(err).Str("id", mediaID).Str("video", media.Path).Msg("failed to generate thumbnail")
		return nil, err
	}

	// Read and cache
	data, err := os.ReadFile(thumbnailPath)
	if err != nil {
		s.logger.Error().Err(err).Str("thumbnail", thumbnailPath).Msg("failed to read generated thumbnail")
		return nil, err
	}

	s.cache.Set(mediaID, data)
	s.logger.Info().Str("id", mediaID).Int("size", len(data)).Msg("thumbnail generated and cached")
	return data, nil
}

// HasThumbnail checks if thumbnail exists
func (s *ThumbnailService) HasThumbnail(mediaID string) bool {
	if _, ok := s.cache.Get(mediaID); ok {
		return true
	}
	return s.generator.Exists(mediaID)
}

// ProcessMediaItem extracts metadata and generates thumbnail for a media item
func (s *ThumbnailService) ProcessMediaItem(ctx context.Context, media *storage.MediaItem) error {
	s.processingMu.Lock()
	if s.processing[media.ID] {
		s.processingMu.Unlock()
		return nil
	}
	s.processing[media.ID] = true
	s.processingMu.Unlock()

	defer func() {
		s.processingMu.Lock()
		delete(s.processing, media.ID)
		s.processingMu.Unlock()
	}()

	// Extract metadata if available
	if s.metadata.IsAvailable() && media.Duration == nil {
		meta, err := s.metadata.Extract(media.Path)
		if err == nil && meta != nil {
			// Update storage with metadata
			if err := s.storage.UpdateMediaMetadata(
				media.ID,
				meta.Duration,
				meta.Width,
				meta.Height,
				meta.VideoCodec,
				meta.AudioCodec,
			); err != nil {
				s.logger.Error().Err(err).Str("id", media.ID).Msg("failed to update metadata")
			} else {
				s.logger.Debug().
					Str("id", media.ID).
					Int64("duration", meta.Duration).
					Int("width", meta.Width).
					Int("height", meta.Height).
					Msg("metadata extracted")
			}
			media.Duration = &meta.Duration
		}
	}

	// Generate thumbnail if ffmpeg available
	if s.generator.IsAvailable() && !s.generator.Exists(media.ID) {
		duration := int64(0)
		if media.Duration != nil {
			duration = *media.Duration
		}

		if _, err := s.generator.Generate(media.Path, media.ID, duration); err != nil {
			s.logger.Debug().Err(err).Str("id", media.ID).Msg("failed to generate thumbnail")
		}
	}

	return nil
}

// StartBackgroundProcessing processes all media items in background
func (s *ThumbnailService) StartBackgroundProcessing(ctx context.Context, batchSize int, delay time.Duration) {
	go func() {
		s.logger.Info().Msg("starting background thumbnail/metadata processing")

		totalProcessed := 0
		for {
			select {
			case <-ctx.Done():
				s.logger.Info().Int("processed", totalProcessed).Msg("background processing cancelled")
				return
			default:
			}

			items, err := s.storage.GetMediaItemsWithoutMetadata(batchSize)
			if err != nil {
				s.logger.Error().Err(err).Msg("failed to get items without metadata")
				return
			}

			if len(items) == 0 {
				break // No more items to process
			}

			for _, item := range items {
				select {
				case <-ctx.Done():
					s.logger.Info().Int("processed", totalProcessed).Msg("background processing cancelled")
					return
				default:
					itemCopy := item
					if err := s.ProcessMediaItem(ctx, &itemCopy); err != nil {
						s.logger.Error().Err(err).Str("id", item.ID).Msg("failed to process item")
					}
					totalProcessed++
					time.Sleep(delay) // Rate limit to avoid overloading weak CPU
				}
			}
		}

		s.logger.Info().Int("processed", totalProcessed).Msg("background processing completed")
	}()
}

// CacheStats returns cache statistics
func (s *ThumbnailService) CacheStats() (count int, size int64) {
	return s.cache.Len(), s.cache.Size()
}
