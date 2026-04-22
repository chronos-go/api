package service

import (
	"encoding/json"
	"net/http"
	"strings"
)

// Service representa um serviço do catálogo de um profissional.
// DurationMinutes é armazenado em minutos (inteiro) para ser consumido
// diretamente pela engine de agendamento nas próximas Sprints.
type Service struct {
	Name            string `json:"name"`
	Description     string `json:"description"`
	PriceCents      int    `json:"price_cents"`
	DurationMinutes int    `json:"duration_minutes"`
	ProviderID      string `json:"provider_id"`
}

type response struct {
	Status  string   `json:"status"`
	Message string   `json:"message"`
	Data    *Service `json:"data,omitempty"`
}

func Create(w http.ResponseWriter, r *http.Request) {
	var s Service
	if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
		writeJSON(w, http.StatusBadRequest, response{
			Status:  "error",
			Message: "invalid request body",
		})
		return
	}

	if err := validate(s); err != nil {
		writeJSON(w, http.StatusUnprocessableEntity, response{
			Status:  "error",
			Message: err.Error(),
		})
		return
	}

	writeJSON(w, http.StatusCreated, response{
		Status:  "ok",
		Message: "service created",
		Data:    &s,
	})
}

func validate(s Service) error {
	var errs []string

	if strings.TrimSpace(s.Name) == "" {
		errs = append(errs, "name")
	}
	if strings.TrimSpace(s.ProviderID) == "" {
		errs = append(errs, "provider_id")
	}
	if s.DurationMinutes <= 0 {
		errs = append(errs, "duration_minutes must be a positive integer")
	}
	if s.PriceCents < 0 {
		errs = append(errs, "price_cents cannot be negative")
	}

	if len(errs) > 0 {
		return &validationError{"validation failed: " + strings.Join(errs, "; ")}
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
