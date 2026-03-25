package middleware

import (
	"net/http"
)

// APIKey returns a middleware that rejects requests missing the expected API key
// in the X-Api-Key header.
func APIKey(key string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("X-Api-Key") != key {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
