package media

import (
	"encoding/json"
	"os/exec"
	"strconv"
	"strings"

	"github.com/rs/zerolog"
)

type Metadata struct {
	Duration      int64  // seconds
	Width         int
	Height        int
	VideoCodec    string
	AudioCodec    string
	AudioChannels int // number of audio channels (2 = stereo, 6 = 5.1, etc.)
	Bitrate       int64
}

type MetadataExtractor struct {
	ffprobePath string
	logger      zerolog.Logger
}

func NewMetadataExtractor(logger zerolog.Logger) *MetadataExtractor {
	// Try to find ffprobe in PATH
	ffprobePath := "ffprobe"
	if path, err := exec.LookPath("ffprobe"); err == nil {
		ffprobePath = path
	}

	return &MetadataExtractor{
		ffprobePath: ffprobePath,
		logger:      logger,
	}
}

func (m *MetadataExtractor) IsAvailable() bool {
	_, err := exec.LookPath(m.ffprobePath)
	return err == nil
}

func (m *MetadataExtractor) Extract(filePath string) (*Metadata, error) {
	args := []string{
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		filePath,
	}

	cmd := exec.Command(m.ffprobePath, args...)
	output, err := cmd.Output()
	if err != nil {
		m.logger.Debug().Err(err).Str("file", filePath).Msg("ffprobe failed")
		return nil, err
	}

	return m.parseOutput(output)
}

type ffprobeOutput struct {
	Streams []ffprobeStream `json:"streams"`
	Format  ffprobeFormat   `json:"format"`
}

type ffprobeStream struct {
	CodecType string `json:"codec_type"`
	CodecName string `json:"codec_name"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	Channels  int    `json:"channels"`
}

type ffprobeFormat struct {
	Duration string `json:"duration"`
	BitRate  string `json:"bit_rate"`
}

func (m *MetadataExtractor) parseOutput(output []byte) (*Metadata, error) {
	var probe ffprobeOutput
	if err := json.Unmarshal(output, &probe); err != nil {
		return nil, err
	}

	meta := &Metadata{}

	// Parse duration
	if probe.Format.Duration != "" {
		if dur, err := strconv.ParseFloat(probe.Format.Duration, 64); err == nil {
			meta.Duration = int64(dur)
		}
	}

	// Parse bitrate
	if probe.Format.BitRate != "" {
		if br, err := strconv.ParseInt(probe.Format.BitRate, 10, 64); err == nil {
			meta.Bitrate = br
		}
	}

	// Parse streams
	for _, stream := range probe.Streams {
		switch stream.CodecType {
		case "video":
			if meta.VideoCodec == "" {
				meta.VideoCodec = strings.ToUpper(stream.CodecName)
				meta.Width = stream.Width
				meta.Height = stream.Height
			}
		case "audio":
			if meta.AudioCodec == "" {
				meta.AudioCodec = strings.ToUpper(stream.CodecName)
				meta.AudioChannels = stream.Channels
			}
		}
	}

	return meta, nil
}
