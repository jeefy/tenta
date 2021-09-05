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

func main() {
	// Create a mux for routing incoming requests
	myHandler := http.NewServeMux()

	// All URLs will be handled by this function
	myHandler.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		//log.Printf("%v", r.Header)
		log.Printf("Retrieving %s://%s%s", r.Header.Get("Scheme"), r.Host, r.URL)
		h1 := fnv1a.HashString64(fmt.Sprintf("%s%s", r.Host, r.URL))
		filename := fmt.Sprintf("data/%d", h1)

		if file, err := os.Open(filename); os.IsNotExist(err) {
			// path/to/whatever does not exist
			data, err := http.Get(fmt.Sprintf("%s://%s%s", r.Header.Get("Scheme"), r.Host, r.URL))
			if err != nil {
				w.WriteHeader(http.StatusNotFound)
				fmt.Fprintf(w, "404! Not found")
				return
			}
			defer data.Body.Close()
			out, err := os.Create(filename)
			if err != nil {
				log.Printf("Error creating file: %s", err)
				return
			}
			defer out.Close()
			writer := io.MultiWriter(w, out)
			io.Copy(writer, data.Body)
		} else {
			log.Printf("Cached file found: %s", filename)
			io.Copy(w, file)
		}
	})

	s := &http.Server{
		Addr:           ":8080",
		Handler:        myHandler,
		ReadTimeout:    60 * time.Second,
		WriteTimeout:   60 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	log.Fatal(s.ListenAndServe())
}
