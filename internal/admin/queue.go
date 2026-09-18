package admin

import (
	"net/http"

	"github.com/usr-wwelsh/answerable/internal/booking"
)

type queueData struct {
	Requests []booking.Request
	Error    string
}

func (s *Server) handleQueue(w http.ResponseWriter, r *http.Request) {
	list, err := s.bookStore.List()
	if err != nil {
		http.Error(w, "failed to load requests", http.StatusInternalServerError)
		return
	}
	s.render(w, http.StatusOK, "queue", "Requests", false, queueData{
		Requests: list,
		Error:    r.URL.Query().Get("error"),
	})
}

func (s *Server) handleQueueConfirm(w http.ResponseWriter, r *http.Request) {
	s.resolveQueueToken(w, r, s.bookStore.Confirm)
}

func (s *Server) handleQueueDeny(w http.ResponseWriter, r *http.Request) {
	s.resolveQueueToken(w, r, s.bookStore.Deny)
}

func (s *Server) resolveQueueToken(w http.ResponseWriter, r *http.Request, resolve func(string) (booking.Request, error)) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}

	token := r.FormValue("token")
	if _, err := resolve(token); err != nil {
		redirectWithError(w, r, "/queue", err)
		return
	}

	http.Redirect(w, r, "/queue", http.StatusFound)
}
