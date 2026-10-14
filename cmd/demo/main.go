package main

import (
	"io"
	"log"
	"strconv"

	"github.com/antoniomiletta/gohttp/httpd"
)

const port = "8080"

func main() {
	router := httpd.NewRouter()

	router.Get("/", getHandler)
	router.Post("/demo/echo", echoHandler)

	s := httpd.NewServer(port, router)

	if err := s.Serve(); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}

func getHandler(w httpd.ResponseWriter, r *httpd.Request) {
	w.Write([]byte("Hello"))
}

func echoHandler(w httpd.ResponseWriter, r *httpd.Request) {
	defer r.Body.Close()

	b, err := io.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(httpd.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Length", strconv.Itoa(len(b)))
	w.Write(b)
}
