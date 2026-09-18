package admin

import (
	"net/http"
	"sort"
)

type kv struct {
	Key   string
	Value string
}

type dashboardData struct {
	ProviderName      string
	Properties        []kv
	SourceKind        string
	RefreshMinutes    int
	WebhookConfigured bool
	QueueCount        int
	LastError         string
}

func (s *Server) renderDashboard(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	provider := s.provider
	cfg := s.cfg
	lastErr := s.lastErr
	s.mu.Unlock()

	keys := make([]string, 0, len(provider.Properties))
	for k := range provider.Properties {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	props := make([]kv, 0, len(keys))
	for _, k := range keys {
		props = append(props, kv{Key: k, Value: provider.Properties[k]})
	}

	queueCount := 0
	if list, err := s.bookStore.List(); err == nil {
		queueCount = len(list)
	}

	errMsg := ""
	if lastErr != nil {
		errMsg = lastErr.Error()
	}

	s.render(w, http.StatusOK, "dashboard", providerTitle(provider.Name), false, dashboardData{
		ProviderName:      provider.Name,
		Properties:        props,
		SourceKind:        string(cfg.Source.Kind),
		RefreshMinutes:    int(cfg.RefreshInterval.Minutes()),
		WebhookConfigured: cfg.WebhookURL != "",
		QueueCount:        queueCount,
		LastError:         errMsg,
	})
}

func providerTitle(name string) string {
	if name == "" {
		return "Dashboard"
	}
	return name
}
