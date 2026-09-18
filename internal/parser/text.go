package parser

import (
	"bufio"
	"io"
	"strings"
)

func ParseText(r io.Reader) (Record, error) {
	rec := Record{}

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := stripMarkdownNoise(scanner.Text())

		idx := strings.Index(line, ":")
		if idx < 0 {
			continue
		}

		key := normalizeKey(line[:idx])
		value := strings.TrimSpace(line[idx+1:])
		if key == "" || value == "" {
			continue
		}
		rec[key] = value
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return rec, nil
}

func stripMarkdownNoise(line string) string {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "- ")
	line = strings.TrimPrefix(line, "* ")
	line = strings.TrimLeft(line, "#")
	line = strings.ReplaceAll(line, "*", "")
	line = strings.ReplaceAll(line, "_", "")
	return strings.TrimSpace(line)
}

func normalizeKey(key string) string {
	key = strings.TrimSpace(key)
	key = strings.ToLower(key)
	return strings.ReplaceAll(key, " ", "_")
}
