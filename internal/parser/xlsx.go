package parser

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
)

var sheetFileRe = regexp.MustCompile(`^xl/worksheets/sheet(\d+)\.xml$`)

// ParseXLSX does best-effort spreadsheet extraction: it reads the
// lowest-numbered worksheet, treating row 1 as headers and row 2 as data
// (like ParseCSV's first row), resolving shared-string cell references. It
// does not understand formulas, formatting, or multiple sheets.
func ParseXLSX(r io.Reader) (Record, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("open xlsx: %w", err)
	}

	shared, err := readSharedStrings(zr)
	if err != nil {
		return nil, err
	}

	sheet := firstSheetFile(zr)
	if sheet == nil {
		return nil, fmt.Errorf("xlsx: no worksheet found")
	}

	xmlData, err := readZipFile(sheet)
	if err != nil {
		return nil, err
	}

	rows, err := readXLSXRows(xmlData, shared)
	if err != nil {
		return nil, err
	}
	if len(rows) < 2 {
		return nil, fmt.Errorf("no rows found")
	}

	header, data0 := rows[0], rows[1]
	rec := make(Record, len(header))
	for i, key := range header {
		key = normalizeKey(key)
		if key == "" {
			continue
		}
		if i < len(data0) {
			rec[key] = data0[i]
		}
	}
	return rec, nil
}

func firstSheetFile(zr *zip.Reader) *zip.File {
	var best *zip.File
	bestN := -1
	for _, f := range zr.File {
		m := sheetFileRe.FindStringSubmatch(f.Name)
		if m == nil {
			continue
		}
		n, err := strconv.Atoi(m[1])
		if err != nil {
			continue
		}
		if best == nil || n < bestN {
			best, bestN = f, n
		}
	}
	return best
}

func readSharedStrings(zr *zip.Reader) ([]string, error) {
	f := findZipFile(zr, "xl/sharedStrings.xml")
	if f == nil {
		return nil, nil
	}

	data, err := readZipFile(f)
	if err != nil {
		return nil, err
	}

	dec := xml.NewDecoder(bytes.NewReader(data))
	var strs []string
	var cur strings.Builder
	inSI, inText := false, false
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "si":
				inSI = true
				cur.Reset()
			case "t":
				inText = true
			}
		case xml.EndElement:
			switch t.Name.Local {
			case "si":
				inSI = false
				strs = append(strs, cur.String())
			case "t":
				inText = false
			}
		case xml.CharData:
			if inSI && inText {
				cur.Write(t)
			}
		}
	}
	return strs, nil
}

// readXLSXRows parses <sheetData> into a slice of rows, each a slice of cell
// values indexed by zero-based column (gaps from skipped columns are "").
func readXLSXRows(data []byte, shared []string) ([][]string, error) {
	dec := xml.NewDecoder(bytes.NewReader(data))

	var rows [][]string
	var row map[int]string
	rowMax := -1
	col := -1
	nextCol := 0
	cellType := ""
	var buf strings.Builder
	inValue := false

	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "row":
				row = map[int]string{}
				rowMax = -1
				nextCol = 0
			case "c":
				cellType = attrVal(t, "t")
				if idx := colIndex(attrVal(t, "r")); idx >= 0 {
					col = idx
				} else {
					col = nextCol
				}
				nextCol = col + 1
			case "v", "t":
				inValue = true
				buf.Reset()
			}
		case xml.EndElement:
			switch t.Name.Local {
			case "v", "t":
				inValue = false
				if col >= 0 {
					val := buf.String()
					if cellType == "s" {
						if idx, err := strconv.Atoi(val); err == nil && idx >= 0 && idx < len(shared) {
							val = shared[idx]
						}
					}
					row[col] = val
					if col > rowMax {
						rowMax = col
					}
				}
			case "row":
				line := make([]string, rowMax+1)
				for i := range line {
					line[i] = row[i]
				}
				rows = append(rows, line)
			}
		case xml.CharData:
			if inValue {
				buf.Write(t)
			}
		}
	}
	return rows, nil
}

func attrVal(t xml.StartElement, local string) string {
	for _, a := range t.Attr {
		if a.Name.Local == local {
			return a.Value
		}
	}
	return ""
}

// colIndex converts a cell reference's column letters ("C" in "C7") to a
// zero-based index, or -1 if ref is empty/unparseable.
func colIndex(ref string) int {
	letters := strings.TrimRightFunc(ref, func(r rune) bool { return r >= '0' && r <= '9' })
	if letters == "" {
		return -1
	}
	idx := 0
	for _, ch := range letters {
		if ch < 'A' || ch > 'Z' {
			return -1
		}
		idx = idx*26 + int(ch-'A'+1)
	}
	return idx - 1
}
