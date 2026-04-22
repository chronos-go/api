package client

import (
	"encoding/json"
	"net/http"
	"strings"
)

type Client struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Phone string `json:"phone"`
}

type response struct {
	Status  string  `json:"status"`
	Message string  `json:"message"`
	Data    *Client `json:"data,omitempty"`
}

func Create(w http.ResponseWriter, r *http.Request) {
	var c Client
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		writeJSON(w, http.StatusBadRequest, response{
			Status:  "error",
			Message: "invalid request body",
		})
		return
	}

	if err := validate(c); err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, response{
			Status:  "error",
			Message: err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusCreated, response{
		Status:  "ok",
		Message: "client created",
		Data:    &c,
	})
}

func validate(c Client) error {
	var missing []string

	if strings.TrimSpace(c.Name) == "" {
		missing = append(missing, "name")
	}
	if strings.TrimSpace(c.Email) == "" {
		missing = append(missing, "email")
	}
	if !strings.Contains(c.Email, "@") && c.Email != "" {
		return &validationError{"email format is invalid"}
	}
	if strings.TrimSpace(c.Phone) == "" {
		missing = append(missing, "phone")
	}

	if len(missing) > 0 {
		return &validationError{"missing required fields: " + strings.Join(missing, ", ")}
	}

	return nil
}

type validationError struct{ msg string }

func (e *validationError) Error() string { return e.msg }

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
