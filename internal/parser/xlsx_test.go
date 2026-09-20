package parser

import (
	"bytes"
	"testing"
)

func TestParseXLSXReadsHeaderAndFirstDataRow(t *testing.T) {
	sharedStrings := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<sst xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" count="4" uniqueCount="4">
  <si><t>name</t></si>
  <si><t>beds</t></si>
  <si><t>Test Shelter</t></si>
</sst>`
	sheet1 := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">
  <sheetData>
    <row r="1"><c r="A1" t="s"><v>0</v></c><c r="B1" t="s"><v>1</v></c></row>
    <row r="2"><c r="A2" t="s"><v>2</v></c><c r="B2"><v>12</v></c></row>
  </sheetData>
</worksheet>`

	xlsx := buildZip(t, map[string]string{
		"xl/sharedStrings.xml":     sharedStrings,
		"xl/worksheets/sheet1.xml": sheet1,
	})

	rec, err := ParseXLSX(bytes.NewReader(xlsx))
	if err != nil {
		t.Fatalf("ParseXLSX: %v", err)
	}
	if rec["name"] != "Test Shelter" {
		t.Errorf("name = %q, want %q", rec["name"], "Test Shelter")
	}
	if rec["beds"] != "12" {
		t.Errorf("beds = %q, want %q", rec["beds"], "12")
	}
}

func TestParseXLSXRejectsSingleRowSheet(t *testing.T) {
	sheet1 := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">
  <sheetData>
    <row r="1"><c r="A1"><v>only-header</v></c></row>
  </sheetData>
</worksheet>`

	xlsx := buildZip(t, map[string]string{"xl/worksheets/sheet1.xml": sheet1})

	if _, err := ParseXLSX(bytes.NewReader(xlsx)); err == nil {
		t.Fatal("expected error for sheet with no data rows")
	}
}

func TestParseXLSXHandlesSparseColumns(t *testing.T) {
	sheet1 := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main">
  <sheetData>
    <row r="1"><c r="A1"><v>name</v></c><c r="C1"><v>beds</v></c></row>
    <row r="2"><c r="A2"><v>Test Shelter</v></c><c r="C2"><v>12</v></c></row>
  </sheetData>
</worksheet>`

	xlsx := buildZip(t, map[string]string{"xl/worksheets/sheet1.xml": sheet1})

	rec, err := ParseXLSX(bytes.NewReader(xlsx))
	if err != nil {
		t.Fatalf("ParseXLSX: %v", err)
	}
	if rec["name"] != "Test Shelter" || rec["beds"] != "12" {
		t.Fatalf("rec = %+v", rec)
	}
}
