package errs

import (
	"encoding/json"
	"errors"
	"net/http"
)

var (
	ErrRateLimitExceeded = errors.New("rate limit exceeded")
)

// JSONError отправляет HTTP-ответ с ошибкой в формате JSON.
// Устанавливает правильные заголовки и статус код.
func JSONError(w http.ResponseWriter, err ErrorResponse, code int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(err)
}

type ErrorResponse struct {
	Error string `json:"error"`
}
