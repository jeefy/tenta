package main

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/segmentio/fasthash/fnv1a"
)

func handleRequest(w http.ResponseWriter, r *http.Request) {
	url := generateURL(r)
	h1 := generateCacheFilename(url, r)
	filename := fmt.Sprintf("%s/%s", args.dataDir, h1)
	tentaReqeusts.Inc()

	if args.debug {
		log.Printf("Request for %s (%s)", filename, url)
	}

	if r.Header.Get("tenta-proxy") == "true" {
		w.WriteHeader(http.StatusLoopDetected)
		log.Printf("Sending Proxy loop detected, aborting")
		fmt.Fprintf(w, "Proxy loop detected, aborting")
		return
	}

	_, err := os.Stat(filename)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("Error checking file: %s", err)
			return
		}

		if args.debug {
			log.Printf("Cache file %s not found", filename)
		}
		tentaMisses.Inc()

		// Presumably, we're running custom DNS pointing to this
		// We need to ignore that and use a custom DNS resolver
		// Otherwise we will have a fun proxy loop situation
		var (
			dnsResolverIP        = "8.8.8.8:53" // Google DNS resolver.
			dnsResolverProto     = "udp"        // Protocol to use for the DNS resolver
			dnsResolverTimeoutMs = 5000         // Timeout (ms) for the DNS resolver (optional)
		)

		dialer := &net.Dialer{
			Resolver: &net.Resolver{
				PreferGo: true,
				Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
					d := net.Dialer{
						Timeout: time.Duration(dnsResolverTimeoutMs) * time.Millisecond,
					}
					return d.DialContext(ctx, dnsResolverProto, dnsResolverIP)
				},
			},
		}

		dialContext := func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialer.DialContext(ctx, network, addr)
		}

		http.DefaultTransport.(*http.Transport).DialContext = dialContext

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			log.Printf("Error creating request: %s", err)
			return
		}
		req.Header.Add("tenta-proxy", `true`)
		req.Header.Add("request-timestamp", fmt.Sprintf("%d", time.Now().Unix()))

		client := &http.Client{}

		data, err := client.Do(req)
		if err != nil {
			log.Printf("Error fetching data: %s", err)
		}
		defer data.Body.Close()
		if data.StatusCode != http.StatusOK {
			if data.StatusCode == http.StatusNotFound {
				w.WriteHeader(http.StatusNotFound)
				fmt.Fprintf(w, "404! Not Found")
				return
			}
			if data.StatusCode == http.StatusLoopDetected {
				w.WriteHeader(http.StatusLoopDetected)
				log.Printf("Received Proxy loop detected, aborting")
				fmt.Fprintf(w, "Proxy loop detected, aborting")
				return
			}
			w.WriteHeader(data.StatusCode)
			io.Copy(w, data.Body)
			return
		}

		file, err := os.Create(filename)
		if err != nil {
			// Still try to send the data to the client
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
		defer file.Close()
		nRead, err := file.ReadFrom(data.Body)
		if err != nil {
			log.Printf("Error writing data: %s", err)
			return
		}
		tentaSize.Add(float64(nRead))
		tentaFiles.Inc()
		if args.debug {
			log.Printf("Cached %s as %s (%d bytes)", url, filename, nRead)
		}
	} else {
		tentaHits.Inc()
	}
	fileBytes, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Printf("Error opening file: %s", err)
		return
	}

	written, err := w.Write(fileBytes)
	if err != nil {
		log.Printf("Error serving %s: %s", filename, err)
		return
	}
	log.Printf("Cached file found: %s (%d)", filename, written)
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
		cacheKey = fmt.Sprintf("%s%s", "steam", r.URL)
	}

	if args.debug {
		log.Printf("Generated cache key: %s", cacheKey)
	}

	return fmt.Sprintf("%d", fnv1a.HashString64(cacheKey))
}
