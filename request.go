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
	log.Printf("Retrieving %s://%s%s", r.Header.Get("Scheme"), r.Host, r.URL)
	h1 := fnv1a.HashString64(fmt.Sprintf("%s%s", r.Host, r.URL))
	filename := fmt.Sprintf("%s/%d", args.dataDir, h1)
	tentaReqeusts.Inc()
	if file, err := os.Open(filename); os.IsNotExist(err) {
		if args.debug {
			log.Printf("Cache file %s not found", filename)
		}
		tentaMisses.Inc()
		scheme := r.Header.Get("Scheme")
		if scheme == "" {
			scheme = "http"
		}
		url := fmt.Sprintf("%s://%s%s", scheme, r.Host, r.URL)
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
