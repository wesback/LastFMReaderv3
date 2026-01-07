package merge_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lastfm-reader/lastfm-sync/internal/merge"
	"github.com/lastfm-reader/lastfm-sync/internal/models"
)

// BenchmarkMerge10K benchmarks merge performance with 10K scrobbles
// Target: ≥10,000 scrobbles/sec as per performance requirements
func BenchmarkMerge10K(b *testing.B) {
	// Setup: Create test files with 10K total scrobbles
	tmpDir := b.TempDir()
	files := createBenchmarkFiles(b, tmpDir, 10000, 5)

	cfg := merge.MergeConfig{
		Strategy:           merge.StrategyDefault,
		ConflictResolution: merge.ResolutionCompleteness,
		CheckpointInterval: 100000, // No checkpoints during benchmark
		ProgressEnabled:    false,  // Disable progress bar
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		outputPath := filepath.Join(tmpDir, "bench-output.json")

		merger := merge.NewMerger(cfg)
		_, err := merger.Merge(files, outputPath)
		if err != nil {
			b.Fatalf("Merge failed: %v", err)
		}

		// Clean up output file for next iteration
		os.Remove(outputPath)
	}

	// Report scrobbles/sec
	scrobblesPerOp := float64(10000)
	opsPerSec := float64(b.N) / b.Elapsed().Seconds()
	scrobblesPerSec := scrobblesPerOp * opsPerSec

	b.ReportMetric(scrobblesPerSec, "scrobbles/sec")

	if scrobblesPerSec < 10000 {
		b.Logf("WARNING: Performance target not met. Got %.0f scrobbles/sec, want ≥10000", scrobblesPerSec)
	}
}

// BenchmarkMerge100K benchmarks merge performance with 100K scrobbles
func BenchmarkMerge100K(b *testing.B) {
	tmpDir := b.TempDir()
	files := createBenchmarkFiles(b, tmpDir, 100000, 10)

	cfg := merge.MergeConfig{
		Strategy:           merge.StrategyDefault,
		ConflictResolution: merge.ResolutionCompleteness,
		CheckpointInterval: 100000,
		ProgressEnabled:    false,
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		outputPath := filepath.Join(tmpDir, "bench-output.json")

		merger := merge.NewMerger(cfg)
		_, err := merger.Merge(files, outputPath)
		if err != nil {
			b.Fatalf("Merge failed: %v", err)
		}

		os.Remove(outputPath)
	}

	scrobblesPerOp := float64(100000)
	opsPerSec := float64(b.N) / b.Elapsed().Seconds()
	scrobblesPerSec := scrobblesPerOp * opsPerSec

	b.ReportMetric(scrobblesPerSec, "scrobbles/sec")
}

// BenchmarkMerge1M benchmarks merge performance with 1M scrobbles
// Tests memory efficiency requirement: <500MB for 1M records
func BenchmarkMerge1M(b *testing.B) {
	tmpDir := b.TempDir()
	files := createBenchmarkFiles(b, tmpDir, 1000000, 20)

	cfg := merge.MergeConfig{
		Strategy:           merge.StrategyDefault,
		ConflictResolution: merge.ResolutionCompleteness,
		CheckpointInterval: 100000,
		ProgressEnabled:    false,
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		outputPath := filepath.Join(tmpDir, "bench-output.json")

		merger := merge.NewMerger(cfg)
		_, err := merger.Merge(files, outputPath)
		if err != nil {
			b.Fatalf("Merge failed: %v", err)
		}

		os.Remove(outputPath)
	}

	scrobblesPerOp := float64(1000000)
	opsPerSec := float64(b.N) / b.Elapsed().Seconds()
	scrobblesPerSec := scrobblesPerOp * opsPerSec

	b.ReportMetric(scrobblesPerSec, "scrobbles/sec")

	// Note: Memory usage should be verified with -benchmem flag
	// Target: <500MB for 1M scrobbles
}

// BenchmarkDeduplication benchmarks deduplication performance specifically
func BenchmarkDeduplication(b *testing.B) {
	// Create test scrobbles with various duplication patterns
	baseTime := time.Now().Unix()
	scrobbles := make([]*models.Scrobble, 10000)

	for i := 0; i < 10000; i++ {
		scrobbles[i] = &models.Scrobble{
			Username: "benchuser",
			Artist:   "Artist " + string(rune('A'+(i%100))),
			Track:    "Track " + string(rune('A'+(i%50))),
			Album:    "Album " + string(rune('A'+(i%25))),
			UTS:      baseTime + int64(i%1000),
		}
	}

	cfg := merge.MergeConfig{
		Strategy:           merge.StrategyDefault,
		ConflictResolution: merge.ResolutionCompleteness,
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		dedupMap := merge.NewDeduplicationMap(cfg.Strategy, cfg.ConflictResolution)

		for _, s := range scrobbles {
			dedupMap.Add(s)
		}

		_ = dedupMap.GetAll()
	}

	scrobblesPerOp := float64(10000)
	opsPerSec := float64(b.N) / b.Elapsed().Seconds()
	scrobblesPerSec := scrobblesPerOp * opsPerSec

	b.ReportMetric(scrobblesPerSec, "scrobbles/sec")
}

// BenchmarkStrategyDefault benchmarks default strategy key generation
func BenchmarkStrategyDefault(b *testing.B) {
	s := &models.Scrobble{
		Username: "user",
		Artist:   "The Beatles",
		Track:    "Come Together",
		Album:    "Abbey Road",
		UTS:      1234567890,
	}

	dedupMap := merge.NewDeduplicationMap(merge.StrategyDefault, merge.ResolutionCompleteness)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		dedupMap.Add(s)
	}
}

// BenchmarkStrategyStrict benchmarks strict strategy key generation
func BenchmarkStrategyStrict(b *testing.B) {
	s := &models.Scrobble{
		Username: "user",
		Artist:   "The Beatles",
		Track:    "Come Together",
		Album:    "Abbey Road",
		UTS:      1234567890,
	}

	dedupMap := merge.NewDeduplicationMap(merge.StrategyStrict, merge.ResolutionCompleteness)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		dedupMap.Add(s)
	}
}

// BenchmarkConflictResolution benchmarks conflict resolution performance
func BenchmarkConflictResolution(b *testing.B) {
	existing := &models.Scrobble{
		Username: "user",
		Artist:   "Artist",
		Track:    "Track",
		UTS:      1234567890,
	}

	new := &models.Scrobble{
		Username: "user",
		Artist:   "Artist",
		Track:    "Track",
		Album:    "Album", // More complete
		UTS:      1234567890,
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = merge.ResolveConflict(existing, new, merge.ResolutionCompleteness)
	}
}

// createBenchmarkFiles creates test NDJSON files for benchmarking
func createBenchmarkFiles(b *testing.B, dir string, totalScrobbles, numFiles int) []string {
	b.Helper()

	scrobblesPerFile := totalScrobbles / numFiles
	files := make([]string, numFiles)
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC).Unix()

	for fileIdx := 0; fileIdx < numFiles; fileIdx++ {
		filePath := filepath.Join(dir, "bench-input-"+string(rune('0'+fileIdx))+".ndjson")
		files[fileIdx] = filePath

		f, err := os.Create(filePath)
		if err != nil {
			b.Fatalf("Failed to create benchmark file: %v", err)
		}

		// Generate scrobbles with realistic distribution
		// 10% duplicates across files, 90% unique
		for i := 0; i < scrobblesPerFile; i++ {
			var s models.Scrobble

			if i < scrobblesPerFile/10 {
				// Duplicate scrobbles (same across files)
				s = models.Scrobble{
					Username: "benchuser",
					Artist:   "Duplicate Artist " + string(rune('A'+(i%26))),
					Track:    "Duplicate Track " + string(rune('A'+(i%26))),
					Album:    "Duplicate Album",
					UTS:      baseTime + int64(i),
				}
			} else {
				// Unique scrobbles
				offset := fileIdx*scrobblesPerFile + i
				s = models.Scrobble{
					Username: "benchuser",
					Artist:   "Artist " + string(rune('A'+(offset%26))),
					Track:    "Track " + string(rune('A'+(offset%26))),
					Album:    "Album " + string(rune('A'+(offset%26))),
					UTS:      baseTime + int64(offset),
				}
			}

			if err := json.NewEncoder(f).Encode(s); err != nil {
				f.Close()
				b.Fatalf("Failed to write scrobble: %v", err)
			}
		}

		f.Close()
	}

	return files
}
