package parser

import (
	"bytes"
	"compress/zlib"
	"io"
	"regexp"
	"strings"
)

const maxPDFStreamBytes = 32 << 20

var (
	streamRe = regexp.MustCompile(`(?s)stream\r?\n(.*?)\r?\n?endstream`)
	tokenRe  = regexp.MustCompile(`\((?:[^()\\]|\\.)*\)|Tj|TJ|'|"`)
)

// ParsePDF does basic, best-effort text extraction from a PDF: it decodes
// FlateDecode content streams and reads literal-string show operators
// (Tj/TJ/'/"). It does not understand layout, fonts, or hex strings, and is
// not a substitute for OCR — "reasonable quality" only.
func ParsePDF(r io.Reader) (Record, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	text := extractPDFText(data)
	return ParseText(strings.NewReader(text))
}

func extractPDFText(data []byte) string {
	var out strings.Builder
	for _, m := range streamRe.FindAllSubmatch(data, -1) {
		raw := m[1]
		content := raw
		if dec, err := inflate(raw); err == nil {
			content = dec
		}
		out.WriteString(extractShowOperators(content))
	}
	return out.String()
}

func inflate(raw []byte) ([]byte, error) {
	zr, err := zlib.NewReader(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	defer zr.Close()

	out, err := io.ReadAll(io.LimitReader(zr, maxPDFStreamBytes))
	if err != nil {
		return nil, err
	}
	return out, nil
}

func extractShowOperators(content []byte) string {
	var out strings.Builder
	var pending strings.Builder
	for _, tok := range tokenRe.FindAll(content, -1) {
		switch string(tok) {
		case "Tj", "TJ", "'", `"`:
			if pending.Len() > 0 {
				out.WriteString(pending.String())
				out.WriteByte('\n')
				pending.Reset()
			}
		default:
			pending.Write(decodePDFString(tok[1 : len(tok)-1]))
		}
	}
	return out.String()
}

func decodePDFString(s []byte) []byte {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] != '\\' || i == len(s)-1 {
			out = append(out, s[i])
			continue
		}
		i++
		switch s[i] {
		case 'n':
			out = append(out, '\n')
		case 'r':
			out = append(out, '\r')
		case 't':
			out = append(out, '\t')
		case 'b':
			out = append(out, '\b')
		case 'f':
			out = append(out, '\f')
		case '(', ')', '\\':
			out = append(out, s[i])
		default:
			out = append(out, s[i])
		}
	}
	return out
}
