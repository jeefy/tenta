package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	tentaRequests  prometheus.Counter
	tentaHits      prometheus.Counter
	tentaMisses    prometheus.Counter
	tentaFiles     prometheus.Gauge
	tentaSize      prometheus.Gauge
	tentaErrors    prometheus.Counter
	tentaNotFound  prometheus.Counter
	tentaServerErr prometheus.Counter

	// Atomic counters for API access
	requestsCount  int64
	hitsCount      int64
	missesCount    int64
	errorsCount    int64
	notFoundCount  int64
	serverErrCount int64
	filesCount     int64
	sizeCount      int64
)

func init() {
	tentaRequests = promauto.NewCounter(prometheus.CounterOpts{
		Name: "tenta_requests_received",
		Help: "The total number of requests",
	})
	tentaHits = promauto.NewCounter(prometheus.CounterOpts{
		Name: "tenta_hits",
		Help: "The total number of cached requests",
	})
	tentaMisses = promauto.NewCounter(prometheus.CounterOpts{
		Name: "tenta_misses",
		Help: "The total number of uncached requests",
	})
	tentaFiles = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "tenta_files",
		Help: "The number of files in the cache",
	})
	tentaSize = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "tenta_size",
		Help: "The size of the cache",
	})
	tentaErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "tenta_errors",
		Help: "The total number of errors",
	})
	tentaNotFound = promauto.NewCounter(prometheus.CounterOpts{
		Name: "tenta_not_found",
		Help: "The total number of 404 responses",
	})
	tentaServerErr = promauto.NewCounter(prometheus.CounterOpts{
		Name: "tenta_server_errors",
		Help: "The total number of 5xx responses",
	})
}

// Helper functions for cache API
func incRequests() {
	tentaRequests.Inc()
	atomic.AddInt64(&requestsCount, 1)
}

func incHits() {
	tentaHits.Inc()
	atomic.AddInt64(&hitsCount, 1)
}

func incMisses() {
	tentaMisses.Inc()
	atomic.AddInt64(&missesCount, 1)
}

func incErrors() {
	tentaErrors.Inc()
	atomic.AddInt64(&errorsCount, 1)
}

func incNotFound() {
	tentaNotFound.Inc()
	atomic.AddInt64(&notFoundCount, 1)
}

func incServerErr() {
	tentaServerErr.Inc()
	atomic.AddInt64(&serverErrCount, 1)
}

func getRequestsCount() int64 {
	return atomic.LoadInt64(&requestsCount)
}

func getHitsCount() int64 {
	return atomic.LoadInt64(&hitsCount)
}

func getMissesCount() int64 {
	return atomic.LoadInt64(&missesCount)
}

func getErrorsCount() int64 {
	return atomic.LoadInt64(&errorsCount)
}

func getNotFoundCount() int64 {
	return atomic.LoadInt64(&notFoundCount)
}

func getServerErrCount() int64 {
	return atomic.LoadInt64(&serverErrCount)
}

func incFiles() {
	tentaFiles.Inc()
	atomic.AddInt64(&filesCount, 1)
}

func decFiles() {
	tentaFiles.Dec()
	atomic.AddInt64(&filesCount, -1)
}

func addSize(size int64) {
	tentaSize.Add(float64(size))
	atomic.AddInt64(&sizeCount, size)
}

func subSize(size int64) {
	tentaSize.Sub(float64(size))
	atomic.AddInt64(&sizeCount, -size)
}

func getFilesCount() int64 {
	return atomic.LoadInt64(&filesCount)
}

func getSizeCount() int64 {
	return atomic.LoadInt64(&sizeCount)
}

func StartMetrics() {
	go func() {
		port := fmt.Sprintf(":%d", 2112)

		if args.debug {
			log.Println("Repopulating prometheus metrics from data directory")
		}

		tmpfiles, err := ioutil.ReadDir(args.dataDir)
		if err != nil {
			log.Fatalf("Error reading data dir %s: %s", args.dataDir, err)
		}

		for _, file := range tmpfiles {
			if file.Mode().IsRegular() {
				incFiles()
				addSize(file.Size())
			}
		}

		log.Printf("Starting metrics server on %s", port)

		s := &http.Server{
			Addr:           port,
			Handler:        promhttp.Handler(),
			ReadTimeout:    60 * time.Second,
			WriteTimeout:   60 * time.Second,
			MaxHeaderBytes: 1 << 20,
		}

		log.Fatal(s.ListenAndServe())
	}()
}
