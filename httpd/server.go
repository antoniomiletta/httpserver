package httpd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"runtime/debug"
	"strings"
	"sync"
	"time"
)

type Server struct {
	Addr   string
	Router *Router

	mu sync.Mutex
	ln net.Listener
}

func NewServer(port string, router *Router) *Server {
	return &Server{
		Addr:   ":" + port,
		Router: router,
	}
}

const maxConcurrent = 100

func (s *Server) Serve() error {
	ln, err := net.Listen("tcp", s.Addr)
	if err != nil {
		return fmt.Errorf("failed to listen to tcp: %v", err)
	}
	s.mu.Lock()
	s.ln = ln
	s.mu.Unlock()

	var wg sync.WaitGroup
	defer wg.Wait()

	sem := make(chan struct{}, maxConcurrent)
	var delay time.Duration

	for {
		conn, err := ln.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return err
			}

			// exponential backoff for persistent errors like EMFILE.
			if delay == 0 {
				delay = time.Millisecond * 5
			} else if delay *= 2; delay > time.Second {
				delay = time.Second
			}
			log.Printf("accept connection: %v; retrying in: %v", err, delay)
			time.Sleep(delay)

			continue
		}

		wg.Go(func() {
			sem <- struct{}{}
			defer func() { <-sem }()
			s.handleConn(conn)
		})

		delay = 0
	}
}

func (s *Server) handleConn(conn net.Conn) {
	defer conn.Close()

	connection := newConnection(conn)
	res := newResponse(conn)

	for {
		req, err := connection.parseRequest()
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, os.ErrDeadlineExceeded) {
				return
			}
			s.writeError(res, mapParseErr(err))
			log.Printf("parse request: %v", err)
			return
		}

		keepAlive := !strings.EqualFold(req.Header.Get("Connection"), "close")

		if !keepAlive {
			res.Header().Set("Connection", "close")
		}

		handler := s.Router.resolve(req.Method + " " + req.Target)

		if err := s.safeHandle(handler, res, req); err != nil {
			if res.sentHeader {
				conn.Close()
				return
			}
			if errors.Is(err, errHandlerTimeout) {
				writeRawTimeout(conn)
				conn.Close()
				return
			}

			res.reset()
			s.writeError(res, StatusInternalServerError)
			return
		}

		if err := res.finish(); err != nil {
			log.Printf("finish response: %v", err)
			return
		}

		res.reset()
		if req.Body != nil {
			if _, err := io.Copy(io.Discard, req.Body); err != nil {
				return
			}
			req.Body.Close()
		}

		if !keepAlive {
			return
		}
	}
}

func (s *Server) safeHandle(h HandlerFunc, w ResponseWriter, r *Request) error {
	ctx, cancel := context.WithTimeout(context.Background(), handlerTimeout)
	defer cancel()
	r.ctx = ctx

	errCh := make(chan error, 1)

	go func() {
		defer func() {
			if p := recover(); p != nil {
				errCh <- errHandlerPanic
				log.Printf("panic handling: %s %s: %v\n%s", r.Method, r.Target, p, debug.Stack())
			}
		}()

		h(w, r)
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		return errHandlerTimeout
	case e := <-errCh:
		return e
	}
}

func (s *Server) writeError(res *response, status int) {
	res.WriteHeader(status)
	res.Header().Set("Connection", "close")

	if err := res.finish(); err != nil {
		log.Printf("finish response: %v", err)
	}
}

func (s *Server) Shutdown() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ln == nil {
		return nil
	}
	return s.ln.Close()
}

func writeRawTimeout(conn net.Conn) {
	const timeoutResponse = "HTTP/1.1 504 Gateway Timeout\r\n" + "Content-Type: text/plain\r\n" +
		"Connection: close\r\n" +
		"Content-Length: 15\r\n" +
		"\r\n" +
		"Gateway Timeout"

	conn.SetWriteDeadline(time.Now().Add(writeTimeout))
	fmt.Fprint(conn, timeoutResponse)

}
