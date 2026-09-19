package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/usr-wwelsh/answerable/internal/admin"
	"github.com/usr-wwelsh/answerable/internal/booking"
	"github.com/usr-wwelsh/answerable/internal/config"
	"github.com/usr-wwelsh/answerable/internal/email"
	"github.com/usr-wwelsh/answerable/internal/endpoint"
	"github.com/usr-wwelsh/answerable/internal/facts"
	"github.com/usr-wwelsh/answerable/internal/parser"
)

func main() {
	file := flag.String("file", "", "optional: path to a CSV/md/txt/pdf fact source to pre-seed on first run")
	webhookURL := flag.String("webhook", "", "optional: webhook URL to pre-seed on first run")
	port := flag.String("port", "8081", "public endpoint port")
	adminBind := flag.String("admin-bind", "127.0.0.1", "admin webui bind address; widening past loopback requires ANSWERABLE_ADMIN_PASSWORD")
	adminPort := flag.String("admin-port", "8090", "admin webui port")
	dbPath := flag.String("db", "answerable.db", "path to the local database (booking queue + config)")
	noBrowser := flag.Bool("no-browser", false, "don't open the admin webui in a browser on startup")
	flag.Parse()

	cfgStore, err := config.Open(*dbPath)
	if err != nil {
		log.Fatalf("failed to open config store: %v", err)
	}
	defer cfgStore.Close()

	bookStore, err := booking.Open(*dbPath)
	if err != nil {
		log.Fatalf("failed to open booking store: %v", err)
	}
	defer bookStore.Close()

	seedIfUnconfigured(cfgStore, *file, *webhookURL)

	srv := endpoint.New(facts.Provider{}, bookStore, "")

	adminSrv, err := admin.New(admin.Deps{
		ConfigStore:  cfgStore,
		BookingStore: bookStore,
		UploadDir:    filepath.Dir(*dbPath),
		OnProvider:   srv.UpdateProvider,
		OnWebhook:    srv.UpdateWebhook,
		OnEmail: func(e config.Email) {
			srv.UpdateEmail(email.Config{
				SMTPHost: e.SMTPHost,
				SMTPPort: e.SMTPPort,
				Username: e.Username,
				Password: e.Password,
				From:     e.From,
				To:       e.To,
			})
		},
	})
	if err != nil {
		log.Fatalf("failed to start admin webui: %v", err)
	}

	adminPassword := os.Getenv("ANSWERABLE_ADMIN_PASSWORD")
	if err := admin.RequireAuthForBind(*adminBind, adminPassword); err != nil {
		log.Fatal(err)
	}
	adminHandler := admin.WrapWithAuth(*adminBind, adminPassword, adminSrv.Handler())

	adminAddr := *adminBind + ":" + *adminPort
	go func() {
		log.Printf("Admin setup at http://%s", adminAddr)
		if err := http.ListenAndServe(adminAddr, adminHandler); err != nil {
			log.Fatal(err)
		}
	}()

	if !*noBrowser {
		openBrowser("http://" + adminAddr)
	}

	log.Printf("Answerable public endpoint on :%s", *port)
	if err := http.ListenAndServe(":"+*port, srv.Handler()); err != nil {
		log.Fatal(err)
	}
}

// seedIfUnconfigured pre-seeds provider config from -file/-webhook on the
// very first run only, so existing demo/test invocations keep working. Once
// a config exists (from a prior run, or from the admin webui), the flags
// are ignored — the persisted config and the admin webui take over.
func seedIfUnconfigured(store *config.Store, file, webhookURL string) {
	if file == "" {
		return
	}
	if _, ok, err := store.Load(); err != nil || ok {
		return
	}

	f, err := os.Open(file)
	if err != nil {
		log.Printf("warning: could not pre-seed from %s: %v", file, err)
		return
	}
	defer f.Close()

	rec, err := parser.ParseByExtension(filepath.Ext(file), f)
	if err != nil {
		log.Printf("warning: could not pre-seed from %s: %v", file, err)
		return
	}
	if _, err := facts.FromRecord(rec); err != nil {
		log.Printf("warning: could not pre-seed from %s: %v", file, err)
		return
	}

	abs, err := filepath.Abs(file)
	if err != nil {
		abs = file
	}

	err = store.Save(config.Config{
		Source:          config.Source{Kind: config.SourceFile, Value: abs},
		WebhookURL:      webhookURL,
		SourceLabel:     filepath.Base(file),
		SourceUpdatedAt: time.Now(),
	})
	if err != nil {
		log.Printf("warning: could not save pre-seeded config: %v", err)
	}
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	if err := cmd.Start(); err != nil {
		log.Printf("could not open a browser automatically — open %s manually", url)
	}
}
