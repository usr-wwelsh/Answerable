package facts

import (
	"errors"

	"github.com/usr-wwelsh/answerable/internal/parser"
)

type Provider struct {
	Name       string
	Properties map[string]string
}

func FromRecord(rec parser.Record) (Provider, error) {
	name, ok := rec["name"]
	if !ok || name == "" {
		return Provider{}, errors.New("record missing required \"name\" field")
	}

	props := make(map[string]string, len(rec)-1)
	for k, v := range rec {
		if k == "name" {
			continue
		}
		props[k] = v
	}

	return Provider{Name: name, Properties: props}, nil
}
