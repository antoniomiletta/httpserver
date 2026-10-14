package httpd

import (
	"bufio"
	"fmt"
	"net"
	"time"
)

type ResponseWriter interface {
	Write([]byte) (int, error)
	WriteHeader(statusCode int)
	Header() Header
}

type response struct {
	conn        net.Conn
	buf         *bufio.Writer
	header      Header
	statusCode  int
	wroteHeader bool
	sentHeader  bool
	chunked     bool
}

func newResponse(conn net.Conn) *response {
	return &response{
		conn:   conn,
		buf:    bufio.NewWriter(conn),
		header: make(Header),
	}
}

const TimeFormat = "Mon, 02 Jan 2006 15:04:05 GMT"

func (r *response) Write(b []byte) (int, error) {
	if len(b) == 0 {
		return 0, nil
	}

	if !r.sentHeader {
		if !r.wroteHeader {
			r.WriteHeader(StatusOK)
		}
		if err := r.sendHeader(); err != nil {
			return 0, err
		}
	}

	if r.chunked {
		fmt.Fprintf(r.buf, "%x\r\n", len(b))
		r.buf.Write(b)
		fmt.Fprintf(r.buf, "\r\n")
		return len(b), nil
	}

	return r.buf.Write(b)
}

func (r *response) WriteHeader(statusCode int) {
	if r.wroteHeader {
		return
	}

	r.wroteHeader = true
	r.statusCode = statusCode
}

func (r *response) Header() Header {
	return r.header
}

func (r *response) finish() error {
	if !r.sentHeader {
		if !r.wroteHeader {
			r.WriteHeader(StatusOK)
		}

		if bodyAllowedForStatus(r.statusCode) && r.header.Get("Content-Length") == "" {
			r.header.Set("Content-Length", "0")
		}

		if err := r.sendHeader(); err != nil {
			return err
		}
	}

	if r.chunked {
		fmt.Fprintf(r.buf, "0\r\n\r\n")
	}

	return r.buf.Flush()
}

func (r *response) sendHeader() error {
	r.conn.SetWriteDeadline(time.Now().Add(writeTimeout))

	fmt.Fprintf(r.buf, "HTTP/1.1 %d %s\r\n", r.statusCode, StatusText(r.statusCode))

	// default
	if r.header.Get("Date") == "" {
		fmt.Fprintf(r.buf, "Date: %s\r\n", time.Now().UTC().Format(TimeFormat))
	}

	// body
	if bodyAllowedForStatus(r.statusCode) {
		if r.header.Get("Content-Length") == "" {
			fmt.Fprintf(r.buf, "Transfer-Encoding: chunked\r\n")
			r.chunked = true
		}
	} else {
		r.header.Del("Content-Length")
	}

	// manually set
	for k, vs := range r.header {
		for _, v := range vs {
			fmt.Fprintf(r.buf, "%s: %s\r\n", k, v)
		}
	}

	fmt.Fprintf(r.buf, "\r\n")
	r.sentHeader = true
	return nil
}

func (r *response) reset() {
	r.buf.Reset(r.conn)
	clear(r.header)
	r.statusCode = 0
	r.wroteHeader = false
	r.sentHeader = false
	r.chunked = false
}

func bodyAllowedForStatus(status int) bool {
	switch {
	case status >= 100 && status <= 199:
		return false
	case status == StatusNoContent, status == StatusNotModified:
		return false
	default:
		return true
	}
}
