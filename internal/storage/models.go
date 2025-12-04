package storage

import "time"

type Folder struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Path      string    `json:"-"`
	ParentID  *string   `json:"-"` // Internal use only
	ItemCount int       `json:"-"` // Internal use only
	CreatedAt time.Time `json:"-"`
}

type MediaItem struct {
	ID            string    `json:"id"`
	FolderID      string    `json:"-"` // Internal use only
	Title         string    `json:"title"`
	Path          string    `json:"-"`
	Size          int64     `json:"size"`
	Duration      *int64    `json:"duration,omitempty"`
	Width         *int      `json:"width,omitempty"`
	Height        *int      `json:"height,omitempty"`
	VideoCodec    *string   `json:"video_codec,omitempty"`
	AudioCodec    *string   `json:"audio_codec,omitempty"`
	AudioChannels *int      `json:"audio_channels,omitempty"` // 2 = stereo, 6 = 5.1, 8 = 7.1
	HasSubtitles  bool      `json:"-"`                        // Internal use only
	ModifiedAt    time.Time `json:"-"`
	CreatedAt     time.Time `json:"-"`
}

type PlaybackState struct {
	MediaID   string    `json:"media_id"`
	Position  int64     `json:"position"` // Seconds
	Duration  int64     `json:"duration"` // Seconds
	Progress  float64   `json:"progress"` // 0.0 - 1.0
	UpdatedAt time.Time `json:"-"`
}

// ContinueWatchingItem combines media info with playback state
type ContinueWatchingItem struct {
	Media         MediaItem     `json:"media"`
	PlaybackState PlaybackState `json:"playback_state"`
}
