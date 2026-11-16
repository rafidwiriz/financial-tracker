package server

import (
	"encoding/json"
	"net/http"
)

// WriteJSON serializes v as the response body with the standard envelope
// (docs/api-design.md §Response envelope).
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// WriteData writes { "data": ... }.
func WriteData(w http.ResponseWriter, status int, data any) {
	WriteJSON(w, status, map[string]any{"data": data})
}

// WriteError writes the { "error": { code, message, details } } envelope.
func WriteError(w http.ResponseWriter, err *AppError) {
	WriteJSON(w, err.Status, map[string]any{"error": err})
}
