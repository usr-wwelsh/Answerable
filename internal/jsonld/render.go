package jsonld

import (
	"encoding/json"
	"sort"

	"github.com/usr-wwelsh/answerable/internal/facts"
)

type propertyValue struct {
	Type  string `json:"@type"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

type reserveAction struct {
	Type   string `json:"@type"`
	Target string `json:"target"`
}

type document struct {
	Context            string          `json:"@context"`
	Type               string          `json:"@type"`
	Name               string          `json:"name"`
	AdditionalProperty []propertyValue `json:"additionalProperty"`
	PotentialAction    reserveAction   `json:"potentialAction"`
}

func RenderProvider(p facts.Provider, bookingURL string) ([]byte, error) {
	keys := make([]string, 0, len(p.Properties))
	for k := range p.Properties {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	props := make([]propertyValue, 0, len(keys))
	for _, k := range keys {
		props = append(props, propertyValue{
			Type:  "PropertyValue",
			Name:  k,
			Value: p.Properties[k],
		})
	}

	doc := document{
		Context:            "https://schema.org",
		Type:               "Service",
		Name:               p.Name,
		AdditionalProperty: props,
		PotentialAction: reserveAction{
			Type:   "ReserveAction",
			Target: bookingURL,
		},
	}

	return json.MarshalIndent(doc, "", "  ")
}
