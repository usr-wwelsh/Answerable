package endpoint

import (
	"encoding/json"
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

	body.Name = strings.TrimSpace(body.Name)
	body.Contact = strings.TrimSpace(body.Contact)
	body.Need = strings.TrimSpace(body.Need)

	if body.Name == "" || body.Need == "" {
		http.Error(w, "name and need are required", http.StatusBadRequest)
		return
	}
	if len(body.Name) > maxFieldLen || len(body.Contact) > maxFieldLen || len(body.Need) > maxFieldLen {
		http.Error(w, "field too long", http.StatusBadRequest)
		return
	}

	req, err := s.store.Enqueue(body.Name, body.Contact, body.Need)
	if err != nil {
		http.Error(w, "failed to queue request", http.StatusInternalServerError)
		return
	}

	message := "This request has been queued for provider review. If this is urgent, contacting them by phone directly may be faster."

	if s.webhookURL != "" {
		confirmURL := baseURL(r) + "/confirm?token=" + req.Token
		denyURL := baseURL(r) + "/deny?token=" + req.Token
		notifyMsg := fmt.Sprintf(
			"New intake request from %s (%s): %s\nConfirm: %s\nDeny: %s",
			req.Name, req.Contact, req.Need, confirmURL, denyURL,
		)
		if err := webhook.Notify(s.webhookURL, notifyMsg); err == nil {
			message = "Your request has been sent to the provider. They'll confirm shortly."
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "queued",
		"message": message,
	})
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
