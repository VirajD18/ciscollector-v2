package main

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

func (a *App) requireCollectorToken(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		expected := strings.TrimSpace(a.ServerConfig.Token)
		if expected == "" {
			http.Error(w, "collector token not configured on server", http.StatusServiceUnavailable)
			return
		}
		got := bearerToken(r)
		if got == "" {
			http.Error(w, "missing authorization", http.StatusUnauthorized)
			return
		}
		if subtle.ConstantTimeCompare([]byte(got), []byte(expected)) != 1 {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

func bearerToken(r *http.Request) string {
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		return strings.TrimSpace(auth[7:])
	}
	return ""
}
