package jsonld

import (
	"encoding/json"
	"testing"

	"github.com/usr-wwelsh/answerable/internal/facts"
)

func TestRenderProviderIncludesContextTypeAndName(t *testing.T) {
	p := facts.Provider{
		Name: "Test Shelter A",
		Properties: map[string]string{
			"capacity_available": "12",
			"eligibility":        "walk-in",
		},
	}

	out, err := RenderProvider(p, "http://localhost:8080/book")
	if err != nil {
		t.Fatalf("RenderProvider returned error: %v", err)
	}

	var doc map[string]any
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatalf("RenderProvider output is not valid JSON: %v", err)
	}

	if doc["@context"] != "https://schema.org" {
		t.Errorf("@context = %v, want https://schema.org", doc["@context"])
	}
	if doc["@type"] != "Service" {
		t.Errorf("@type = %v, want Service", doc["@type"])
	}
	if doc["name"] != "Test Shelter A" {
		t.Errorf("name = %v, want Test Shelter A", doc["name"])
	}
}

func TestRenderProviderIncludesPropertiesAsAdditionalProperty(t *testing.T) {
	p := facts.Provider{
		Name: "Test Shelter A",
		Properties: map[string]string{
			"capacity_available": "12",
		},
	}

	out, err := RenderProvider(p, "http://localhost:8080/book")
	if err != nil {
		t.Fatalf("RenderProvider returned error: %v", err)
	}

	var doc struct {
		AdditionalProperty []struct {
			Type  string `json:"@type"`
			Name  string `json:"name"`
			Value string `json:"value"`
		} `json:"additionalProperty"`
	}
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if len(doc.AdditionalProperty) != 1 {
		t.Fatalf("expected 1 additionalProperty entry, got %d", len(doc.AdditionalProperty))
	}
	got := doc.AdditionalProperty[0]
	if got.Type != "PropertyValue" || got.Name != "capacity_available" || got.Value != "12" {
		t.Errorf("additionalProperty[0] = %+v, want {PropertyValue capacity_available 12}", got)
	}
}

func TestRenderProviderIncludesReserveAction(t *testing.T) {
	p := facts.Provider{Name: "Test Shelter A", Properties: map[string]string{}}

	out, err := RenderProvider(p, "http://localhost:8080/book")
	if err != nil {
		t.Fatalf("RenderProvider returned error: %v", err)
	}

	var doc struct {
		PotentialAction struct {
			Type   string `json:"@type"`
			Target string `json:"target"`
		} `json:"potentialAction"`
	}
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if doc.PotentialAction.Type != "ReserveAction" {
		t.Errorf("potentialAction.@type = %q, want ReserveAction", doc.PotentialAction.Type)
	}
	if doc.PotentialAction.Target != "http://localhost:8080/book" {
		t.Errorf("potentialAction.target = %q, want http://localhost:8080/book", doc.PotentialAction.Target)
	}
}
