package httpd

import "time"

const (
	maxLineSize = 1024 * 10
	maxBodySize = 1024 * 1024

	idleTimeout       = time.Second * 5
	readHeaderTimeout = time.Second * 10
	readBodyTimeout   = time.Second * 60
	writeTimeout      = time.Second * 30

	handlerTimeout = time.Second * 60
)
