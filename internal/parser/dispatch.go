package parser

import (
	"fmt"
	"io"
	"strings"
)

// ParseByExtension dispatches to the right parser for a file extension
// (".csv", ".md", ".txt", ".pdf", ".json", ".html", ".htm", ".xml", ".docx",
// ".pptx", ".xlsx"), returning a single Record. For CSV/XLSX it returns the
// first data row.
func ParseByExtension(ext string, r io.Reader) (Record, error) {
	switch strings.ToLower(ext) {
	case ".csv":
		records, err := ParseCSV(r)
		if err != nil {
			return nil, err
		}
		if len(records) == 0 {
			return nil, fmt.Errorf("no rows found")
		}
		return records[0], nil
	case ".md", ".txt":
		return ParseText(r)
	case ".pdf":
		return ParsePDF(r)
	case ".json":
		return ParseJSON(r)
	case ".html", ".htm":
		return ParseHTML(r)
	case ".xml":
		return ParseXML(r)
	case ".docx":
		return ParseDOCX(r)
	case ".pptx":
		return ParsePPTX(r)
	case ".xlsx":
		return ParseXLSX(r)
	default:
		return nil, fmt.Errorf("unsupported file type: %s", ext)
	}
}
