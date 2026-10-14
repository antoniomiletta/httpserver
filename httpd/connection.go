package httpd

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type connection struct {
	client net.Conn
	stream *bufio.Reader
}

type emptyBody struct{}

func (emptyBody) Read([]byte) (int, error) {
	return 0, io.EOF
}

func (emptyBody) Close() error {
	return nil
}

func newConnection(conn net.Conn) *connection {
	return &connection{
		client: conn,
		stream: bufio.NewReaderSize(conn, maxLineSize),
	}
}

func (c *connection) parseRequest() (*Request, error) {
	c.client.SetReadDeadline(time.Now().Add(idleTimeout))
	line, err := c.readLine()
	if err != nil {
		return nil, fmt.Errorf("read line: %w", err)
	}

	parts := strings.Split(line, " ")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid request header")
	}

	method := parts[0]
	rawURI := parts[1]
	version := strings.TrimSuffix(parts[2], "\r\n")

	if version != "HTTP/1.1" {
		return nil, errHTTPVersionNotSupported
	}

	parsedURL, err := url.ParseRequestURI(rawURI)
	if err != nil {
		return nil, fmt.Errorf("invalid request uri: %v", err)
	}

	target := parsedURL.Path
	query := parsedURL.Query()

	c.client.SetReadDeadline(time.Now().Add(readHeaderTimeout))
	headers, err := c.readHeaders()
	if err != nil {
		return nil, fmt.Errorf("read headers: %w", err)
	}

	if headers.Get("Host") == "" {
		return nil, fmt.Errorf("read headers: required header: 'Host' is missing")
	}

	if headers.Get("Transfer-Encoding") != "" {
		return nil, errNotImplemented
	}

	var body io.ReadCloser
	cl := headers.Get("Content-Length")

	if cl != "" && cl != "0" {
		size, err := strconv.ParseInt(cl, 10, 63)
		if err != nil {
			return nil, fmt.Errorf("parse content-length: %w", err)
		}

		if size > maxBodySize {
			return nil, errBodyTooLong
		}

		c.client.SetReadDeadline(time.Now().Add(readBodyTimeout))
		reader, err := c.readBody(size)
		if err != nil {
			return nil, fmt.Errorf("read body: %w", err)
		}
		body = reader
	} else {
		body = emptyBody{}
	}

	return &Request{
		Method:  method,
		Target:  target,
		Query:   query,
		Version: version,
		Header:  headers,
		Body:    body,
	}, nil
}

func (c *connection) readHeaders() (Header, error) {
	headers := make(Header)

	for {
		line, err := c.readLine()
		if err != nil {
			return nil, err
		}

		line = strings.TrimRight(line, "\r\n")

		if line == "" {
			return headers, nil
		}

		key, val, ok := strings.Cut(line, ":")
		if !ok {
			return nil, fmt.Errorf("read headers: invalid header format")
		}

		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)

		headers.Add(key, val)
	}
}

func (c *connection) readBody(size int64) (io.ReadCloser, error) {
	return io.NopCloser(io.LimitReader(c.stream, size)), nil
}

func (c *connection) readLine() (string, error) {
	line, err := c.stream.ReadSlice('\n')
	if err != nil {
		if err == bufio.ErrBufferFull {
			return "", errLineTooLong
		}
		return "", err
	}

	return string(bytes.TrimRight(line, "\r\n")), nil
}
