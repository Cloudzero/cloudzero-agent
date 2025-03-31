package remotewrite

import (
	"log/slog"
	"net/http"
)

func authMiddleware(next http.Handler, apiKey string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get the Authorization header
		authHeader := r.Header.Get("Authorization")

		// Check if the header is present and matches the expected value
		if authHeader == "" {
			slog.Default().Info("Auth header not passed")
			writeAPIResponse(w, http.StatusUnauthorized, "Missing Authorization header")
			return
		}

		if authHeader != apiKey {
			slog.Default().Info("Invalid api key")
			writeAPIResponse(w, http.StatusForbidden, "Invalid API key")
			return
		}

		// If authentication is successful, call the next handler
		next.ServeHTTP(w, r)
	})
}
