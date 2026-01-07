package merge

import (
	"fmt"
	"time"
)

// Update increments statistics counters
func (s *MergeStats) Update() {
	if !s.StartTime.IsZero() && !s.EndTime.IsZero() {
		s.Duration = s.EndTime.Sub(s.StartTime).Seconds()
		if s.Duration > 0 {
			s.Rate = float64(s.TotalScrobbles) / s.Duration
		}
	}
}

// Summary returns a human-readable summary of merge statistics
func (s *MergeStats) Summary() string {
	return fmt.Sprintf(
		"Processed %d files, %d scrobbles (%d unique, %d duplicates) in %.2fs (%.0f scrobbles/sec)",
		s.ProcessedFiles,
		s.TotalScrobbles,
		s.UniqueScrobbles,
		s.Duplicates,
		s.Duration,
		s.Rate,
	)
}

// SuccessRate returns the percentage of successfully processed scrobbles
func (s *MergeStats) SuccessRate() float64 {
	if s.TotalScrobbles == 0 {
		return 0
	}
	successful := s.TotalScrobbles - s.SkippedScrobbles
	return float64(successful) / float64(s.TotalScrobbles) * 100
}

// DateRange returns the date range of scrobbles as a string
func (s *MergeStats) DateRange() string {
	if s.EarliestTimestamp == 0 || s.LatestTimestamp == 0 {
		return "Unknown"
	}
	earliest := time.Unix(s.EarliestTimestamp, 0).Format("2006-01-02")
	latest := time.Unix(s.LatestTimestamp, 0).Format("2006-01-02")
	return fmt.Sprintf("%s to %s", earliest, latest)
}

// NewStats creates a new MergeStats with initialized time
func NewStats() *MergeStats {
	return &MergeStats{
		StartTime:           time.Now(),
		ConflictsByStrategy: make(map[string]int),
	}
}
