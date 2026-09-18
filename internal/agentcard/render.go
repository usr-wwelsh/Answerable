package agentcard

import (
	"encoding/json"

	"github.com/usr-wwelsh/answerable/internal/facts"
)

type skill struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Tags        []string `json:"tags"`
}

type capabilities struct {
	Streaming         bool `json:"streaming"`
	PushNotifications bool `json:"pushNotifications"`
}

type card struct {
	Name               string       `json:"name"`
	Description        string       `json:"description"`
	URL                string       `json:"url"`
	Version            string       `json:"version"`
	Capabilities       capabilities `json:"capabilities"`
	DefaultInputModes  []string     `json:"defaultInputModes"`
	DefaultOutputModes []string     `json:"defaultOutputModes"`
	Skills             []skill      `json:"skills"`
}

func Render(p facts.Provider, baseURL string) ([]byte, error) {
	c := card{
		Name:               p.Name,
		Description:        "Live availability and eligibility facts, published by " + p.Name + ".",
		URL:                baseURL,
		Version:            "1.0.0",
		Capabilities:       capabilities{Streaming: false, PushNotifications: false},
		DefaultInputModes:  []string{"application/json"},
		DefaultOutputModes: []string{"application/json"},
		Skills: []skill{
			{
				ID:          "check-availability",
				Name:        "Check availability",
				Description: "Look up current capacity, eligibility, and hours.",
				Tags:        []string{"availability", "eligibility"},
			},
			{
				ID:          "book-intake",
				Name:        "Request intake booking",
				Description: "Submit an intake request for a person or people in need.",
				Tags:        []string{"booking", "intake"},
			},
		},
	}

	return json.MarshalIndent(c, "", "  ")
}
