package main

import (
	"bytes"
	"encoding/gob"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/evstack/ev-node/types"
)

// DataHashForEmptyTxs is the hash of an empty block data.
var DataHashForEmptyTxs = []byte{110, 52, 11, 156, 255, 179, 122, 152, 156, 165, 68, 230, 187, 120, 10, 44, 120, 144, 29, 63, 179, 55, 56, 118, 133, 17, 163, 6, 23, 175, 160, 29}

const (
	pendingEventsCacheDir = "cache/pending_da_events"
	itemsByHeightFilename = "items_by_height.gob"
)

// DAHeightEvent represents a DA event for caching (copied from internal package)
type DAHeightEvent struct {
	Header *types.SignedHeader
	Data   *types.Data
	// DaHeight corresponds to the highest DA included height between the Header and Data.
	DaHeight uint64
}

// registerGobTypes registers types needed for decoding
func registerGobTypes() {
	gob.Register(&types.SignedHeader{})
	gob.Register(&types.Data{})
	gob.Register(&DAHeightEvent{})
}

// loadEventsFromDisk loads pending events from the cache directory
func loadEventsFromDisk(dataDir string) (map[uint64]*DAHeightEvent, error) {
	registerGobTypes()

	cachePath := filepath.Join(dataDir, pendingEventsCacheDir, itemsByHeightFilename)

	file, err := os.Open(cachePath)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[uint64]*DAHeightEvent), nil
		}
		return nil, fmt.Errorf("failed to open cache file %s: %w", cachePath, err)
	}
	defer file.Close()

	var events map[uint64]*DAHeightEvent
	decoder := gob.NewDecoder(file)
	if err := decoder.Decode(&events); err != nil {
		return nil, fmt.Errorf("failed to decode cache file: %w", err)
	}

	return events, nil
}

// formatTable creates a formatted table string
func formatTable(events []eventEntry) string {
	if len(events) == 0 {
		return "No pending events found.\n"
	}

	var sb strings.Builder

	// Header
	sb.WriteString("┌─────────────┬──────────────┬─────────────────────────────────────────────────────────────────────┐\n")
	sb.WriteString("│   Height    │  DA Height   │                            Hash/Details                             │\n")
	sb.WriteString("├─────────────┼──────────────┼─────────────────────────────────────────────────────────────────────┤\n")

	for _, entry := range events {
		heightStr := fmt.Sprintf("%d", entry.Height)
		daHeightStr := fmt.Sprintf("%d", entry.DAHeight)

		// Format each row
		sb.WriteString(fmt.Sprintf("│ %-11s │ %-12s │ %-67s │\n",
			heightStr, daHeightStr, truncateString(entry.Details, 67)))
	}

	sb.WriteString("└─────────────┴──────────────┴─────────────────────────────────────────────────────────────────────┘\n")

	return sb.String()
}

// truncateString truncates a string to maxLen, adding "..." if needed
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return "..."
	}
	return s[:maxLen-3] + "..."
}

// eventEntry represents a table row
type eventEntry struct {
	Height   uint64
	DAHeight uint64
	Details  string
}

// analyzeEvents processes the events and creates table entries
func analyzeEvents(events map[uint64]*DAHeightEvent, limit int) []eventEntry {
	var entries []eventEntry

	// Convert map to sorted slice by height
	heights := make([]uint64, 0, len(events))
	for height := range events {
		heights = append(heights, height)
	}
	sort.Slice(heights, func(i, j int) bool {
		return heights[i] < heights[j]
	})

	// Apply limit
	if limit > 0 && len(heights) > limit {
		heights = heights[:limit]
	}

	for _, height := range heights {
		event := events[height]
		if event == nil {
			continue
		}

		headerHash := "N/A"
		if event.Header.Hash() != nil {
			headerHash = fmt.Sprintf("%.8x", event.Header.Hash())
		}
		dataHash := "N/A"
		if event.Data.DACommitment() != nil {
			if bytes.Equal(event.Data.DACommitment(), DataHashForEmptyTxs) {
				dataHash = "Hash for empty transactions"
			} else {
				dataHash = fmt.Sprintf("%.8x", event.Data.DACommitment())
			}
		}
		txCount := 0
		if event.Data.Txs != nil {
			txCount = len(event.Data.Txs)
		}
		details := fmt.Sprintf("H:%s D:%s TxCount:%d", headerHash, dataHash, txCount)

		entries = append(entries, eventEntry{
			Height:   height,
			DAHeight: event.DaHeight,
			Details:  details,
		})
	}

	return entries
}

// printSummary prints a summary of the cache contents
func printSummary(events map[uint64]*DAHeightEvent) {
	if len(events) == 0 {
		fmt.Println("Cache Summary: No events found")
		return
	}

	var minHeight, maxHeight uint64
	var minDaHeight, maxDaHeight uint64
	first := true

	for height, event := range events {
		if first {
			minHeight = height
			maxHeight = height
			minDaHeight = event.DaHeight
			maxDaHeight = event.DaHeight
			first = false
		} else {
			if height < minHeight {
				minHeight = height
				minDaHeight = event.DaHeight
			}
			if height > maxHeight {
				maxHeight = height
				maxDaHeight = event.DaHeight
			}
		}

	}

	fmt.Printf("Cache Summary:\n")
	fmt.Printf("  Total Events: %d\n", len(events))
	fmt.Printf("  Height Range: %d - %d\n", minHeight, maxHeight)
	fmt.Printf("  DA Height Range: %d - %d\n", minDaHeight, maxDaHeight)
	fmt.Printf("\n")
}

func main() {
	var (
		dataDir = flag.String("data-dir", "data", "Path to the data directory containing cache")
		limit   = flag.Int("limit", 10, "Maximum number of events to display (0 for no limit)")
		summary = flag.Bool("summary", false, "Show only summary without table")
	)
	flag.Parse()

	// Load events from cache
	events, err := loadEventsFromDisk(*dataDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading cache: %v\n", err)
		os.Exit(1)
	}

	// Print summary
	printSummary(events)

	// Exit if only summary requested
	if *summary {
		return
	}

	// Analyze and display events
	entries := analyzeEvents(events, *limit)

	if len(entries) == 0 {
		fmt.Println("No pending events to display.")
		return
	}

	if *limit > 0 && len(events) > *limit {
		fmt.Printf("Showing first %d of %d events:\n\n", len(entries), len(events))
	}

	fmt.Print(formatTable(entries))
}
