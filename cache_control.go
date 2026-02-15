package main

import (
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// CacheControl holds parsed Cache-Control directives
type CacheControl struct {
	NoStore        bool
	NoCache        bool
	MaxAge         int // seconds, -1 means not specified
	Public         bool
	Private        bool
	MustRevalidate bool
}

// ParseCacheControl parses Cache-Control header
func ParseCacheControl(header string) CacheControl {
	cc := CacheControl{MaxAge: -1}
	
	if header == "" {
		return cc
	}

	parts := strings.Split(header, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "max-age=") {
			if age, err := strconv.Atoi(strings.TrimPrefix(part, "max-age=")); err == nil {
				cc.MaxAge = age
			}
		} else if part == "no-store" {
			cc.NoStore = true
		} else if part == "no-cache" {
			cc.NoCache = true
		} else if part == "public" {
			cc.Public = true
		} else if part == "private" {
			cc.Private = true
		} else if part == "must-revalidate" {
			cc.MustRevalidate = true
		}
	}
	
	return cc
}

// shouldCacheResponse determines if response should be cached based on headers
func shouldCacheResponse(resp *http.Response) bool {
	// Don't cache if Cache-Control: no-store is present
	cacheControl := ParseCacheControl(resp.Header.Get("Cache-Control"))
	if cacheControl.NoStore {
		if args.debug {
			log.Printf("Skipping cache: no-store directive present")
		}
		return false
	}

	// Don't cache non-200 responses
	if resp.StatusCode != http.StatusOK {
		if args.debug {
			log.Printf("Skipping cache: status code %d", resp.StatusCode)
		}
		return false
	}

	// Don't cache if response has no content length
	if resp.ContentLength <= 0 {
		if args.debug {
			log.Printf("Skipping cache: no content or invalid content-length")
		}
		return false
	}

	return true
}

// canServeFromCache checks if cached response can still be served
func canServeFromCache(cachedTime time.Time, cacheControl CacheControl) bool {
	// If max-age is specified, check if still valid
	if cacheControl.MaxAge >= 0 {
		if time.Since(cachedTime) > time.Duration(cacheControl.MaxAge)*time.Second {
			if args.debug {
				log.Printf("Cached response expired (max-age=%d)", cacheControl.MaxAge)
			}
			return false
		}
	}

	return true
}

// logRequest logs HTTP request/response with timing and size information
func logRequest(method string, path string, statusCode int, duration time.Duration, size int64, source string) {
	if args.debug {
		log.Printf("[%s] %s %s %d - %dms - %d bytes", 
			source, method, path, statusCode, duration.Milliseconds(), size)
	}
}
