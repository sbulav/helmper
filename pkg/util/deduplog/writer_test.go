package deduplog

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeduplicatingWriter_UniqueMessages(t *testing.T) {
	var buf bytes.Buffer
	writer, err := NewDeduplicatingWriter(&buf, `skipping loading invalid entry`)
	assert.NoError(t, err)

	// Write three different messages
	msg1 := []byte(`skipping loading invalid entry for chart "argo-workflows" "0.46.00" from /root/.cache/helm/repository/argo-index.yaml: validation: chart.metadata.version "0.46.00" is invalid` + "\n")
	msg2 := []byte(`skipping loading invalid entry for chart "cert-manager" "1.00.00" from /root/.cache/helm/repository/cert-manager-index.yaml: validation: chart.metadata.version "1.00.00" is invalid` + "\n")
	msg3 := []byte(`some other log message that doesn't match` + "\n")

	_, err = writer.Write(msg1)
	assert.NoError(t, err)
	_, err = writer.Write(msg2)
	assert.NoError(t, err)
	_, err = writer.Write(msg3)
	assert.NoError(t, err)

	// All three should be written (no duplicates yet)
	output := buf.String()
	assert.Equal(t, 3, strings.Count(output, "\n"))
	assert.Contains(t, output, "argo-workflows")
	assert.Contains(t, output, "cert-manager")
	assert.Contains(t, output, "some other log message")
}

func TestDeduplicatingWriter_DuplicateMessages(t *testing.T) {
	var buf bytes.Buffer
	writer, err := NewDeduplicatingWriter(&buf, `skipping loading invalid entry`)
	assert.NoError(t, err)

	msg := []byte(`skipping loading invalid entry for chart "argo-workflows" "0.46.00" from /root/.cache/helm/repository/argo-index.yaml: validation: chart.metadata.version "0.46.00" is invalid` + "\n")

	// Write the same message 5 times
	for i := 0; i < 5; i++ {
		_, err = writer.Write(msg)
		assert.NoError(t, err)
	}

	// Only the first occurrence should be written
	output := buf.String()
	assert.Equal(t, 1, strings.Count(output, "\n"))
	assert.Contains(t, output, "argo-workflows")

	// Check stats
	stats := writer.GetStats()
	assert.Len(t, stats, 1)
	// 4 suppressed (5 total - 1 shown)
	for _, count := range stats {
		assert.Equal(t, 4, count)
	}
}

func TestDeduplicatingWriter_MixedMessages(t *testing.T) {
	var buf bytes.Buffer
	writer, err := NewDeduplicatingWriter(&buf, `skipping loading invalid entry`)
	assert.NoError(t, err)

	msg1 := []byte(`skipping loading invalid entry for chart "argo-workflows" "0.46.00" from /root/.cache/helm/repository/argo-index.yaml: validation: chart.metadata.version "0.46.00" is invalid` + "\n")
	msg2 := []byte(`some other log message` + "\n")
	msg3 := []byte(`skipping loading invalid entry for chart "cert-manager" "1.00.00" from /root/.cache/helm/repository/cert-manager-index.yaml: validation: chart.metadata.version "1.00.00" is invalid` + "\n")

	// Write pattern: msg1, msg2, msg1, msg3, msg1, msg2
	_, err = writer.Write(msg1)
	assert.NoError(t, err)
	_, err = writer.Write(msg2)
	assert.NoError(t, err)
	_, err = writer.Write(msg1) // duplicate
	assert.NoError(t, err)
	_, err = writer.Write(msg3)
	assert.NoError(t, err)
	_, err = writer.Write(msg1) // duplicate
	assert.NoError(t, err)
	_, err = writer.Write(msg2)
	assert.NoError(t, err)

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Should have 4 lines: msg1(1st), msg2(1st), msg3(1st), msg2(2nd)
	// msg2 doesn't match pattern, so both msg2 instances appear
	// msg1 appears 3 times but only first is written (2 suppressed)
	assert.Equal(t, 4, len(lines))

	// Check stats - msg1 appeared 3 times (1 shown, 2 suppressed), msg3 appeared 1 time (1 shown, 0 suppressed)
	stats := writer.GetStats()
	assert.Len(t, stats, 1) // only msg1 was deduplicated (msg3 has no suppressed count)
}

func TestDeduplicatingWriter_NoPattern(t *testing.T) {
	var buf bytes.Buffer
	writer, err := NewDeduplicatingWriter(&buf, `PATTERN_THAT_NEVER_MATCHES`)
	assert.NoError(t, err)

	msg := []byte(`skipping loading invalid entry for chart "argo-workflows"` + "\n")

	// Write the same message 3 times
	for i := 0; i < 3; i++ {
		_, err = writer.Write(msg)
		assert.NoError(t, err)
	}

	// All messages should be written (pattern doesn't match)
	output := buf.String()
	assert.Equal(t, 3, strings.Count(output, "\n"))

	// No stats (nothing suppressed)
	stats := writer.GetStats()
	assert.Len(t, stats, 0)
}

func TestDeduplicatingWriter_EmptyStats(t *testing.T) {
	var buf bytes.Buffer
	writer, err := NewDeduplicatingWriter(&buf, `skipping loading invalid entry`)
	assert.NoError(t, err)

	// No messages written
	stats := writer.GetStats()
	assert.Len(t, stats, 0)
}

func TestDeduplicatingWriter_InvalidPattern(t *testing.T) {
	var buf bytes.Buffer
	_, err := NewDeduplicatingWriter(&buf, `[invalid(regex`)
	assert.Error(t, err)
}
