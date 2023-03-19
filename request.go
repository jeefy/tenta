package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/segmentio/fasthash/fnv1a"
)

func handleRequest(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("tenta-proxy") == "true" {
		log.Printf("Proxy loop detected, aborting")
		return
	}
	url := generateURL(r)
	log.Printf("Retrieving %s", url)

	h1 := generateCacheFilename(url, r)

	filename := fmt.Sprintf("%s/%s", args.dataDir, h1)
	tentaReqeusts.Inc()
	if file, err := os.Open(filename); os.IsNotExist(err) {
		if args.debug {
			log.Printf("Cache file %s not found", filename)
		}
		tentaMisses.Inc()
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			log.Printf("Error creating request: %s", err)
			return
		}
		req.Header.Add("tenta-proxy", `true`)
		req.Header.Add("request-timestamp", fmt.Sprintf("%d", time.Now().Unix()))
		client := &http.Client{}
		data, err := client.Do(req)
		//data, err := http.Get(url)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, "404! Not found")
			return
		}
		defer data.Body.Close()
		out, err := os.Create(filename)
		if err != nil {
			sent, err := io.Copy(w, data.Body)
			if err != nil {
				log.Printf("Error creating local file, no data sent: %s", err)
			}
			log.Printf("Error creating local file, sent %d bytes: %s", sent, err)
			return
		}
		if args.debug {
			log.Printf("Created cache file %s", filename)
		}
		defer out.Close()
		writer := io.MultiWriter(w, out)
		nRead, err := io.Copy(writer, data.Body)
		if err != nil {
			log.Printf("Error writing data: %s", err)
			return
		}
		tentaSize.Add(float64(nRead))
		tentaFiles.Inc()
		if args.debug {
			log.Printf("Served / Cached file %s (%d)", filename, nRead)
		}
	} else {
		tentaHits.Inc()
		written, err := io.Copy(w, file)
		if err != nil {
			log.Printf("Error serving %s: %s", filename, err)
			return
		}
		log.Printf("Cached file found: %s (%d)", filename, written)
	}
}

func generateURL(r *http.Request) string {
	scheme := r.Header.Get("Scheme")
	if scheme == "" {
		scheme = "http"
	}
	return fmt.Sprintf("%s://%s%s", scheme, r.Host, r.URL)
}

func generateCacheFilename(url string, r *http.Request) string {
	cacheKey := url
	// Steam has too many CDN URLs, but they have a consistent URL
	// We can assume that if the user agent is Steam, the cache key is the same
	if r.UserAgent() == "Valve/Steam HTTP Client 1.0" {
		cacheKey = fmt.Sprintf("%s/%s", "steam", r.URL)
	}

	if args.debug {
		log.Printf("Generated cache key: %s", cacheKey)
	}

	return fmt.Sprintf("%d", fnv1a.HashString64(cacheKey))
}
