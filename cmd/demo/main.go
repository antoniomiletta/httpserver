package main

import (
	"io"
	"log"
	"strconv"

	"github.com/antoniomiletta/httpserver/httpd"
)

func main() {
	router := httpd.NewRouter()

	router.Get("/", func(w httpd.ResponseWriter, r *httpd.Request) {
		w.Write([]byte("Hello\n"))
	})

	router.Post("/echo", func(w httpd.ResponseWriter, r *httpd.Request) {
		defer r.Body.Close()

		payload, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(httpd.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Content-Length", strconv.Itoa(len(payload)))
		w.Write(payload)
	})

	server := httpd.NewServer("8080", router)
	if err := server.Serve(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
