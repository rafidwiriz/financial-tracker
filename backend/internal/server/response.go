package server

import (
	"encoding/json"
	"net/http"
)

// writeJSON serializes v as the response body with the standard envelope
// (docs/api-design.md §Response envelope).
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeData writes { "data": ... }.
func writeData(w http.ResponseWriter, status int, data any) {
	writeJSON(w, status, map[string]any{"data": data})
}

// writeError writes the { "error": { code, message, details } } envelope.
func writeError(w http.ResponseWriter, err *AppError) {
	writeJSON(w, err.Status, map[string]any{"error": err})
}