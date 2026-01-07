package service

import (
	"context"
	"fmt"
	"strconv"

	"github.com/lastfm-reader/lastfm-sync/internal/lastfm"
	"github.com/lastfm-reader/lastfm-sync/internal/models"
	"github.com/lastfm-reader/lastfm-sync/internal/progress"
	"github.com/lastfm-reader/lastfm-sync/internal/watermark"
	"github.com/lastfm-reader/lastfm-sync/internal/writer"
	"go.uber.org/zap"
)

// SyncService orchestrates fetching scrobbles and writing them to storage.
type SyncService struct {
	username string
	from     int64 // Unix timestamp; lower bound (inclusive)
	to       int64 // Unix timestamp; upper bound (inclusive)
	pageSize int
	maxPages int
	dryRun   bool
	client   *lastfm.Client
	w        writer.Writer
	wm       watermark.WatermarkStore
	logger   *zap.Logger
	progress progress.ProgressReporter
}

// NewSyncService creates a new SyncService.
func NewSyncService(
	username string,
	from, to int64,
	pageSize, maxPages int,
	dryRun bool,
	client *lastfm.Client,
	w writer.Writer,
	wm watermark.WatermarkStore,
	logger *zap.Logger,
	progressReporter progress.ProgressReporter,
) *SyncService {
	return &SyncService{
		username: username,
		from:     from,
		to:       to,
		pageSize: pageSize,
		maxPages: maxPages,
		dryRun:   dryRun,
		client:   client,
		w:        w,
		wm:       wm,
		logger:   logger,
		progress: progressReporter,
	}
}

// Sync fetches scrobbles and writes them to storage.
// Returns the number of records written and any error.
func (s *SyncService) Sync(ctx context.Context) (int, error) {
	s.logger.Info("sync.start", zap.String("username", s.username), zap.Int64("from", s.from), zap.Int64("to", s.to), zap.Bool("dry_run", s.dryRun))

	// Load existing watermark
	existingWatermark, exists, err := s.wm.Get(ctx, s.username)
	if err != nil {
		s.logger.Error("watermark.get_failed", zap.Error(err))
		return 0, fmt.Errorf("get watermark: %w", err)
	}

	// Effective lower bound: max(--since, watermark)
	lowerBound := s.from
	if exists && existingWatermark > lowerBound {
		lowerBound = existingWatermark
		s.logger.Info("watermark.loaded", zap.Int64("watermark", existingWatermark))
	}

	// DRY RUN MODE: Preview what would happen without side effects
	if s.dryRun {
		s.logger.Info("dry_run.preview",
			zap.String("username", s.username),
			zap.Int64("would_fetch_from", lowerBound),
			zap.Int64("would_fetch_to", s.to),
			zap.Int("page_size", s.pageSize),
			zap.Int("max_pages", s.maxPages),
		)
		s.logger.Info("dry_run.output",
			zap.String("would_write_via", fmt.Sprintf("%T", s.w)),
		)
		if exists {
			s.logger.Info("dry_run.watermark",
				zap.String("current_watermark", strconv.FormatInt(existingWatermark, 10)),
				zap.String("would_update_to", "max_uts_from_fetched_pages"),
			)
		} else {
			s.logger.Info("dry_run.watermark", zap.String("status", "no_existing_watermark"))
		}
		return 0, nil
	}

	totalRecords := 0
	pageNum := 1
	var highestUTS int64 = lowerBound

	// Initialize progress reporting
	// If maxPages set, we know the limit; otherwise estimate based on typical pagination
	totalPages := s.maxPages
	if totalPages <= 0 {
		totalPages = 10 // Initial estimate, will be updated after first page
	}
	s.progress.Start(int64(totalPages), fmt.Sprintf("Fetching scrobbles for %s", s.username))

	// Main fetch loop
	for {
		// Check context
		if ctx.Err() != nil {
			s.logger.Info("sync.cancelled", zap.Error(ctx.Err()))
			return totalRecords, ctx.Err()
		}

		// Max pages limit
		if s.maxPages > 0 && pageNum > s.maxPages {
			s.logger.Info("sync.max_pages_reached", zap.Int("max_pages", s.maxPages))
			break
		}

		// Fetch page
		s.logger.Info("fetch.page.start", zap.Int("page", pageNum), zap.Int("page_size", s.pageSize))
		s.logger.Debug("fetch.page.params",
			zap.String("username", s.username),
			zap.Int64("from", lowerBound),
			zap.Int64("to", s.to),
			zap.Int("page", pageNum),
		)
		page, err := s.client.FetchPage(ctx, s.username, lowerBound, s.to, pageNum, s.pageSize)
		if err != nil {
			s.logger.Error("fetch.page.failed", zap.Error(err), zap.Int("page", pageNum))
			s.progress.FinishWithError(fmt.Sprintf("Failed to fetch page %d: %v", pageNum, err))
			return totalRecords, fmt.Errorf("fetch page %d: %w", pageNum, err)
		}

		s.logger.Info("fetch.page.success", zap.Int("page", pageNum), zap.Int("tracks", len(page.Tracks)))
		s.logger.Debug("fetch.page.details",
			zap.Int("total_pages", page.TotalPages),
			zap.Int("current_page", pageNum),
		)

		// Update progress bar total after first page when we know actual total pages
		if pageNum == 1 && page.TotalPages > 0 {
			actualTotal := page.TotalPages
			if s.maxPages > 0 && actualTotal > s.maxPages {
				actualTotal = s.maxPages // Cap at maxPages if set
			}
			s.progress.SetCurrent(0) // Reset to 0 before updating total
			s.progress.Start(int64(actualTotal), fmt.Sprintf("Fetching %d pages for %s", actualTotal, s.username))
		}

		// Increment progress after successful page fetch
		s.progress.Add(1)
		s.progress.SetDescription(fmt.Sprintf("Page %d/%d: %d tracks", pageNum, page.TotalPages, len(page.Tracks)))

		// Filter and transform tracks to scrobbles
		scrobbles := make([]models.Scrobble, 0, len(page.Tracks))
		pageHighest := int64(0)
		foundWatermark := false

		for _, track := range page.Tracks {
			// Parse UTS from Date.UTS field
			uts := int64(0)
			if track.Date.UTS != "" {
				uts = parseUTS(track.Date.UTS)
			}

			// Skip now-playing tracks (no uts, or nowplaying field set)
			if uts == 0 || track.NowPlaying != "" {
				continue
			}

			// Check short-circuit: if UTS <= existing watermark, mark and continue processing current page
			if uts <= lowerBound {
				s.logger.Info("sync.short_circuit", zap.Int64("uts", uts), zap.Int64("watermark", lowerBound))
				foundWatermark = true
				continue // Skip this track but continue processing remaining tracks on the page
			}

			// Build MBID pointer (can be nil)
			var mbid *string
			if track.MBID != "" {
				mbid = &track.MBID
			}

			// Convert track fields
			artistName := track.Artist.Text
			trackName := track.Name.Text
			albumName := track.Album.Text

			scrobble := models.NewScrobble(
				s.username,
				artistName,
				trackName,
				albumName,
				uts,
				mbid,
				track.Raw,
			)
			scrobbles = append(scrobbles, *scrobble)

			// Track highest UTS
			if uts > pageHighest {
				pageHighest = uts
			}
		}

		// Write batch after processing all tracks on the page
		if len(scrobbles) > 0 {
			if !s.dryRun {
				s.logger.Info("fetch.write.start", zap.Int("records", len(scrobbles)))
				s.logger.Debug("fetch.write.details",
					zap.String("writer_type", fmt.Sprintf("%T", s.w)),
					zap.Int("batch_size", len(scrobbles)),
				)
				if err := s.w.WriteBatch(ctx, scrobbles); err != nil {
					s.logger.Error("fetch.write.failed", zap.Error(err))
					s.progress.FinishWithError(fmt.Sprintf("Failed to write batch: %v", err))
					return totalRecords, fmt.Errorf("write batch: %w", err)
				}

				s.logger.Debug("fetch.flush.start")
				if err := s.w.Flush(ctx); err != nil {
					s.logger.Error("fetch.flush.failed", zap.Error(err))
					s.progress.FinishWithError(fmt.Sprintf("Failed to flush: %v", err))
					return totalRecords, fmt.Errorf("flush: %w", err)
				}

				// Update highest UTS tracker
				if pageHighest > highestUTS {
					highestUTS = pageHighest
				}

				// Update watermark after successful write
				s.logger.Debug("watermark.update.start",
					zap.String("username", s.username),
					zap.Int64("page_highest", pageHighest),
					zap.Int64("absolute_highest", highestUTS),
				)
				if err := s.wm.Put(ctx, s.username, highestUTS); err != nil {
					s.logger.Error("watermark.put.failed", zap.Error(err))
					return totalRecords, fmt.Errorf("update watermark: %w", err)
				}

				s.logger.Info("watermark.updated", zap.Int64("new_watermark", highestUTS))
			} else {
				s.logger.Info("fetch.dry_run", zap.Int("records", len(scrobbles)))
			}

			totalRecords += len(scrobbles)
		}

		// Check if we found the watermark - stop pagination after processing this page
		if foundWatermark {
			s.logger.Info("sync.watermark_reached", zap.Int("page", pageNum))
			break
		}

		// Check if we should exit loop
		if len(page.Tracks) == 0 || page.Page >= page.TotalPages {
			s.logger.Info("sync.pagination.done", zap.Int("pages", page.Page), zap.Int("total_pages", page.TotalPages))
			break
		}

		pageNum++
	}

	// Close writer
	if !s.dryRun {
		if err := s.w.Close(ctx); err != nil {
			s.logger.Error("writer.close.failed", zap.Error(err))
			s.progress.FinishWithError(fmt.Sprintf("Failed to close writer: %v", err))
			return totalRecords, fmt.Errorf("close writer: %w", err)
		}
	}

	s.progress.Finish(fmt.Sprintf("Sync complete: %d records", totalRecords))
	s.logger.Info("sync.done", zap.Int("total_records", totalRecords), zap.Int64("highest_uts", highestUTS))
	return totalRecords, nil
}

// parseUTS converts a UTS string to int64, returns 0 on error.
func parseUTS(s string) int64 {
	uts, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0
	}
	return uts
}
