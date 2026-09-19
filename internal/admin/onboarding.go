package admin

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/usr-wwelsh/answerable/internal/config"
	"github.com/usr-wwelsh/answerable/internal/email"
	"github.com/usr-wwelsh/answerable/internal/facts"
	"github.com/usr-wwelsh/answerable/internal/ingest"
	"github.com/usr-wwelsh/answerable/internal/parser"
)

const maxUploadBytes = 10 << 20

type sourceData struct {
	Error          string
	URL            string
	RefreshMinutes string
}

type webhookData struct {
	Error      string
	WebhookURL string
}

type emailData struct {
	Error       string
	SMTPHost    string
	SMTPPort    string
	Username    string
	From        string
	To          string
	HasPassword bool
}

func (s *Server) handleOnboardingIntro(w http.ResponseWriter, r *http.Request) {
	s.render(w, http.StatusOK, "intro", "Getting started", true, nil)
}

func (s *Server) renderSourceForm(w http.ResponseWriter, status int, data sourceData) {
	s.render(w, status, "source", "Where's your information?", s.isOnboarding(), data)
}

func (s *Server) handleSource(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		s.renderSourceForm(w, http.StatusOK, sourceData{RefreshMinutes: "0"})
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	wasConfigured := s.configured()

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)
	if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
		s.renderSourceForm(w, http.StatusBadRequest, sourceData{Error: "That upload was too large or malformed. Try again.", RefreshMinutes: "0"})
		return
	}

	urlVal := strings.TrimSpace(r.FormValue("url"))
	refreshRaw := r.FormValue("refresh_minutes")

	var fileHeader *multipart.FileHeader
	if r.MultipartForm != nil {
		if files := r.MultipartForm.File["file"]; len(files) > 0 && files[0].Filename != "" {
			fileHeader = files[0]
		}
	}

	if fileHeader != nil && urlVal != "" {
		s.renderSourceForm(w, http.StatusBadRequest, sourceData{Error: "Pick either a file or a link, not both.", URL: urlVal, RefreshMinutes: refreshRaw})
		return
	}
	if fileHeader == nil && urlVal == "" {
		s.renderSourceForm(w, http.StatusBadRequest, sourceData{Error: "Drop in a file or paste a link to continue.", RefreshMinutes: "0"})
		return
	}

	var (
		provider    facts.Provider
		newSrc      config.Source
		interval    time.Duration
		sourceLabel string
	)

	if fileHeader != nil {
		f, err := fileHeader.Open()
		if err != nil {
			s.renderSourceForm(w, http.StatusBadRequest, sourceData{Error: "Couldn't read that file.", RefreshMinutes: "0"})
			return
		}
		data, err := io.ReadAll(io.LimitReader(f, maxUploadBytes))
		f.Close()
		if err != nil {
			s.renderSourceForm(w, http.StatusBadRequest, sourceData{Error: "Couldn't read that file.", RefreshMinutes: "0"})
			return
		}

		ext := filepath.Ext(fileHeader.Filename)
		rec, err := parser.ParseByExtension(ext, strings.NewReader(string(data)))
		if err != nil {
			s.renderSourceForm(w, http.StatusBadRequest, sourceData{Error: friendlyParseError(err), RefreshMinutes: "0"})
			return
		}
		provider, err = facts.FromRecord(rec)
		if err != nil {
			s.renderSourceForm(w, http.StatusBadRequest, sourceData{Error: friendlyParseError(err), RefreshMinutes: "0"})
			return
		}

		savedPath := filepath.Join(s.uploadDir, "source"+strings.ToLower(ext))
		if err := os.WriteFile(savedPath, data, 0o600); err != nil {
			http.Error(w, "failed to save upload", http.StatusInternalServerError)
			return
		}
		newSrc = config.Source{Kind: config.SourceFile, Value: savedPath}
		sourceLabel = fileHeader.Filename
	} else {
		normalizedURL := ingest.NormalizeSourceURL(urlVal)
		if err := ingest.ValidateURL(normalizedURL); err != nil {
			s.renderSourceForm(w, http.StatusBadRequest, sourceData{Error: "That doesn't look like a web link (needs to start with http:// or https://).", URL: urlVal, RefreshMinutes: refreshRaw})
			return
		}
		rec, err := s.fetcher.Fetch(normalizedURL)
		if err != nil {
			s.renderSourceForm(w, http.StatusBadRequest, sourceData{Error: "Couldn't read that link: " + friendlyParseError(err), URL: urlVal, RefreshMinutes: refreshRaw})
			return
		}
		provider, err = facts.FromRecord(rec)
		if err != nil {
			s.renderSourceForm(w, http.StatusBadRequest, sourceData{Error: friendlyParseError(err), URL: urlVal, RefreshMinutes: refreshRaw})
			return
		}

		minutes, err := strconv.Atoi(strings.TrimSpace(refreshRaw))
		if err != nil || minutes < 0 {
			minutes = 0
		}
		newSrc = config.Source{Kind: config.SourceURL, Value: normalizedURL}
		interval = time.Duration(minutes) * time.Minute
		sourceLabel = urlVal
	}

	s.mu.Lock()
	s.cfg.Source = newSrc
	s.cfg.RefreshInterval = interval
	s.cfg.SourceLabel = sourceLabel
	s.cfg.SourceUpdatedAt = nowFunc()
	cfgCopy := s.cfg
	s.provider = provider
	s.lastErr = nil
	s.startRefresherLocked()
	s.mu.Unlock()

	if err := s.cfgStore.Save(cfgCopy); err != nil {
		http.Error(w, "failed to save configuration", http.StatusInternalServerError)
		return
	}

	if s.onProvider != nil {
		s.onProvider(provider)
	}

	if wasConfigured {
		http.Redirect(w, r, "/", http.StatusFound)
	} else {
		http.Redirect(w, r, "/onboarding/delivery", http.StatusFound)
	}
}

func (s *Server) handleOnboardingDelivery(w http.ResponseWriter, r *http.Request) {
	s.render(w, http.StatusOK, "delivery", "How should requests reach you?", s.isOnboarding(), nil)
}

func (s *Server) handleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		s.mu.Lock()
		current := s.cfg.WebhookURL
		s.mu.Unlock()
		s.render(w, http.StatusOK, "webhook", "Where should requests go?", s.isOnboarding(), webhookData{WebhookURL: current})
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		s.render(w, http.StatusBadRequest, "webhook", "Where should requests go?", s.isOnboarding(), webhookData{Error: "Something went wrong reading that form."})
		return
	}

	webhookURL := strings.TrimSpace(r.FormValue("webhook_url"))
	if webhookURL == "" {
		s.finishOnboarding()
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	if err := ingest.ValidateURL(webhookURL); err != nil {
		s.render(w, http.StatusBadRequest, "webhook", "Where should requests go?", s.isOnboarding(), webhookData{
			Error:      "That doesn't look like a web link (needs to start with http:// or https://).",
			WebhookURL: webhookURL,
		})
		return
	}

	s.mu.Lock()
	s.cfg.WebhookURL = webhookURL
	cfgCopy := s.cfg
	s.mu.Unlock()

	if err := s.cfgStore.Save(cfgCopy); err != nil {
		http.Error(w, "failed to save configuration", http.StatusInternalServerError)
		return
	}

	if s.onWebhook != nil {
		s.onWebhook(webhookURL)
	}

	s.finishOnboarding()
	http.Redirect(w, r, "/", http.StatusFound)
}

func (s *Server) handleOnboardingSkip(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.finishOnboarding()
	http.Redirect(w, r, "/", http.StatusFound)
}

func (s *Server) renderEmailForm(w http.ResponseWriter, status int, data emailData) {
	s.render(w, status, "email", "Where should requests go?", s.isOnboarding(), data)
}

func (s *Server) handleEmail(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		s.mu.Lock()
		current := s.cfg.Email
		s.mu.Unlock()
		s.renderEmailForm(w, http.StatusOK, emailDataFromConfig(current))
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		s.renderEmailForm(w, http.StatusBadRequest, emailData{Error: "Something went wrong reading that form."})
		return
	}

	host := strings.TrimSpace(r.FormValue("smtp_host"))
	portRaw := strings.TrimSpace(r.FormValue("smtp_port"))
	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")
	from := strings.TrimSpace(r.FormValue("from"))
	to := strings.TrimSpace(r.FormValue("to"))

	if host == "" && portRaw == "" && username == "" && password == "" && from == "" && to == "" {
		s.finishOnboarding()
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	echo := emailData{SMTPHost: host, SMTPPort: portRaw, Username: username, From: from, To: to}

	port, err := strconv.Atoi(portRaw)
	if err != nil {
		echo.Error = "Port must be a number, like 587."
		s.renderEmailForm(w, http.StatusBadRequest, echo)
		return
	}

	s.mu.Lock()
	if password == "" {
		password = s.cfg.Email.Password
	}
	s.mu.Unlock()

	newEmail := config.Email{SMTPHost: host, SMTPPort: port, Username: username, Password: password, From: from, To: to}
	if err := (email.Config{SMTPHost: newEmail.SMTPHost, SMTPPort: newEmail.SMTPPort, From: newEmail.From, To: newEmail.To}).Validate(); err != nil {
		echo.Error = "That doesn't look right: " + err.Error()
		s.renderEmailForm(w, http.StatusBadRequest, echo)
		return
	}

	s.mu.Lock()
	s.cfg.Email = newEmail
	cfgCopy := s.cfg
	s.mu.Unlock()

	if err := s.cfgStore.Save(cfgCopy); err != nil {
		http.Error(w, "failed to save configuration", http.StatusInternalServerError)
		return
	}

	if s.onEmail != nil {
		s.onEmail(newEmail)
	}

	s.finishOnboarding()
	http.Redirect(w, r, "/", http.StatusFound)
}

func emailDataFromConfig(e config.Email) emailData {
	data := emailData{
		SMTPHost:    e.SMTPHost,
		Username:    e.Username,
		From:        e.From,
		To:          e.To,
		HasPassword: e.Password != "",
	}
	if e.SMTPPort != 0 {
		data.SMTPPort = strconv.Itoa(e.SMTPPort)
	}
	return data
}

func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.refreshFromSource()
	http.Redirect(w, r, "/", http.StatusFound)
}

func (s *Server) finishOnboarding() {
	s.mu.Lock()
	s.onboardingActive = false
	s.mu.Unlock()
}

func friendlyParseError(err error) string {
	return fmt.Sprintf("We couldn't make sense of that: %v", err)
}
