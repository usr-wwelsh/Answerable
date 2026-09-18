package endpoint

import (
	"net/http"

	"github.com/usr-wwelsh/answerable/internal/agentcard"
	"github.com/usr-wwelsh/answerable/internal/facts"
	"github.com/usr-wwelsh/answerable/internal/jsonld"
	"github.com/usr-wwelsh/answerable/internal/llmstxt"
)

type Server struct {
	provider facts.Provider
}

func New(p facts.Provider) *Server {
	return &Server{provider: p}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/facts.jsonld", s.handleFacts)
	mux.HandleFunc("/.well-known/agent.json", s.handleAgentCard)
	mux.HandleFunc("/llms.txt", s.handleLLMsTxt)
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
