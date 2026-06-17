package response

import (
	"encoding/json"
	"net/http"
)

// JSON sends a JSON response to the client
func JSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}

// Error sends an error response to the client
func Error(w http.ResponseWriter, status int, msg string) {
	JSON(w, status, map[string]string{"error": msg})
}

// Success sends a success response (optional)
func Success(w http.ResponseWriter, status int, data any) {
	JSON(w, status, map[string]any{"data": data})
}
