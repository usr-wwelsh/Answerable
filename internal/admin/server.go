package admin

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/usr-wwelsh/answerable/internal/booking"
	"github.com/usr-wwelsh/answerable/internal/config"
	"github.com/usr-wwelsh/answerable/internal/facts"
	"github.com/usr-wwelsh/answerable/internal/ingest"
	"github.com/usr-wwelsh/answerable/internal/parser"
)

var nowFunc = time.Now

type Deps struct {
	ConfigStore  *config.Store
	BookingStore *booking.Store
	UploadDir    string
	OnProvider   func(facts.Provider)
	OnWebhook    func(string)
}

type Server struct {
	cfgStore   *config.Store
	bookStore  *booking.Store
	uploadDir  string
	onProvider func(facts.Provider)
	onWebhook  func(string)
	fetcher    *ingest.Fetcher
	tmpl       *template.Template

	mu               sync.Mutex
	cfg              config.Config
	provider         facts.Provider
	lastErr          error
	refresher        *ingest.Refresher
	onboardingActive bool
}

func New(d Deps) (*Server, error) {
	tmpl, err := parseTemplates()
	if err != nil {
		return nil, err
	}

	s := &Server{
		cfgStore:         d.ConfigStore,
		bookStore:        d.BookingStore,
		uploadDir:        d.UploadDir,
		onProvider:       d.OnProvider,
		onWebhook:        d.OnWebhook,
		fetcher:          ingest.NewFetcher(),
		tmpl:             tmpl,
		onboardingActive: true,
	}

	if cfg, ok, err := d.ConfigStore.Load(); err == nil && ok {
		s.cfg = cfg
		s.onboardingActive = false
		s.restore()
	}

	return s, nil
}

// restore re-establishes provider state from a config loaded at startup:
// re-parses the configured source, seeds the public endpoint, and starts
// periodic refresh if the source is a URL with an interval set.
func (s *Server) restore() {
	rec, err := s.loadFromSource(s.cfg.Source)
	if err != nil {
		s.lastErr = err
		return
	}
	p, err := facts.FromRecord(rec)
	if err != nil {
		s.lastErr = err
		return
	}

	s.provider = p
	if s.onProvider != nil {
		s.onProvider(p)
	}
	if s.onWebhook != nil && s.cfg.WebhookURL != "" {
		s.onWebhook(s.cfg.WebhookURL)
	}
	s.startRefresherLocked()
}

func (s *Server) loadFromSource(src config.Source) (parser.Record, error) {
	switch src.Kind {
	case config.SourceURL:
		return s.fetcher.Fetch(src.Value)
	case config.SourceFile:
		f, err := os.Open(src.Value)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		return parser.ParseByExtension(filepath.Ext(src.Value), f)
	default:
		return nil, fmt.Errorf("no source configured")
	}
}

func (s *Server) startRefresherLocked() {
	if s.refresher != nil {
		s.refresher.Stop()
		s.refresher = nil
	}
	if s.cfg.Source.Kind == config.SourceURL && s.cfg.RefreshInterval > 0 {
		s.refresher = ingest.NewRefresher(s.cfg.RefreshInterval, s.refreshFromSource)
		s.refresher.Start()
	}
}

// refreshFromSource re-fetches and re-parses the current source, updating
// the served provider on success and recording the error otherwise (the
// last known-good provider stays live either way).
func (s *Server) refreshFromSource() {
	s.mu.Lock()
	src := s.cfg.Source
	s.mu.Unlock()

	rec, err := s.loadFromSource(src)
	if err != nil {
		s.mu.Lock()
		s.lastErr = err
		s.mu.Unlock()
		return
	}
	p, err := facts.FromRecord(rec)
	if err != nil {
		s.mu.Lock()
		s.lastErr = err
		s.mu.Unlock()
		return
	}

	s.mu.Lock()
	s.provider = p
	s.lastErr = nil
	s.cfg.SourceUpdatedAt = nowFunc()
	cfgCopy := s.cfg
	s.mu.Unlock()

	if err := s.cfgStore.Save(cfgCopy); err != nil {
		log.Printf("failed to persist refreshed source timestamp: %v", err)
	}

	if s.onProvider != nil {
		s.onProvider(p)
	}
}

func (s *Server) configured() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cfg.Source.Value != ""
}

func (s *Server) isOnboarding() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.onboardingActive
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleRoot)
	mux.HandleFunc("/onboarding", s.handleOnboardingIntro)
	mux.HandleFunc("/onboarding/source", s.handleSource)
	mux.HandleFunc("/onboarding/webhook", s.handleWebhook)
	mux.HandleFunc("/onboarding/webhook/skip", s.handleWebhookSkip)
	mux.HandleFunc("/refresh", s.handleRefresh)
	mux.HandleFunc("/queue", s.handleQueue)
	mux.HandleFunc("/queue/confirm", s.handleQueueConfirm)
	mux.HandleFunc("/queue/deny", s.handleQueueDeny)
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFileSystem()))))
	return mux
}

func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	if !s.configured() {
		http.Redirect(w, r, "/onboarding", http.StatusFound)
		return
	}
	s.renderDashboard(w, r)
}
