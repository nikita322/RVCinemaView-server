package storage

import (
	"database/sql"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type SQLiteStorage struct {
	db *sql.DB
}

func NewSQLiteStorage(dbPath string) (*SQLiteStorage, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	s := &SQLiteStorage{db: db}

	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}

	return s, nil
}

func (s *SQLiteStorage) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS folders (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		path TEXT NOT NULL UNIQUE,
		parent_id TEXT REFERENCES folders(id),
		item_count INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS media_items (
		id TEXT PRIMARY KEY,
		folder_id TEXT DEFAULT '' REFERENCES folders(id),
		title TEXT NOT NULL,
		path TEXT NOT NULL UNIQUE,
		size INTEGER NOT NULL,
		duration INTEGER,
		width INTEGER,
		height INTEGER,
		video_codec TEXT,
		audio_codec TEXT,
		has_subtitles BOOLEAN DEFAULT FALSE,
		thumbnail_generated BOOLEAN DEFAULT FALSE,
		file_modified_at DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_media_folder ON media_items(folder_id);
	CREATE INDEX IF NOT EXISTS idx_media_title ON media_items(title);
	CREATE INDEX IF NOT EXISTS idx_folders_parent ON folders(parent_id);

	CREATE TABLE IF NOT EXISTS playback_states (
		media_id TEXT PRIMARY KEY REFERENCES media_items(id) ON DELETE CASCADE,
		position INTEGER NOT NULL,
		duration INTEGER NOT NULL,
		progress REAL NOT NULL,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_playback_updated ON playback_states(updated_at DESC);
	`

	_, err := s.db.Exec(schema)
	return err
}

func (s *SQLiteStorage) Close() error {
	return s.db.Close()
}

// Folders
func (s *SQLiteStorage) GetRootFolders() ([]Folder, error) {
	rows, err := s.db.Query(`
		SELECT id, name, path, parent_id, item_count, created_at
		FROM folders WHERE parent_id IS NULL ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var folders []Folder
	for rows.Next() {
		var f Folder
		if err := rows.Scan(&f.ID, &f.Name, &f.Path, &f.ParentID, &f.ItemCount, &f.CreatedAt); err != nil {
			return nil, err
		}
		folders = append(folders, f)
	}

	return folders, rows.Err()
}

func (s *SQLiteStorage) GetSubFolders(parentID string) ([]Folder, error) {
	rows, err := s.db.Query(`
		SELECT id, name, path, parent_id, item_count, created_at
		FROM folders WHERE parent_id = ? ORDER BY name
	`, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var folders []Folder
	for rows.Next() {
		var f Folder
		if err := rows.Scan(&f.ID, &f.Name, &f.Path, &f.ParentID, &f.ItemCount, &f.CreatedAt); err != nil {
			return nil, err
		}
		folders = append(folders, f)
	}

	return folders, rows.Err()
}

func (s *SQLiteStorage) CreateFolder(f *Folder) error {
	_, err := s.db.Exec(`
		INSERT INTO folders (id, name, path, parent_id, item_count, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(path) DO UPDATE SET name = excluded.name
	`, f.ID, f.Name, f.Path, f.ParentID, f.ItemCount, f.CreatedAt)

	return err
}

func (s *SQLiteStorage) UpdateFolderItemCount(id string, count int) error {
	_, err := s.db.Exec("UPDATE folders SET item_count = ? WHERE id = ?", count, id)
	return err
}

// Media Items
func (s *SQLiteStorage) GetMediaItem(id string) (*MediaItem, error) {
	row := s.db.QueryRow(`
		SELECT id, folder_id, title, path, size, duration, width, height,
		       video_codec, audio_codec, has_subtitles, file_modified_at, created_at
		FROM media_items WHERE id = ?
	`, id)

	var m MediaItem
	var modifiedAt sql.NullTime
	err := row.Scan(
		&m.ID, &m.FolderID, &m.Title, &m.Path, &m.Size,
		&m.Duration, &m.Width, &m.Height,
		&m.VideoCodec, &m.AudioCodec, &m.HasSubtitles,
		&modifiedAt, &m.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if modifiedAt.Valid {
		m.ModifiedAt = modifiedAt.Time
	}

	return &m, nil
}

func (s *SQLiteStorage) GetMediaItemByPath(path string) (*MediaItem, error) {
	row := s.db.QueryRow(`
		SELECT id, folder_id, title, path, size, duration, width, height,
		       video_codec, audio_codec, has_subtitles, file_modified_at, created_at
		FROM media_items WHERE path = ?
	`, path)

	var m MediaItem
	var modifiedAt sql.NullTime
	err := row.Scan(
		&m.ID, &m.FolderID, &m.Title, &m.Path, &m.Size,
		&m.Duration, &m.Width, &m.Height,
		&m.VideoCodec, &m.AudioCodec, &m.HasSubtitles,
		&modifiedAt, &m.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if modifiedAt.Valid {
		m.ModifiedAt = modifiedAt.Time
	}

	return &m, nil
}

// GetRootMedia returns media items that are in the library root (folder_id is empty)
func (s *SQLiteStorage) GetRootMedia() ([]MediaItem, error) {
	rows, err := s.db.Query(`
		SELECT id, folder_id, title, path, size, duration, width, height,
		       video_codec, audio_codec, has_subtitles, file_modified_at, created_at
		FROM media_items WHERE folder_id = '' ORDER BY title
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []MediaItem
	for rows.Next() {
		var m MediaItem
		var modifiedAt sql.NullTime
		if err := rows.Scan(
			&m.ID, &m.FolderID, &m.Title, &m.Path, &m.Size,
			&m.Duration, &m.Width, &m.Height,
			&m.VideoCodec, &m.AudioCodec, &m.HasSubtitles,
			&modifiedAt, &m.CreatedAt,
		); err != nil {
			return nil, err
		}
		if modifiedAt.Valid {
			m.ModifiedAt = modifiedAt.Time
		}
		items = append(items, m)
	}

	return items, rows.Err()
}

func (s *SQLiteStorage) GetMediaItemsByFolder(folderID string) ([]MediaItem, error) {
	rows, err := s.db.Query(`
		SELECT id, folder_id, title, path, size, duration, width, height,
		       video_codec, audio_codec, has_subtitles, file_modified_at, created_at
		FROM media_items WHERE folder_id = ? ORDER BY title
	`, folderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []MediaItem
	for rows.Next() {
		var m MediaItem
		var modifiedAt sql.NullTime
		if err := rows.Scan(
			&m.ID, &m.FolderID, &m.Title, &m.Path, &m.Size,
			&m.Duration, &m.Width, &m.Height,
			&m.VideoCodec, &m.AudioCodec, &m.HasSubtitles,
			&modifiedAt, &m.CreatedAt,
		); err != nil {
			return nil, err
		}
		if modifiedAt.Valid {
			m.ModifiedAt = modifiedAt.Time
		}
		items = append(items, m)
	}

	return items, rows.Err()
}

func (s *SQLiteStorage) CreateMediaItem(m *MediaItem) error {
	_, err := s.db.Exec(`
		INSERT INTO media_items (
			id, folder_id, title, path, size, duration, width, height,
			video_codec, audio_codec, has_subtitles, file_modified_at, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(path) DO UPDATE SET
			title = excluded.title,
			size = excluded.size,
			file_modified_at = excluded.file_modified_at,
			updated_at = excluded.updated_at
	`,
		m.ID, m.FolderID, m.Title, m.Path, m.Size,
		m.Duration, m.Width, m.Height,
		m.VideoCodec, m.AudioCodec, m.HasSubtitles,
		m.ModifiedAt, m.CreatedAt, time.Now(),
	)

	return err
}

// UpdateMediaMetadata updates metadata fields for a media item
func (s *SQLiteStorage) UpdateMediaMetadata(id string, duration int64, width, height int, videoCodec, audioCodec string) error {
	_, err := s.db.Exec(`
		UPDATE media_items SET
			duration = ?,
			width = ?,
			height = ?,
			video_codec = ?,
			audio_codec = ?,
			updated_at = ?
		WHERE id = ?
	`, duration, width, height, videoCodec, audioCodec, time.Now(), id)
	return err
}

// GetMediaItemsWithoutMetadata returns media items without duration (metadata not extracted)
func (s *SQLiteStorage) GetMediaItemsWithoutMetadata(limit int) ([]MediaItem, error) {
	rows, err := s.db.Query(`
		SELECT id, folder_id, title, path, size, duration, width, height,
		       video_codec, audio_codec, has_subtitles, file_modified_at, created_at
		FROM media_items WHERE duration IS NULL LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []MediaItem
	for rows.Next() {
		var m MediaItem
		var modifiedAt sql.NullTime
		if err := rows.Scan(
			&m.ID, &m.FolderID, &m.Title, &m.Path, &m.Size,
			&m.Duration, &m.Width, &m.Height,
			&m.VideoCodec, &m.AudioCodec, &m.HasSubtitles,
			&modifiedAt, &m.CreatedAt,
		); err != nil {
			return nil, err
		}
		if modifiedAt.Valid {
			m.ModifiedAt = modifiedAt.Time
		}
		items = append(items, m)
	}

	return items, rows.Err()
}

// Playback State methods

// SavePlaybackState saves or updates playback position for a media item
func (s *SQLiteStorage) SavePlaybackState(state *PlaybackState) error {
	_, err := s.db.Exec(`
		INSERT INTO playback_states (media_id, position, duration, progress, updated_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(media_id) DO UPDATE SET
			position = excluded.position,
			duration = excluded.duration,
			progress = excluded.progress,
			updated_at = excluded.updated_at
	`, state.MediaID, state.Position, state.Duration, state.Progress, time.Now())
	return err
}

// GetPlaybackState returns playback state for a media item
func (s *SQLiteStorage) GetPlaybackState(mediaID string) (*PlaybackState, error) {
	row := s.db.QueryRow(`
		SELECT media_id, position, duration, progress, updated_at
		FROM playback_states WHERE media_id = ?
	`, mediaID)

	var state PlaybackState
	err := row.Scan(&state.MediaID, &state.Position, &state.Duration, &state.Progress, &state.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &state, nil
}

// GetContinueWatching returns media items with playback progress (not finished)
// Progress between 5% and 95% is considered "in progress"
func (s *SQLiteStorage) GetContinueWatching(limit int) ([]ContinueWatchingItem, error) {
	rows, err := s.db.Query(`
		SELECT
			m.id, m.folder_id, m.title, m.path, m.size, m.duration, m.width, m.height,
			m.video_codec, m.audio_codec, m.has_subtitles, m.file_modified_at, m.created_at,
			p.media_id, p.position, p.duration, p.progress, p.updated_at
		FROM playback_states p
		JOIN media_items m ON p.media_id = m.id
		WHERE p.progress > 0.02 AND p.progress < 0.95
		ORDER BY p.updated_at DESC
		LIMIT ?
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []ContinueWatchingItem
	for rows.Next() {
		var item ContinueWatchingItem
		var modifiedAt sql.NullTime
		if err := rows.Scan(
			&item.Media.ID, &item.Media.FolderID, &item.Media.Title, &item.Media.Path,
			&item.Media.Size, &item.Media.Duration, &item.Media.Width, &item.Media.Height,
			&item.Media.VideoCodec, &item.Media.AudioCodec, &item.Media.HasSubtitles,
			&modifiedAt, &item.Media.CreatedAt,
			&item.PlaybackState.MediaID, &item.PlaybackState.Position,
			&item.PlaybackState.Duration, &item.PlaybackState.Progress,
			&item.PlaybackState.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if modifiedAt.Valid {
			item.Media.ModifiedAt = modifiedAt.Time
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

// GetAllMediaPaths returns all media file paths for cleanup
func (s *SQLiteStorage) GetAllMediaPaths() (map[string]string, error) {
	rows, err := s.db.Query("SELECT id, path FROM media_items")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	paths := make(map[string]string)
	for rows.Next() {
		var id, path string
		if err := rows.Scan(&id, &path); err != nil {
			return nil, err
		}
		paths[id] = path
	}
	return paths, rows.Err()
}

// DeleteMediaItem removes a media item by ID
func (s *SQLiteStorage) DeleteMediaItem(id string) error {
	_, err := s.db.Exec("DELETE FROM media_items WHERE id = ?", id)
	return err
}

// GetAllFolderPaths returns all folder paths for cleanup
func (s *SQLiteStorage) GetAllFolderPaths() (map[string]string, error) {
	rows, err := s.db.Query("SELECT id, path FROM folders")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	paths := make(map[string]string)
	for rows.Next() {
		var id, path string
		if err := rows.Scan(&id, &path); err != nil {
			return nil, err
		}
		paths[id] = path
	}
	return paths, rows.Err()
}

// DeleteFolder removes a folder by ID
func (s *SQLiteStorage) DeleteFolder(id string) error {
	_, err := s.db.Exec("DELETE FROM folders WHERE id = ?", id)
	return err
}
