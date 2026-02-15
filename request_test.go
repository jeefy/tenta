package main

import (
	"net/http"
	"net/url"
	"testing"
)

func TestGeneratedURL(t *testing.T) {
	tests := []struct {
		name     string
		request  *http.Request
		expected string
	}{
		{
			name:     "blank request",
			request:  &http.Request{},
			expected: "http://<nil>",
		},
		{
			name: "no schema",
			request: &http.Request{
				URL: &url.URL{
					Path: "/",
				},
				Host: "google.com",
			},
			expected: "http://google.com/",
		},
		{
			name: "https",
			request: &http.Request{
				Header: http.Header{
					"Scheme": []string{"https"},
				},
				URL: &url.URL{
					Path: "/",
				},
				Host: "google.com",
			},
			expected: "https://google.com/",
		},
	}

	for _, test := range tests {
		if url := generateURL(test.request); url != test.expected {
			t.Errorf("%s: expected URL `%s` doesn't match `%s`", test.name, test.expected, url)
		}
	}
}

func TestGeneratedCacheFilename(t *testing.T) {
	tests := []struct {
		name     string
		request  *http.Request
		expected string
	}{
		{
			name:     "blank request",
			request:  &http.Request{},
			expected: "12898796920235164326",
		},
		{
			name: "generic URL",
			request: &http.Request{
				URL: &url.URL{
					Path: "/",
				},
				Host: "google.com",
			},
			expected: "3495272084109939400",
		},
		{
			name: "steam client",
			request: &http.Request{
				Header: http.Header{
					"User-Agent": []string{"Valve/Steam HTTP Client 1.0"},
				},
				URL: &url.URL{
					Path: "/abc123",
				},
				Host: "google.com",
			},
			expected: "13712127455315645540",
		},
	}

	for _, test := range tests {
		url := generateURL(test.request)
		if filename := generateCacheFilename(url, test.request); filename != test.expected {
			t.Errorf("%s: expected filename `%s` doesn't match `%s`", test.name, test.expected, filename)
		}
	}
}

// TestMetrics verifies that metric counters are properly incremented
func TestMetrics(t *testing.T) {
	// Store initial values
	initialReqs := getRequestsCount()
	initialHits := getHitsCount()
	initialMisses := getMissesCount()
	initialErrors := getErrorsCount()

	// Increment counters
	incRequests()
	incHits()
	incMisses()
	incErrors()

	// Verify increments
	if getRequestsCount() != initialReqs+1 {
		t.Errorf("incRequests() failed: expected %d, got %d", initialReqs+1, getRequestsCount())
	}
	if getHitsCount() != initialHits+1 {
		t.Errorf("incHits() failed: expected %d, got %d", initialHits+1, getHitsCount())
	}
	if getMissesCount() != initialMisses+1 {
		t.Errorf("incMisses() failed: expected %d, got %d", initialMisses+1, getMissesCount())
	}
	if getErrorsCount() != initialErrors+1 {
		t.Errorf("incErrors() failed: expected %d, got %d", initialErrors+1, getErrorsCount())
	}
}

// TestSizeMetrics verifies size tracking
func TestSizeMetrics(t *testing.T) {
	initialSize := getSizeCount()
	initialFiles := getFilesCount()

	// Add size and file
	addSize(1024)
	incFiles()

	if getSizeCount() != initialSize+1024 {
		t.Errorf("addSize() failed: expected %d, got %d", initialSize+1024, getSizeCount())
	}
	if getFilesCount() != initialFiles+1 {
		t.Errorf("incFiles() failed: expected %d, got %d", initialFiles+1, getFilesCount())
	}

	// Subtract size and file
	subSize(1024)
	decFiles()

	if getSizeCount() != initialSize {
		t.Errorf("subSize() failed: expected %d, got %d", initialSize, getSizeCount())
	}
	if getFilesCount() != initialFiles {
		t.Errorf("decFiles() failed: expected %d, got %d", initialFiles, getFilesCount())
	}
}
