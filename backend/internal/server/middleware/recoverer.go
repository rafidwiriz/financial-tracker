package middleware

import (
	"log/slog"
	"net/http"
)

// Recoverer converts panics into a 500 error envelope and logs the stack.
// Kept envelope-compatible with docs/api-design.md without importing server
// (avoids an import cycle).
func Recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("panic recovered", "value", rec, "request_id", RequestIDFrom(r.Context()))
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"error":{"code":"INTERNAL_ERROR","message":"Internal server error"}}`))
			}
		}()
		next.ServeHTTP(w, r)
	})
}