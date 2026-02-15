package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// CacheStats represents cache statistics
type CacheStats struct {
	TotalRequests  int64   `json:"total_requests"`
	CacheHits      int64   `json:"cache_hits"`
	CacheMisses    int64   `json:"cache_misses"`
	HitRatio       float64 `json:"hit_ratio"`
	NotFound       int64   `json:"not_found_404"`
	ServerErrors   int64   `json:"server_errors_5xx"`
	OtherErrors    int64   `json:"other_errors"`
	FileCount      int64   `json:"file_count"`
	CacheSize      int64   `json:"cache_size_bytes"`
}

// HealthStatus represents service health information
type HealthStatus struct {
	Status       string `json:"status"`
	Uptime       string `json:"uptime"`
	DataDir      string `json:"data_dir"`
	CacheFiles   int64  `json:"cache_files"`
	CacheSize    int64  `json:"cache_size_bytes"`
	MaxCacheAge  int    `json:"max_cache_age_hours"`
	CronSchedule string `json:"cron_schedule"`
}

// CacheEntry represents a single cached file
type CacheEntry struct {
	Filename string `json:"filename"`
	Size     int64  `json:"size"`
	ModTime  string `json:"mod_time"`
}

// CacheListResponse represents the response for listing cache entries
type CacheListResponse struct {
	Count   int           `json:"count"`
	Entries []CacheEntry  `json:"entries"`
}

var startTime = time.Now()

// handleHealth returns service health status
func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	health := HealthStatus{
		Status:       "healthy",
		Uptime:       time.Since(startTime).String(),
		DataDir:      args.dataDir,
		CacheFiles:   getFilesCount(),
		CacheSize:    getSizeCount(),
		MaxCacheAge:  args.maxCacheAge,
		CronSchedule: args.cronSchedule,
	}

	json.NewEncoder(w).Encode(health)
}

// handleCacheStats returns current cache statistics
func handleCacheStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	totalReq := getRequestsCount()
	hits := getHitsCount()
	misses := getMissesCount()
	notFound := getNotFoundCount()
	serverErrors := getServerErrCount()
	otherErrors := getErrorsCount()

	hitRatio := 0.0
	if totalReq > 0 {
		hitRatio = float64(hits) / float64(totalReq)
	}

	stats := CacheStats{
		TotalRequests: totalReq,
		CacheHits:     hits,
		CacheMisses:   misses,
		HitRatio:      hitRatio,
		NotFound:      notFound,
		ServerErrors:  serverErrors,
		OtherErrors:   otherErrors,
		FileCount:     getFilesCount(),
		CacheSize:     getSizeCount(),
	}

	json.NewEncoder(w).Encode(stats)
}

// handleCacheList lists all cached items
func handleCacheList(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	files, err := os.ReadDir(args.dataDir)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"error": fmt.Sprintf("Error reading cache dir: %s", err.Error()),
		})
		incErrors()
		return
	}

	entries := []CacheEntry{}
	for _, file := range files {
		if !file.IsDir() {
			fileInfo, _ := file.Info()
			entries = append(entries, CacheEntry{
				Filename: file.Name(),
				Size:     fileInfo.Size(),
				ModTime:  fileInfo.ModTime().String(),
			})
		}
	}

	response := CacheListResponse{
		Count:   len(entries),
		Entries: entries,
	}

	json.NewEncoder(w).Encode(response)
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
	key := strings.TrimPrefix(r.URL.Path, "/api/cache/delete/")
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
			incErrors()
			return
		}

		subSize(file.Size())
		decFiles()
		log.Printf("Deleted cache entry: %s", key)

		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "deleted",
			"key":    key,
			"size":   file.Size(),
		})
	} else {
		// Delete all cache entries
		files, err := os.ReadDir(args.dataDir)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{
				"error": fmt.Sprintf("Error reading cache dir: %s", err.Error()),
			})
			incErrors()
			return
		}

		deleted := 0
		var totalSize int64
		for _, file := range files {
			if !file.IsDir() {
				fileInfo, _ := file.Info()
				fullPath := filepath.Join(args.dataDir, file.Name())
				if err := os.Remove(fullPath); err != nil {
					log.Printf("Error deleting %s: %s", fullPath, err)
					incErrors()
					continue
				}
				subSize(fileInfo.Size())
				decFiles()
				totalSize += fileInfo.Size()
				deleted++
			}
		}

		log.Printf("Cleared entire cache: deleted %d files", deleted)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":      "cleared",
			"deleted":     deleted,
			"size_freed":  totalSize,
		})
	}
}

func StartHTTP() {
	port := fmt.Sprintf(":%d", args.httpPort)
	log.Printf("Starting HTTP server on %s", port)
	// Create a mux for routing incoming requests
	myHandler := http.NewServeMux()

	// API endpoints
	myHandler.HandleFunc("/api/health", handleHealth)
	myHandler.HandleFunc("/api/cache/stats", handleCacheStats)
	myHandler.HandleFunc("/api/cache/list", handleCacheList)
	myHandler.HandleFunc("/api/cache/delete", handleCacheDelete)
	myHandler.HandleFunc("/api/cache/delete/", handleCacheDelete)

	// Proxy endpoint (all other paths)
	myHandler.HandleFunc("/", handleRequest)

	s := &http.Server{
		Addr:           port,
		Handler:        myHandler,
		ReadTimeout:    60 * time.Second,
		WriteTimeout:   60 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()
	log.Print("Server Started")

	<-done
	log.Print("Server Stopped")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		// extra handling here
		cancel()
	}()

	if err := s.Shutdown(ctx); err != nil {
		log.Fatalf("Server Shutdown Failed:%+v", err)
	}
	log.Print("Server Exited Properly")
}
