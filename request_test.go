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
