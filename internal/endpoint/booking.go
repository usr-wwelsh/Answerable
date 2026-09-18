package endpoint

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/usr-wwelsh/answerable/internal/booking"
	"github.com/usr-wwelsh/answerable/internal/webhook"
)

const maxFieldLen = 500
const maxBookBodyBytes = 4096

type bookRequestBody struct {
	Name    string `json:"name"`
	Contact string `json:"contact"`
	Need    string `json:"need"`
}

// bookingValidationError marks an intake error as caused by bad caller
// input (safe to relay verbatim), as opposed to a store failure (which
// isn't, since it may carry internal detail).
type bookingValidationError struct{ msg string }

func (e *bookingValidationError) Error() string { return e.msg }

// bookIntake runs the intake-request flow shared by the HTTP /book route
// and the MCP book_intake tool: validate, enqueue, and notify the webhook
// if one is configured. baseURL is used to build the confirm/deny links
// sent to the webhook.
func (s *Server) bookIntake(baseURL, name, contact, need string) (booking.Request, string, error) {
	name = strings.TrimSpace(name)
	contact = strings.TrimSpace(contact)
	need = strings.TrimSpace(need)

	if name == "" || need == "" {
		return booking.Request{}, "", &bookingValidationError{"name and need are required"}
	}
	if len(name) > maxFieldLen || len(contact) > maxFieldLen || len(need) > maxFieldLen {
		return booking.Request{}, "", &bookingValidationError{"field too long"}
	}

	req, err := s.store.Enqueue(name, contact, need)
	if err != nil {
		return booking.Request{}, "", err
	}

	message := "This request has been queued for provider review. If this is urgent, contacting them by phone directly may be faster."

	if webhookURL := s.currentWebhook(); webhookURL != "" {
		confirmURL := baseURL + "/confirm?token=" + req.Token
		denyURL := baseURL + "/deny?token=" + req.Token
		notifyMsg := fmt.Sprintf(
			"New intake request from %s (%s): %s\nConfirm: %s\nDeny: %s",
			req.Name, req.Contact, req.Need, confirmURL, denyURL,
		)
		if err := webhook.Notify(webhookURL, notifyMsg); err == nil {
			message = "Your request has been sent to the provider. They'll confirm shortly."
		}
	}

	return req, message, nil
}

func (s *Server) handleBook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxBookBodyBytes)

	var body bookRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	req, message, err := s.bookIntake(baseURL(r), body.Name, body.Contact, body.Need)
	if err != nil {
		var verr *bookingValidationError
		if errors.As(err, &verr) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, "failed to queue request", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{
		"status":      "queued",
		"message":     message,
		"statusToken": req.StatusToken,
	})
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "missing token", http.StatusBadRequest)
		return
	}

	req, err := s.store.Status(token)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": string(req.Status)})
}

func (s *Server) handleConfirm(w http.ResponseWriter, r *http.Request) {
	s.resolveToken(w, r, s.store.Confirm, "Request confirmed.")
}

func (s *Server) handleDeny(w http.ResponseWriter, r *http.Request) {
	s.resolveToken(w, r, s.store.Deny, "Request denied.")
}

func (s *Server) resolveToken(w http.ResponseWriter, r *http.Request, resolve func(string) (booking.Request, error), okMessage string) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "missing token", http.StatusBadRequest)
		return
	}

	if _, err := resolve(token); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Write([]byte(okMessage))
}
