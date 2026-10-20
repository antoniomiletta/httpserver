# httpserver

Simple, dependency-free HTTP/1.1 server implemented from scratch over raw TCP socket.

- *[Notes](#notes)*
- *[Layout](#layout)*
- *[Quickstart](#quickstart)*
- *[API](#api)*


## Notes

### Bounds & Allocation

- **Line Size Limits**: Request lines and headers are capped by `maxLineSize` (10 KB). Overlong lines return `414 Request URI Too Long`.
- **Payload Ceiling**: Request bodies are capped at `maxBodySize` (1 MB). If `Content-Length` exceeds this limit, parsing aborts with `413 Request Entity Too Large`.
- **Bounded Streaming**: Request bodies are wrapped in `io.LimitReader`, preventing the client from streaming more bytes than declared.
- **Buffer Recycling**: On persistent connections, the response internal `bufio.Writer` and header map are cleared and reused, avoiding reallocations per request.

### Wire Protocol Conformance

- **Version**: Requests must specify `HTTP/1.1`. Other protocol versions are rejected with `505 HTTP Version Not Supported`.
- **Persistent Connections (Keep-Alive)**: Connections default to persistent unless `Connection: close` is specified.
- **Framing**:
  - Responses with known content lengths send an explicit `Content-Length` header.
  - Responses without a predefined length automatically stream using `Transfer-Encoding: chunked`, emitting chunk-size headers in hexadecimal, data frames, and terminating with `0\r\n\r\n`.
  - Status codes that disallow bodies (1xx, `204 No Content`, `304 Not Modified`) have `Content-Length` stripped automatically and omit body transmission.
  - Responses with empty bodies default to `Content-Length: 0`.
  - Responses automatically inject the formatted `Date` header if not supplied by the application.

### Panic Recovery

- **Goroutine Isolation**: Application handlers are executed in an isolated goroutine managed by `safeHandle`.
- **Panic Boundary**: Any handler panic is intercepted via `recover()`. The stack trace is captured and logged with `runtime/debug.Stack()`, and the server sends a `500 Internal Server Error` to the client without terminating the process.
- **Timeout Isolation**: If a handler exceeds `handlerTimeout`, a raw `504 Gateway Timeout` is emitted directly to the socket, and the connection is closed.
- **State-Aware Teardown**: If headers have already been written to the wire before a failure or panic occurs, the connection is closed immediately to prevent sending malformed frames.

---

## Layout

```
.
├── cmd/
│   └── demo/
│       └── main.go         # Example server application and routing setup
├── httpd/
│   ├── connection.go       # HTTP/1.1 stream parser (request line, headers, body)
│   ├── errors.go           # Protocol error definitions and status code mapping
│   ├── header.go           # Header collection with MIME key canonicalization
│   ├── limits.go           # Buffer size limits and timeout constants
│   ├── request.go          # Request struct, query extraction, context propagation
│   ├── response.go         # ResponseWriter implementation, chunked encoding, framing
│   ├── router.go           # Method/path routing multiplexer
│   ├── server.go           # TCP listener loop, concurrency semaphore, panic recovery
│   └── status.go           # Standard HTTP status code constants and status text
├── ...
```

---

## Quickstart

### Running the Demo Server

```bash
# go run cmd/demo/main.go (listens on :8080)
make demo
```

### Verifying Endpoints

```bash
# GET endpoint
curl -v http://localhost:8080

# POST endpoint with payload echo
curl -v -X POST http://localhost:8080/echo -d 'Echo!'
```

---

## API

### Creating Routes and Starting the Server

```go
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
```
