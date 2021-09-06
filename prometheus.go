package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	tentaReqeusts = promauto.NewCounter(prometheus.CounterOpts{
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
)

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
			tentaFiles.Inc()
			tentaSize.Add(float64(file.Size()))
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
