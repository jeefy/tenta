package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/segmentio/fasthash/fnv1a"
)

func handleRequest(w http.ResponseWriter, r *http.Request) {
	//log.Printf("%v", r.Header)
	scheme := r.Header.Get("Scheme")
	if scheme == "" {
		scheme = "http"
	}
	url := fmt.Sprintf("%s://%s%s", scheme, r.Host, r.URL)
	cacheKey := url
	log.Printf("Retrieving %s", url)

	// Steam has too many CDN URLs, but they have a consistent URL
	// We can assume that if the user agent is Steam, the cache key is the same
	if r.UserAgent() == "Valve/Steam HTTP Client 1.0" {
		cacheKey = fmt.Sprintf("%s/%s", "steam", r.URL)
	}

	h1 := fnv1a.HashString64(cacheKey)
	filename := fmt.Sprintf("%s/%d", args.dataDir, h1)
	tentaReqeusts.Inc()
	if file, err := os.Open(filename); os.IsNotExist(err) {
		if args.debug {
			log.Printf("Cache file %s not found", filename)
		}
		tentaMisses.Inc()
		data, err := http.Get(url)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, "404! Not found")
			return
		}
		defer data.Body.Close()
		out, err := os.Create(filename)
		if err != nil {
			log.Printf("Error creating file: %s", err)
			io.Copy(w, data.Body)
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
		}
		tentaSize.Add(float64(nRead))
		tentaFiles.Inc()
		if args.debug {
			log.Printf("Served / Cached file %s (%d)", filename, nRead)
		}
	} else {
		tentaHits.Inc()
		log.Printf("Cached file found: %s", filename)
		io.Copy(w, file)
	}
}
