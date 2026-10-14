package httpd

import "errors"

var (
	errLineTooLong = errors.New("maximum line size exceeded.")
	errBodyTooLong = errors.New("maximum body size exceeded.")

	errHTTPVersionNotSupported = errors.New("specified http version not supported.")
	errNotImplemented          = errors.New("not implemented.")

	errHandlerTimeout = errors.New("handler timeout.")
	errHandlerPanic   = errors.New("handler panicked.")
)

func mapParseErr(err error) int {
	switch {
	case errors.Is(err, errLineTooLong):
		return StatusRequestURITooLong
	case errors.Is(err, errBodyTooLong):
		return StatusRequestEntityTooLarge
	case errors.Is(err, errHTTPVersionNotSupported):
		return StatusHTTPVersionNotSupported
	case errors.Is(err, errNotImplemented):
		return StatusNotImplemented
	default:
		return StatusBadRequest
	}
}
