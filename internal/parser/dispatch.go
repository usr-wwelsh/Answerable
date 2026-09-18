package parser

import (
	"fmt"
	"io"
	"strings"
)

// ParseByExtension dispatches to the right parser for a file extension
// (".csv", ".md", ".txt", ".pdf"), returning a single Record. For CSV it
// returns the first data row.
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
	default:
		return nil, fmt.Errorf("unsupported file type: %s", ext)
	}
}
