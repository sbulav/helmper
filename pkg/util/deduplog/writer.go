package deduplog

import (
	"bytes"
	"io"
	"log/slog"
	"regexp"
	"sync"
)

// DeduplicatingWriter wraps an io.Writer and deduplicates log messages
// based on their normalized content. It's thread-safe and tracks suppressed
// message counts for reporting.
type DeduplicatingWriter struct {
	mu            sync.Mutex
	underlying    io.Writer
	seen          map[string]int // normalized message -> count (1 = shown once, >1 = shown + suppressed)
	skipPattern   *regexp.Regexp
	suppressAfter int // suppress after N occurrences (1 = show once, then suppress)
}

// NewDeduplicatingWriter creates a new deduplicating writer that wraps the given writer.
// Messages matching skipPattern are deduplicated - only the first occurrence is written.
func NewDeduplicatingWriter(w io.Writer, skipPattern string) (*DeduplicatingWriter, error) {
	pattern, err := regexp.Compile(skipPattern)
	if err != nil {
		return nil, err
	}

	return &DeduplicatingWriter{
		underlying:    w,
		seen:          make(map[string]int),
		skipPattern:   pattern,
		suppressAfter: 1, // show first occurrence, suppress subsequent
	}, nil
}

// Write implements io.Writer. It deduplicates messages matching the skip pattern.
func (d *DeduplicatingWriter) Write(p []byte) (n int, err error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Check if message matches the skip pattern
	if !d.skipPattern.Match(p) {
		// Not a duplicate-able message, write it directly
		return d.underlying.Write(p)
	}

	// Normalize the message by extracting the key parts
	normalized := d.normalize(p)

	// Track this message
	count := d.seen[normalized]
	d.seen[normalized] = count + 1

	// Only write if we haven't seen it before (or haven't exceeded suppressAfter)
	if count < d.suppressAfter {
		return d.underlying.Write(p)
	}

	// Message suppressed - pretend we wrote it to avoid errors
	return len(p), nil
}

// normalize extracts the meaningful parts of a log message for comparison.
// For Helm validation errors, we normalize based on chart name and version.
func (d *DeduplicatingWriter) normalize(p []byte) string {
	// Pattern: skipping loading invalid entry for chart "NAME" "VERSION" from ...
	// We'll use the full message but trim timestamp if present
	msg := string(bytes.TrimSpace(p))

	// Simple normalization: use the message as-is
	// The messages are already quite specific (chart name + version + error)
	return msg
}

// GetStats returns statistics about suppressed messages.
func (d *DeduplicatingWriter) GetStats() map[string]int {
	d.mu.Lock()
	defer d.mu.Unlock()

	stats := make(map[string]int)
	for msg, count := range d.seen {
		if count > d.suppressAfter {
			stats[msg] = count - d.suppressAfter // only count the suppressed ones
		}
	}
	return stats
}

// LogSummary logs a summary of suppressed messages using slog.
func (d *DeduplicatingWriter) LogSummary() {
	stats := d.GetStats()
	if len(stats) == 0 {
		return
	}

	totalSuppressed := 0
	for _, count := range stats {
		totalSuppressed += count
	}

	if totalSuppressed > 0 {
		slog.Debug("suppressed duplicate helm validation warnings",
			slog.Int("total_suppressed", totalSuppressed),
			slog.Int("unique_messages", len(stats)))
	}
}
