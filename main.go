package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// MongoDB marker files that indicate a data directory
var markerFiles = map[string]bool{
	"WiredTiger":      true, // WiredTiger engine (v3.2+)
	"WiredTiger.lock": true,
	"WiredTiger.wt":   true,
	"mongod.lock":     true,
	"storage.bson":    true,
}

type MongoDir struct {
	Path       string `json:"path"`
	Engine     string `json:"engine"`
	Version    string `json:"version,omitempty"`
	SizeMB     int64  `json:"size_mb"`
	DBNames    []string `json:"databases,omitempty"`
	HasJournal bool   `json:"has_journal"`
}

func main() {
	jsonOut := flag.Bool("json", false, "Output as JSON")
	workers := flag.Int("workers", runtime.NumCPU()*2, "Number of parallel workers")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: scanmongodb [options] <path> [path2...]\n\n")
		fmt.Fprintf(os.Stderr, "Scan directories to find MongoDB database files (v3/v4/v5).\n\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	paths := flag.Args()
	if len(paths) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	start := time.Now()
	var scannedDirs atomic.Int64

	// Channel for directories to inspect
	candidates := make(chan string, 1000)
	results := make(chan MongoDir, 100)

	var wg sync.WaitGroup

	// Producer: walk filesystem and find candidate directories
	go func() {
		var walkers sync.WaitGroup
		for _, root := range paths {
			root := root
			walkers.Add(1)
			go func() {
				defer walkers.Done()
				filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
					if err != nil {
						return nil // skip permission errors etc.
					}
					if d.IsDir() {
						scannedDirs.Add(1)
						name := d.Name()
						// Quick skip known irrelevant dirs
						if name == ".git" || name == "node_modules" || name == ".npm" || name == ".cache" {
							return filepath.SkipDir
						}
					} else if markerFiles[d.Name()] {
						candidates <- filepath.Dir(path)
					}
					return nil
				})
			}()
		}
		walkers.Wait()
		close(candidates)
	}()

	// Dedup candidates (multiple markers in same dir)
	seen := make(map[string]bool)
	var mu sync.Mutex
	deduped := make(chan string, 1000)
	go func() {
		for dir := range candidates {
			mu.Lock()
			if !seen[dir] {
				seen[dir] = true
				mu.Unlock()
				deduped <- dir
			} else {
				mu.Unlock()
			}
		}
		close(deduped)
	}()

	// Workers: inspect each candidate directory
	for i := 0; i < *workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for dir := range deduped {
				if info := inspectMongoDir(dir); info != nil {
					results <- *info
				}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results
	var all []MongoDir
	for r := range results {
		all = append(all, r)
	}

	elapsed := time.Since(start)

	if *jsonOut {
		out := struct {
			Results    []MongoDir `json:"results"`
			ScannedDir int64      `json:"scanned_dirs"`
			ElapsedMs  int64      `json:"elapsed_ms"`
		}{all, scannedDirs.Load(), elapsed.Milliseconds()}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(out)
	} else {
		if len(all) == 0 {
			fmt.Println("No MongoDB data directories found.")
		} else {
			fmt.Printf("Found %d MongoDB data directory(ies):\n\n", len(all))
			for _, m := range all {
				fmt.Printf("  Path:      %s\n", m.Path)
				fmt.Printf("  Engine:    %s\n", m.Engine)
				if m.Version != "" {
					fmt.Printf("  Version:   %s\n", m.Version)
				}
				fmt.Printf("  Size:      %d MB\n", m.SizeMB)
				fmt.Printf("  Journal:   %v\n", m.HasJournal)
				if len(m.DBNames) > 0 {
					fmt.Printf("  Databases: %s\n", strings.Join(m.DBNames, ", "))
				}
				fmt.Println()
			}
		}
		fmt.Printf("Scanned %d dirs in %s\n", scannedDirs.Load(), elapsed.Round(time.Millisecond))
	}
}

func inspectMongoDir(dir string) *MongoDir {
	info := MongoDir{Path: dir}

	// Detect engine
	if fileExists(filepath.Join(dir, "WiredTiger")) || fileExists(filepath.Join(dir, "WiredTiger.wt")) {
		info.Engine = "WiredTiger"
		// Try to read version from WiredTiger file
		if data, err := os.ReadFile(filepath.Join(dir, "WiredTiger")); err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				if strings.HasPrefix(line, "WiredTiger") {
					info.Version = strings.TrimSpace(line)
					break
				}
			}
		}
	} else if hasMMAPv1Files(dir) {
		info.Engine = "MMAPv1"
	} else if fileExists(filepath.Join(dir, "mongod.lock")) {
		info.Engine = "unknown"
	} else {
		return nil // Not a real MongoDB dir
	}

	// Check journal
	info.HasJournal = dirExists(filepath.Join(dir, "journal"))

	// Calculate total size & find database names
	dbNames := make(map[string]bool)
	filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() {
			if fi, err := d.Info(); err == nil {
				info.SizeMB += fi.Size()
			}
			// WiredTiger: database dirs contain collection-*.wt files
			name := d.Name()
			if strings.HasSuffix(name, ".wt") && strings.HasPrefix(name, "collection-") {
				rel, _ := filepath.Rel(dir, filepath.Dir(path))
				if rel != "." && rel != "" {
					dbNames[rel] = true
				}
			}
		} else {
			// Direct subdirs with .wt files are database names
			name := d.Name()
			if name != "journal" && name != "diagnostic.data" && name != "_tmp" {
				rel, _ := filepath.Rel(dir, path)
				if rel != "." && hasWTFiles(path) {
					dbNames[rel] = true
				}
			}
		}
		return nil
	})
	info.SizeMB = info.SizeMB / (1024 * 1024)

	for db := range dbNames {
		info.DBNames = append(info.DBNames, db)
	}

	return &info
}

func fileExists(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && !fi.IsDir()
}

func dirExists(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.IsDir()
}

func hasMMAPv1Files(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".ns") {
			return true
		}
	}
	return false
}

func hasWTFiles(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".wt") {
			return true
		}
	}
	return false
}
