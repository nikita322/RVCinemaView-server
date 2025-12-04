package media

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/rs/zerolog"
)

type ThumbnailGenerator struct {
	ffmpegPath string
	outputDir  string
	logger     zerolog.Logger
}

func NewThumbnailGenerator(outputDir string, logger zerolog.Logger) *ThumbnailGenerator {
	// Try to find ffmpeg in PATH
	ffmpegPath := "ffmpeg"
	if path, err := exec.LookPath("ffmpeg"); err == nil {
		ffmpegPath = path
	}

	// Ensure output directory exists
	os.MkdirAll(outputDir, 0755)

	return &ThumbnailGenerator{
		ffmpegPath: ffmpegPath,
		outputDir:  outputDir,
		logger:     logger,
	}
}

func (t *ThumbnailGenerator) IsAvailable() bool {
	_, err := exec.LookPath(t.ffmpegPath)
	return err == nil
}

func (t *ThumbnailGenerator) GetOutputDir() string {
	return t.outputDir
}

// Generate creates a thumbnail for the video file
// Returns the path to the generated thumbnail
func (t *ThumbnailGenerator) Generate(videoPath string, mediaID string, duration int64) (string, error) {
	outputPath := filepath.Join(t.outputDir, mediaID+".jpg")

	// Check if thumbnail already exists
	if _, err := os.Stat(outputPath); err == nil {
		return outputPath, nil
	}

	// Calculate timestamp for thumbnail (10% into video, or 5 seconds, whichever is smaller)
	timestamp := int64(5)
	if duration > 0 {
		tenPercent := duration / 10
		if tenPercent > 0 && tenPercent < timestamp {
			timestamp = tenPercent
		}
		if timestamp > duration {
			timestamp = duration / 2
		}
	}

	// ffmpeg arguments for thumbnail generation
	// -ss: seek to timestamp
	// -i: input file
	// -vframes 1: extract one frame
	// -vf scale: resize maintaining aspect ratio (max 320px width)
	// -q:v 2: quality (2 = high quality JPEG)
	args := []string{
		"-ss", fmt.Sprintf("%d", timestamp),
		"-i", videoPath,
		"-vframes", "1",
		"-vf", "scale=320:-1",
		"-q:v", "2",
		"-y", // overwrite output
		outputPath,
	}

	cmd := exec.Command(t.ffmpegPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.logger.Debug().
			Err(err).
			Str("video", videoPath).
			Str("output", string(output)).
			Msg("ffmpeg thumbnail generation failed")
		return "", fmt.Errorf("ffmpeg failed: %w", err)
	}

	// Verify thumbnail was created
	if _, err := os.Stat(outputPath); err != nil {
		return "", fmt.Errorf("thumbnail file not created")
	}

	t.logger.Debug().
		Str("video", videoPath).
		Str("thumbnail", outputPath).
		Msg("thumbnail generated")

	return outputPath, nil
}

// Delete removes a thumbnail file
func (t *ThumbnailGenerator) Delete(mediaID string) error {
	outputPath := filepath.Join(t.outputDir, mediaID+".jpg")
	return os.Remove(outputPath)
}

// Exists checks if thumbnail exists for the given media ID
func (t *ThumbnailGenerator) Exists(mediaID string) bool {
	outputPath := filepath.Join(t.outputDir, mediaID+".jpg")
	_, err := os.Stat(outputPath)
	return err == nil
}

// GetPath returns the thumbnail path for a media ID
func (t *ThumbnailGenerator) GetPath(mediaID string) string {
	return filepath.Join(t.outputDir, mediaID+".jpg")
}
