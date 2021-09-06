package main

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

func StartHTTP() {
	port := fmt.Sprintf(":%d", args.httpPort)
	log.Printf("Starting HTTP server on %s", port)
	// Create a mux for routing incoming requests
	myHandler := http.NewServeMux()

	// All URLs will be handled by this function
	myHandler.HandleFunc("/", handleRequest)

	s := &http.Server{
		Addr:           port,
		Handler:        myHandler,
		ReadTimeout:    60 * time.Second,
		WriteTimeout:   60 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	log.Fatal(s.ListenAndServe())
}
