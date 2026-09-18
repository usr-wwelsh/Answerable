package agentcard

import (
	"encoding/json"
	"testing"

	"github.com/usr-wwelsh/answerable/internal/facts"
)

func TestRenderIncludesNameAndURL(t *testing.T) {
	p := facts.Provider{Name: "Test Shelter A"}

	out, err := Render(p, "http://localhost:8080")
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}

	var doc map[string]any
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatalf("Render output is not valid JSON: %v", err)
	}

	if doc["name"] != "Test Shelter A" {
		t.Errorf("name = %v, want Test Shelter A", doc["name"])
	}
	if doc["url"] != "http://localhost:8080" {
		t.Errorf("url = %v, want http://localhost:8080", doc["url"])
	}
}

func TestRenderIncludesAvailabilityAndBookingSkills(t *testing.T) {
	p := facts.Provider{Name: "Test Shelter A"}

	out, err := Render(p, "http://localhost:8080")
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}

	var doc struct {
		Skills []struct {
			ID   string   `json:"id"`
			Tags []string `json:"tags"`
		} `json:"skills"`
	}
	if err := json.Unmarshal(out, &doc); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if len(doc.Skills) != 2 {
		t.Fatalf("expected 2 skills, got %d", len(doc.Skills))
	}
	ids := map[string]bool{doc.Skills[0].ID: true, doc.Skills[1].ID: true}
	if !ids["check-availability"] || !ids["book-intake"] {
		t.Errorf("skills = %+v, want check-availability and book-intake", doc.Skills)
	}
}
