package httpd

import "net/textproto"

type Header map[string][]string

func (h Header) Add(key, value string) {
	key = CanonicalHeaderKey(key)
	h[key] = append(h[key], value)
}

func (h Header) Set(key, value string) {
	key = CanonicalHeaderKey(key)
	h[key] = []string{value}
}

func (h Header) Get(key string) string {
	key = CanonicalHeaderKey(key)

	values := h[key]
	if len(values) == 0 {
		return ""
	}

	return values[0]
}

func (h Header) Values(key string) []string {
	key = CanonicalHeaderKey(key)

	values, ok := h[key]
	if !ok {
		return nil
	}

	return values
}

func (h Header) Del(key string) {
	key = CanonicalHeaderKey(key)
	delete(h, key)
}

func CanonicalHeaderKey(key string) string {
	return textproto.CanonicalMIMEHeaderKey(key)
}
