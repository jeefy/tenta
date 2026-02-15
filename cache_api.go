package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// CacheStats represents cache statistics
type CacheStats struct {
	TotalRequests int64   `json:"total_requests"`
	CacheHits     int64   `json:"cache_hits"`
	CacheMisses   int64   `json:"cache_misses"`
	HitRatio      float64 `json:"hit_ratio"`
	FileCount     int     `json:"file_count"`
	CacheSize     int64   `json:"cache_size_bytes"`
}

// handleCacheStats returns current cache statistics
func handleCacheStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	hitsVal := getHitsCount()
	missesVal := getMissesCount()
	requestsVal := getRequestsCount()

	hitRatio := 0.0
	if requestsVal > 0 {
		hitRatio = float64(hitsVal) / float64(requestsVal)
	}

	// Count files manually for accurate count
	files, _ := ioutil.ReadDir(args.dataDir)
	fileCount := 0
	var totalSize int64 = 0
	for _, f := range files {
		if f.Mode().IsRegular() {
			fileCount++
			totalSize += f.Size()
		}
	}

	stats := CacheStats{
		TotalRequests: requestsVal,
		CacheHits:     hitsVal,
		CacheMisses:   missesVal,
		HitRatio:      hitRatio,
		FileCount:     fileCount,
		CacheSize:     totalSize,
	}

	json.NewEncoder(w).Encode(stats)
}

// handleCacheList lists all cached items
func handleCacheList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	files, err := ioutil.ReadDir(args.dataDir)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": fmt.Sprintf("Error reading cache dir: %s", err.Error()),
		})
		return
	}

	type CacheEntry struct {
		Filename string `json:"filename"`
		Size     int64  `json:"size"`
		ModTime  string `json:"mod_time"`
	}

	entries := []CacheEntry{}
	for _, file := range files {
		if file.Mode().IsRegular() {
			entries = append(entries, CacheEntry{
				Filename: file.Name(),
				Size:     file.Size(),
				ModTime:  file.ModTime().String(),
			})
		}
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"count":   len(entries),
		"entries": entries,
	})
}

// handleCacheDelete handles cache deletion
func handleCacheDelete(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodDelete {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "Only DELETE method allowed",
		})
		return
	}

	// Check if specific cache key is provided
	key := strings.TrimPrefix(r.URL.Path, "/cache/delete/")
	if key != "" && key != r.URL.Path {
		// Delete specific cache entry
		filename := filepath.Join(args.dataDir, key)

		// Security check: ensure we're only accessing files in dataDir
		absDataDir, _ := filepath.Abs(args.dataDir)
		absFilename, _ := filepath.Abs(filename)
		if !strings.HasPrefix(absFilename, absDataDir) {
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Invalid cache key",
			})
			return
		}

		file, err := os.Stat(filename)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Cache entry not found",
			})
			return
		}

		if err := os.Remove(filename); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{
				"error": fmt.Sprintf("Failed to delete cache entry: %s", err.Error()),
			})
			return
		}

		tentaSize.Sub(float64(file.Size()))
		tentaFiles.Dec()
		log.Printf("Deleted cache entry: %s", key)

		json.NewEncoder(w).Encode(map[string]string{
			"status": "deleted",
			"key":    key,
		})
	} else {
		// Delete all cache entries
		files, err := ioutil.ReadDir(args.dataDir)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{
				"error": fmt.Sprintf("Error reading cache dir: %s", err.Error()),
			})
			return
		}

		deleted := 0
		for _, file := range files {
			if file.Mode().IsRegular() {
				fullPath := filepath.Join(args.dataDir, file.Name())
				if err := os.Remove(fullPath); err != nil {
					log.Printf("Error deleting %s: %s", fullPath, err)
					continue
				}
				tentaSize.Sub(float64(file.Size()))
				tentaFiles.Dec()
				deleted++
			}
		}

		log.Printf("Cleared entire cache: deleted %d files", deleted)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "cleared",
			"deleted": deleted,
		})
	}
}

// handleCacheInfo returns detailed cache information
func handleCacheInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	files, err := ioutil.ReadDir(args.dataDir)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": fmt.Sprintf("Error reading cache dir: %s", err.Error()),
		})
		return
	}

	// Calculate size distribution
	smallFiles := 0   // < 1MB
	mediumFiles := 0  // 1-10MB
	largeFiles := 0   // 10-100MB
	hugeFiles := 0    // > 100MB

	for _, file := range files {
		if !file.Mode().IsRegular() {
			continue
		}
		size := file.Size()
		if size < 1024*1024 {
			smallFiles++
		} else if size < 10*1024*1024 {
			mediumFiles++
		} else if size < 100*1024*1024 {
			largeFiles++
		} else {
			hugeFiles++
		}
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"total_files": len(files),
		"size_distribution": map[string]int{
			"small_<1mb":     smallFiles,
			"medium_1-10mb":  mediumFiles,
			"large_10-100mb": largeFiles,
			"huge_>100mb":    hugeFiles,
		},
	})
}
