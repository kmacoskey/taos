package middleware

import (
	"net/http"
)

func Adapt(h http.Handler, adapters ...Adapter) http.Handler {
	for _, adapter := range adapters {
		h = adapter(h)
	}
	return h
}

type Adapter func(http.Handler) http.Handler
