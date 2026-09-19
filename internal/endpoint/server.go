package endpoint

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"

	"github.com/usr-wwelsh/answerable/internal/agentcard"
	"github.com/usr-wwelsh/answerable/internal/booking"
	"github.com/usr-wwelsh/answerable/internal/email"
	"github.com/usr-wwelsh/answerable/internal/facts"
	"github.com/usr-wwelsh/answerable/internal/jsonld"
	"github.com/usr-wwelsh/answerable/internal/llmstxt"
)

type Server struct {
	mu         sync.RWMutex
	provider   facts.Provider
	store      *booking.Store
	webhookURL string
	emailCfg   email.Config
}

func New(p facts.Provider, store *booking.Store, webhookURL string) *Server {
	return &Server{provider: p, store: store, webhookURL: webhookURL}
}

// UpdateProvider replaces the served provider facts, e.g. after the admin
// webui re-ingests a source. Safe to call while the server is serving.
func (s *Server) UpdateProvider(p facts.Provider) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.provider = p
}

// UpdateWebhook replaces the booking-notification webhook URL.
func (s *Server) UpdateWebhook(url string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.webhookURL = url
}

// UpdateEmail replaces the booking-notification SMTP configuration.
func (s *Server) UpdateEmail(cfg email.Config) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.emailCfg = cfg
}

func (s *Server) currentProvider() facts.Provider {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.provider
}

func (s *Server) currentWebhook() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.webhookURL
}

func (s *Server) currentEmail() email.Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.emailCfg
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/{$}", s.handleIndex)
	mux.HandleFunc("/robots.txt", s.handleRobots)
	mux.HandleFunc("/facts.jsonld", s.handleFacts)
	mux.HandleFunc("/.well-known/agent.json", s.handleAgentCard)
	mux.HandleFunc("/.well-known/mcp.json", s.handleMCPManifest)
	mux.HandleFunc("/llms.txt", s.handleLLMsTxt)
	mux.HandleFunc("/book", s.handleBook)
	mux.HandleFunc("/status", s.handleStatus)
	mux.HandleFunc("/confirm", s.handleConfirm)
	mux.HandleFunc("/deny", s.handleDeny)
	mux.Handle("/mcp", s.mcpHandler())
	return withDiscoveryLinks(mux)
}

// withDiscoveryLinks adds a Link header advertising the agent-discovery
// routes to every response, including 404s, so any known URL on this host
// leads an agent back to the discovery trailhead without a body to parse.
func withDiscoveryLinks(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		base := baseURL(r)
		w.Header().Set("Link", strings.Join([]string{
			`<` + base + `/llms.txt>; rel="llms-txt"`,
			`<` + base + `/.well-known/agent.json>; rel="agent-card"`,
			`<` + base + `/.well-known/mcp.json>; rel="mcp-manifest"`,
			`<` + base + `/mcp>; rel="mcp-server"`,
		}, ", "))
		next.ServeHTTP(w, r)
	})
}

func baseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}

func (s *Server) handleFacts(w http.ResponseWriter, r *http.Request) {
	out, err := jsonld.RenderProvider(s.currentProvider(), baseURL(r)+"/book")
	if err != nil {
		http.Error(w, "failed to render facts", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/ld+json")
	w.Write(out)
}

func (s *Server) handleAgentCard(w http.ResponseWriter, r *http.Request) {
	out, err := agentcard.Render(s.currentProvider(), baseURL(r))
	if err != nil {
		http.Error(w, "failed to render agent card", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(out)
}

func (s *Server) handleLLMsTxt(w http.ResponseWriter, r *http.Request) {
	out := llmstxt.Render(s.currentProvider(), baseURL(r)+"/facts.jsonld", baseURL(r)+"/.well-known/agent.json", baseURL(r)+"/mcp")
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte(out))
}

type discoveryIndex struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	LLMsTxt     string `json:"llms_txt"`
	AgentCard   string `json:"agent_card"`
	MCPManifest string `json:"mcp_manifest"`
	MCP         string `json:"mcp"`
	Facts       string `json:"facts"`
}

// handleIndex is the first URL any crawler or agent tries. A plain GET
// (no Accept: application/json) redirects straight to llms.txt, the
// human- and agent-readable discovery doc; an explicit JSON request gets
// a machine-readable index of every discovery route instead of a 404.
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	base := baseURL(r)
	if !strings.Contains(r.Header.Get("Accept"), "application/json") {
		http.Redirect(w, r, "/llms.txt", http.StatusFound)
		return
	}
	idx := discoveryIndex{
		Name:        s.currentProvider().Name,
		Description: "Agent discovery index for " + s.currentProvider().Name + ".",
		LLMsTxt:     base + "/llms.txt",
		AgentCard:   base + "/.well-known/agent.json",
		MCPManifest: base + "/.well-known/mcp.json",
		MCP:         base + "/mcp",
		Facts:       base + "/facts.jsonld",
	}
	out, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		http.Error(w, "failed to render index", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(out)
}

type mcpManifest struct {
	SchemaVersion   string `json:"schemaVersion"`
	Name            string `json:"name"`
	Version         string `json:"version"`
	ProtocolVersion string `json:"protocolVersion"`
	Transport       string `json:"transport"`
	Endpoint        string `json:"endpoint"`
}

func (s *Server) handleMCPManifest(w http.ResponseWriter, r *http.Request) {
	out, err := json.MarshalIndent(mcpManifest{
		SchemaVersion:   "1.0",
		Name:            "answerable",
		Version:         "0.1.0",
		ProtocolVersion: "2024-11-05",
		Transport:       "streamable-http",
		Endpoint:        baseURL(r) + "/mcp",
	}, "", "  ")
	if err != nil {
		http.Error(w, "failed to render mcp manifest", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(out)
}

func (s *Server) handleRobots(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte("User-agent: *\nAllow: /\n\n# llms.txt: " + baseURL(r) + "/llms.txt\n"))
}
