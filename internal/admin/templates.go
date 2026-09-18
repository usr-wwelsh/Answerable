package admin

import (
	"bytes"
	"embed"
	"html/template"
	"net/http"
	"net/url"
)

//go:embed templates/*.html
var templateFS embed.FS

func parseTemplates() (*template.Template, error) {
	return template.ParseFS(templateFS, "templates/*.html")
}

func redirectWithError(w http.ResponseWriter, r *http.Request, path string, err error) {
	http.Redirect(w, r, path+"?error="+url.QueryEscape(err.Error()), http.StatusFound)
}

type pageData struct {
	Title    string
	Centered bool
	Body     template.HTML
}

func (s *Server) render(w http.ResponseWriter, status int, page, title string, centered bool, data any) {
	var buf bytes.Buffer
	if err := s.tmpl.ExecuteTemplate(&buf, page, data); err != nil {
		http.Error(w, "failed to render page", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := s.tmpl.ExecuteTemplate(w, "layout", pageData{
		Title:    title,
		Centered: centered,
		Body:     template.HTML(buf.String()),
	}); err != nil {
		http.Error(w, "failed to render page", http.StatusInternalServerError)
	}
}
