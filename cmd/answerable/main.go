package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/usr-wwelsh/answerable/internal/booking"
	"github.com/usr-wwelsh/answerable/internal/endpoint"
	"github.com/usr-wwelsh/answerable/internal/facts"
	"github.com/usr-wwelsh/answerable/internal/parser"
)

func main() {
	file := flag.String("file", "", "path to a CSV, md, or txt fact source")
	webhookURL := flag.String("webhook", "", "Slack/Discord/Teams incoming webhook URL for booking notifications")
	port := flag.String("port", "8081", "public endpoint port")
	dbPath := flag.String("db", "answerable.db", "path to the local booking queue database")
	flag.Parse()

	if *file == "" {
		fmt.Fprintln(os.Stderr, "usage: answerable -file <path.csv|.md|.txt> [-webhook <url>] [-port 8081]")
		os.Exit(1)
	}

	provider, err := loadProvider(*file)
	if err != nil {
		log.Fatalf("failed to load provider from %s: %v", *file, err)
	}

	store, err := booking.Open(*dbPath)
	if err != nil {
		log.Fatalf("failed to open booking store: %v", err)
	}
	defer store.Close()

	srv := endpoint.New(provider, store, *webhookURL)

	log.Printf("Answerable serving %q on :%s (facts.jsonld, .well-known/agent.json, llms.txt, book)", provider.Name, *port)
	if err := http.ListenAndServe(":"+*port, srv.Handler()); err != nil {
		log.Fatal(err)
	}
}

func loadProvider(path string) (facts.Provider, error) {
	f, err := os.Open(path)
	if err != nil {
		return facts.Provider{}, err
	}
	defer f.Close()

	switch strings.ToLower(filepath.Ext(path)) {
	case ".csv":
		records, err := parser.ParseCSV(f)
		if err != nil {
			return facts.Provider{}, err
		}
		if len(records) == 0 {
			return facts.Provider{}, fmt.Errorf("no rows found in %s", path)
		}
		return facts.FromRecord(records[0])
	case ".md", ".txt":
		rec, err := parser.ParseText(f)
		if err != nil {
			return facts.Provider{}, err
		}
		return facts.FromRecord(rec)
	default:
		return facts.Provider{}, fmt.Errorf("unsupported file type: %s", path)
	}
}
