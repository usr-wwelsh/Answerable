package admin

import (
	"net/http"

	"github.com/usr-wwelsh/answerable/internal/booking"
)

type queueData struct {
	Requests []booking.Request
}

func (s *Server) handleQueue(w http.ResponseWriter, r *http.Request) {
	list, err := s.bookStore.List()
	if err != nil {
		http.Error(w, "failed to load requests", http.StatusInternalServerError)
		return
	}
	s.render(w, http.StatusOK, "queue", "Requests", false, queueData{Requests: list})
}
