package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

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

	rec, err := parser.ParseByExtension(filepath.Ext(path), f)
	if err != nil {
		return facts.Provider{}, err
	}
	return facts.FromRecord(rec)
}
