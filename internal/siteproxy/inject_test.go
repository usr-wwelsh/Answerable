package siteproxy

import (
	"strings"
	"testing"
)

func TestInjectLinksAddsTagsRightAfterHead(t *testing.T) {
	body := []byte("<html><head><title>x</title></head><body></body></html>")
	links := []DiscoveryLink{{"llms-txt", "/llms.txt"}, {"agent-card", "/.well-known/agent.json"}}

	out := string(InjectLinks(body, links))

	if !strings.Contains(out, `<head><link rel="llms-txt" href="/llms.txt"><link rel="agent-card" href="/.well-known/agent.json"><title>x</title>`) {
		t.Errorf("links not inserted right after <head>: %s", out)
	}
}

func TestInjectLinksHandlesHeadWithAttributes(t *testing.T) {
	body := []byte(`<html><head lang="en"><title>x</title></head></html>`)
	out := string(InjectLinks(body, []DiscoveryLink{{"llms-txt", "/llms.txt"}}))

	if !strings.Contains(out, `<head lang="en"><link rel="llms-txt" href="/llms.txt">`) {
		t.Errorf("links not inserted after attributed <head>: %s", out)
	}
}

func TestInjectLinksLeavesBodyWithoutHeadUntouched(t *testing.T) {
	body := []byte(`{"just":"json"}`)
	out := InjectLinks(body, DefaultLinks)

	if string(out) != string(body) {
		t.Errorf("body without <head> was modified: %s", out)
	}
}

func TestInjectLinksIsCaseInsensitiveToHeadTag(t *testing.T) {
	body := []byte("<HTML><HEAD><title>x</title></HEAD></HTML>")
	out := string(InjectLinks(body, []DiscoveryLink{{"llms-txt", "/llms.txt"}}))

	if !strings.Contains(out, `<HEAD><link rel="llms-txt" href="/llms.txt">`) {
		t.Errorf("links not inserted after uppercase <HEAD>: %s", out)
	}
}
