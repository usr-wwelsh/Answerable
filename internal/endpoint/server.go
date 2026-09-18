package endpoint

import (
	"net/http"

	"github.com/usr-wwelsh/answerable/internal/agentcard"
	"github.com/usr-wwelsh/answerable/internal/booking"
	"github.com/usr-wwelsh/answerable/internal/facts"
	"github.com/usr-wwelsh/answerable/internal/jsonld"
	"github.com/usr-wwelsh/answerable/internal/llmstxt"
)

type Server struct {
	provider   facts.Provider
	store      *booking.Store
	webhookURL string
}

func New(p facts.Provider, store *booking.Store, webhookURL string) *Server {
	return &Server{provider: p, store: store, webhookURL: webhookURL}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/facts.jsonld", s.handleFacts)
	mux.HandleFunc("/.well-known/agent.json", s.handleAgentCard)
	mux.HandleFunc("/llms.txt", s.handleLLMsTxt)
	mux.HandleFunc("/book", s.handleBook)
	mux.HandleFunc("/confirm", s.handleConfirm)
	mux.HandleFunc("/deny", s.handleDeny)
	return mux
}

func baseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}

func (s *Server) handleFacts(w http.ResponseWriter, r *http.Request) {
	out, err := jsonld.RenderProvider(s.provider, baseURL(r)+"/book")
	if err != nil {
		http.Error(w, "failed to render facts", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/ld+json")
	w.Write(out)
}

func (s *Server) handleAgentCard(w http.ResponseWriter, r *http.Request) {
	out, err := agentcard.Render(s.provider, baseURL(r))
	if err != nil {
		http.Error(w, "failed to render agent card", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(out)
}

func (s *Server) handleLLMsTxt(w http.ResponseWriter, r *http.Request) {
	out := llmstxt.Render(s.provider, baseURL(r)+"/facts.jsonld", baseURL(r)+"/.well-known/agent.json")
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte(out))
}
