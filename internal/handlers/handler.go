package handlers

import (
	"encoding/json"
	"github.com/dhruv15803/go-community-platform/internal/storage"
	"github.com/redis/go-redis/v9"
	"net/http"
)

type Handler struct {
	storage *storage.Storage
	rdb     *redis.Client
}

func NewHandler(storage *storage.Storage, rdb *redis.Client) *Handler {
	return &Handler{
		storage: storage,
		rdb:     rdb,
	}
}

func (h *Handler) HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	type Response struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	if err := writeJSON(w, Response{Success: true, Message: "health ok"}, http.StatusOK); err != nil {
		writeJSONError(w, "internal server error", http.StatusInternalServerError)
	}
}

func writeJSON(w http.ResponseWriter, data interface{}, status int) error {

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(data)
}

func writeJSONError(w http.ResponseWriter, message string, status int) error {

	type ErrorResponse struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}

	return writeJSON(w, ErrorResponse{Success: false, Message: message}, status)

}

func readJSON(r *http.Request, v interface{}) error {
	// v will be a pointer to a go struct to decode into
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(v)
}
