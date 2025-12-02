package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"
	"rvcinemaview/internal/api"
	"rvcinemaview/internal/config"
	"rvcinemaview/internal/media"
	"rvcinemaview/internal/storage"
)

type Server struct {
	cfg        *config.Config
	logger     zerolog.Logger
	httpServer *http.Server
	router     *chi.Mux
	storage    *storage.SQLiteStorage
	handler    *api.Handler
}

func New(cfg *config.Config, logger zerolog.Logger, store *storage.SQLiteStorage) *Server {
	s := &Server{
		cfg:     cfg,
		logger:  logger,
		storage: store,
	}

	s.router = chi.NewRouter()
	s.setupMiddleware()
	s.setupRoutes()

	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      s.router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	return s
}

func (s *Server) setupMiddleware() {
	s.router.Use(CORSMiddleware)
	s.router.Use(LoggingMiddleware(s.logger))
}

func (s *Server) setupRoutes() {
	s.handler = api.NewHandler(s.storage, s.logger, s.cfg.Library.Path, s.cfg.Library.Name)

	s.router.Route("/api/v1", func(r chi.Router) {
		r.Get("/health", s.handler.Health)

		r.Get("/library/tree", s.handler.GetLibraryTree)
		r.Post("/library/scan", s.handler.ScanLibrary)

		r.Get("/media/{id}", s.handler.GetMedia)
		r.Get("/media/{id}/stream", s.handler.StreamMedia)
		r.Get("/media/{id}/thumbnail", s.handler.GetThumbnail)

		// Playback progress
		r.Post("/playback/{id}/position", s.handler.SavePlaybackPosition)
		r.Get("/playback/{id}/position", s.handler.GetPlaybackPosition)
		r.Get("/playback/continue", s.handler.GetContinueWatching)
	})
}

func (s *Server) SetScanner(scanner api.ScannerInterface) {
	s.handler.SetScanner(scanner)
}

func (s *Server) SetThumbnailService(service *media.ThumbnailService) {
	s.handler.SetThumbnailService(service)
}

func (s *Server) Start() error {
	s.logger.Info().
		Str("addr", s.httpServer.Addr).
		Msg("starting server")

	if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}

	return nil
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info().Msg("shutting down server")

	shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	return s.httpServer.Shutdown(shutdownCtx)
}
