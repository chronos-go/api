package health

import (
	"encoding/json"
	"net/http"
)

type response struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func Get(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response{
		Status:  "ok",
		Message: "hello world",
	})
}
