package httpd

import (
	"context"
	"io"
	"net/url"
)

// TODO: split url target and params
type Request struct {
	Method  string
	Target  string
	URL     *url.URL
	Query   url.Values
	Version string
	Header  Header
	Body    io.ReadCloser

	ctx context.Context
}

func (r *Request) Context() context.Context {
	if r.ctx != nil {
		return r.ctx
	}
	return context.Background()
}
