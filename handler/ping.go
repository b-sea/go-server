package handler

import (
	"encoding/json"
	"net/http"
)

func Ping() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode("pong")
	}
}
