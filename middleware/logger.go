package middleware

import (
	"fmt"
	"net/http"
)

func Logging() Adapter {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Println("before")
			defer fmt.Println("after")
			h.ServeHTTP(w, r)
		})
	}
}
