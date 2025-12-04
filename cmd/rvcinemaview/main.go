package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"rvcinemaview/internal/api"
	"rvcinemaview/internal/config"
	"rvcinemaview/internal/media"
	"rvcinemaview/internal/server"
	"rvcinemaview/internal/storage"
)

func main() {
	configPath := flag.String("config", "", "path to config file")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		panic("failed to load config: " + err.Error())
	}

	// Setup logger
	logger := setupLogger(cfg.Logging)

	logger.Info().
		Str("version", api.Version).
		Msg("starting RVCinemaView server")

	// Initialize storage
	store, err := storage.NewSQLiteStorage(cfg.Database.Path)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to initialize storage")
	}
	defer store.Close()

	// Initialize scanner
	scanner := media.NewScanner(store, logger)

	// Initialize metadata extractor and thumbnail generator
	metadataExtractor := media.NewMetadataExtractor(logger)
	thumbnailGenerator := media.NewThumbnailGenerator(cfg.Thumbnails.OutputDir, logger)

	// Log ffmpeg/ffprobe availability
	if metadataExtractor.IsAvailable() {
		logger.Info().Msg("ffprobe available - metadata extraction enabled")
	} else {
		logger.Warn().Msg("ffprobe not found - metadata extraction disabled")
	}
	if thumbnailGenerator.IsAvailable() {
		logger.Info().Msg("ffmpeg available - thumbnail generation enabled")
	} else {
		logger.Warn().Msg("ffmpeg not found - thumbnail generation disabled")
	}

	// Initialize thumbnail service
	thumbnailService := media.NewThumbnailService(
		thumbnailGenerator,
		metadataExtractor,
		store,
		cfg.Thumbnails.CacheCapacity,
		cfg.Thumbnails.CacheMaxSize,
		logger,
	)

	// Create server
	srv := server.New(cfg, logger, store)
	srv.SetScanner(scanner)
	srv.SetThumbnailService(thumbnailService)

	// Handle shutdown signals
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initial scan if library path configured
	if cfg.Library.Path != "" {
		go func() {
			logger.Info().
				Str("path", cfg.Library.Path).
				Str("name", cfg.Library.Name).
				Msg("starting initial library scan")
			if err := scanner.ScanPath(cfg.Library.Path, cfg.Library.Name); err != nil {
				logger.Error().Err(err).Msg("initial scan failed")
			} else {
				logger.Info().Msg("initial scan completed")
				// Start background metadata/thumbnail processing after scan
				thumbnailService.StartBackgroundProcessing(ctx, 100, 500*time.Millisecond)
			}
		}()
	}

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh

		logger.Info().Msg("received shutdown signal")
		cancel()

		if err := srv.Shutdown(ctx); err != nil {
			logger.Error().Err(err).Msg("shutdown error")
		}
	}()

	// Start server
	if err := srv.Start(); err != nil {
		logger.Error().Err(err).Msg("server error")
	}

	logger.Info().Msg("server stopped")
}

func setupLogger(cfg config.LoggingConfig) zerolog.Logger {
	level, err := zerolog.ParseLevel(cfg.Level)
	if err != nil {
		level = zerolog.InfoLevel
	}

	zerolog.SetGlobalLevel(level)

	if cfg.Pretty {
		return zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout}).
			With().
			Timestamp().
			Logger()
	}

	return zerolog.New(os.Stdout).
		With().
		Timestamp().
		Logger()
}
