package main

import (
	"context"
	"flag"
	"fmt"
	"io/fs"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"time"

	ds "github.com/ipfs/go-datastore"
	badger4 "github.com/ipfs/go-ds-badger4"

	"github.com/evstack/ev-node/pkg/store"
)

type config struct {
	baseDir        string
	reset          bool
	keepTemp       bool
	totalBytes     int64
	valueSize      int
	batchSize      int
	overwriteRatio float64
	profile        string
}

type profile struct {
	name string
	open func(path string) (ds.Batching, error)
}

type result struct {
	profile    string
	writes     int
	bytes      int64
	duration   time.Duration
	mbPerSec   float64
	writesPerS float64
	dbSize     int64
}

func main() {
	cfg := parseFlags()

	profiles := []profile{
		{name: "evnode", open: openEvnode},
		{name: "default", open: openDefault},
	}

	baseDir, cleanup := ensureBaseDir(cfg.baseDir, cfg.keepTemp)
	defer cleanup()

	selected := selectProfiles(profiles, cfg.profile)
	if len(selected) == 0 {
		fmt.Fprintf(os.Stderr, "Unknown profile %q (valid: evnode, default, all)\n", cfg.profile)
		os.Exit(1)
	}

	for _, p := range selected {
		profileDir := filepath.Join(baseDir, p.name)
		if cfg.reset {
			_ = os.RemoveAll(profileDir)
		}
		if err := os.MkdirAll(profileDir, 0o755); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to create db dir %s: %v\n", profileDir, err)
			os.Exit(1)
		}

		res, err := runProfile(p, profileDir, cfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Profile %s failed: %v\n", p.name, err)
			os.Exit(1)
		}
		printResult(res)
	}
}

func parseFlags() config {
	cfg := config{}
	flag.StringVar(&cfg.baseDir, "dir", "", "DB base directory (default: temp dir)")
	flag.BoolVar(&cfg.reset, "reset", false, "remove existing DB directory before running")
	flag.BoolVar(&cfg.keepTemp, "keep", false, "keep temp directory (only used when -dir is empty)")
	flag.Int64Var(&cfg.totalBytes, "bytes", 512<<20, "total bytes to write")
	flag.IntVar(&cfg.valueSize, "value-size", 1024, "value size in bytes")
	flag.IntVar(&cfg.batchSize, "batch-size", 1000, "writes per batch commit")
	flag.Float64Var(&cfg.overwriteRatio, "overwrite-ratio", 0.1, "fraction of writes that overwrite existing keys (0..1)")
	flag.StringVar(&cfg.profile, "profile", "evnode", "profile to run: evnode, default, all")
	flag.Parse()

	if cfg.totalBytes <= 0 {
		exitError("bytes must be > 0")
	}
	if cfg.valueSize <= 0 {
		exitError("value-size must be > 0")
	}
	if cfg.batchSize <= 0 {
		exitError("batch-size must be > 0")
	}
	if cfg.overwriteRatio < 0 || cfg.overwriteRatio > 1 {
		exitError("overwrite-ratio must be between 0 and 1")
	}

	return cfg
}

func runProfile(p profile, dir string, cfg config) (result, error) {
	totalWrites := int(cfg.totalBytes / int64(cfg.valueSize))
	if totalWrites == 0 {
		return result{}, fmt.Errorf("total bytes (%d) smaller than value size (%d)", cfg.totalBytes, cfg.valueSize)
	}
	actualBytes := int64(totalWrites) * int64(cfg.valueSize)

	rng := rand.New(rand.NewSource(1)) // Deterministic data for comparable runs.
	value := make([]byte, cfg.valueSize)
	if _, err := rng.Read(value); err != nil {
		return result{}, fmt.Errorf("failed to seed value bytes: %w", err)
	}

	overwriteEvery := 0
	if cfg.overwriteRatio > 0 {
		overwriteEvery = int(math.Round(1.0 / cfg.overwriteRatio))
		if overwriteEvery < 1 {
			overwriteEvery = 1
		}
	}

	kv, err := p.open(dir)
	if err != nil {
		return result{}, fmt.Errorf("failed to open db: %w", err)
	}

	ctx := context.Background()
	start := time.Now()

	batch, err := kv.Batch(ctx)
	if err != nil {
		_ = kv.Close()
		return result{}, fmt.Errorf("failed to create batch: %w", err)
	}

	pending := 0
	keysWritten := 0
	for i := 0; i < totalWrites; i++ {
		keyIndex := keysWritten
		if overwriteEvery > 0 && i%overwriteEvery == 0 && keysWritten > 0 {
			keyIndex = i % keysWritten
		} else {
			keysWritten++
		}

		if err := batch.Put(ctx, keyForIndex(keyIndex), value); err != nil {
			_ = kv.Close()
			return result{}, fmt.Errorf("batch put failed: %w", err)
		}

		pending++
		if pending == cfg.batchSize {
			if err := batch.Commit(ctx); err != nil {
				_ = kv.Close()
				return result{}, fmt.Errorf("batch commit failed: %w", err)
			}
			batch, err = kv.Batch(ctx)
			if err != nil {
				_ = kv.Close()
				return result{}, fmt.Errorf("failed to create batch: %w", err)
			}
			pending = 0
		}
	}

	if pending > 0 {
		if err := batch.Commit(ctx); err != nil {
			_ = kv.Close()
			return result{}, fmt.Errorf("final batch commit failed: %w", err)
		}
	}

	if err := kv.Sync(ctx, ds.NewKey("/")); err != nil {
		_ = kv.Close()
		return result{}, fmt.Errorf("sync failed: %w", err)
	}

	if err := kv.Close(); err != nil {
		return result{}, fmt.Errorf("close failed: %w", err)
	}

	duration := time.Since(start)
	mbPerSec := (float64(actualBytes) / (1024 * 1024)) / duration.Seconds()
	writesPerSec := float64(totalWrites) / duration.Seconds()
	dbSize, err := dirSize(dir)
	if err != nil {
		return result{}, fmt.Errorf("failed to compute db size: %w", err)
	}

	return result{
		profile:    p.name,
		writes:     totalWrites,
		bytes:      actualBytes,
		duration:   duration,
		mbPerSec:   mbPerSec,
		writesPerS: writesPerSec,
		dbSize:     dbSize,
	}, nil
}

func openEvnode(path string) (ds.Batching, error) {
	return badger4.NewDatastore(path, store.BadgerOptions())
}

func openDefault(path string) (ds.Batching, error) {
	return badger4.NewDatastore(path, nil)
}

func keyForIndex(i int) ds.Key {
	return ds.NewKey("k/" + strconv.Itoa(i))
}

func dirSize(root string) (int64, error) {
	var size int64
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type().IsRegular() {
			info, err := d.Info()
			if err != nil {
				return err
			}
			size += info.Size()
		}
		return nil
	})
	return size, err
}

func printResult(res result) {
	fmt.Printf("\nProfile: %s\n", res.profile)
	fmt.Printf("Writes: %d\n", res.writes)
	fmt.Printf("Data: %s\n", humanBytes(res.bytes))
	fmt.Printf("Duration: %s\n", res.duration)
	fmt.Printf("Throughput: %.2f MB/s\n", res.mbPerSec)
	fmt.Printf("Writes/sec: %.2f\n", res.writesPerS)
	fmt.Printf("DB size: %s\n", humanBytes(res.dbSize))
}

func selectProfiles(profiles []profile, name string) []profile {
	if name == "all" {
		return profiles
	}
	for _, p := range profiles {
		if p.name == name {
			return []profile{p}
		}
	}
	return nil
}

func ensureBaseDir(dir string, keep bool) (string, func()) {
	if dir != "" {
		return dir, func() {}
	}

	tempDir, err := os.MkdirTemp("", "evnode-db-bench-*")
	if err != nil {
		exitError(fmt.Sprintf("failed to create temp dir: %v", err))
	}

	if keep {
		fmt.Printf("Using temp dir: %s\n", tempDir)
		return tempDir, func() {}
	}

	return tempDir, func() { _ = os.RemoveAll(tempDir) }
}

func humanBytes(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(size)/float64(div), "KMGTPE"[exp])
}

func exitError(msg string) {
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(1)
}
