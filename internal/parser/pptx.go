package parser

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var pptxSlideRe = regexp.MustCompile(`^ppt/slides/slide(\d+)\.xml$`)

// ParsePPTX does best-effort text extraction from a PowerPoint deck: it reads
// every ppt/slides/slideN.xml in slide order and concatenates text-run
// content, then feeds the result through ParseText's "Key: value" line
// scanner. It does not understand layout, tables, or speaker notes.
func ParsePPTX(r io.Reader) (Record, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("open pptx: %w", err)
	}

	slides := slideFiles(zr)
	if len(slides) == 0 {
		return nil, fmt.Errorf("pptx: no slides found")
	}

	var out strings.Builder
	for _, f := range slides {
		xmlData, err := readZipFile(f)
		if err != nil {
			return nil, err
		}
		out.WriteString(extractOOXMLText(xmlData))
		out.WriteByte('\n')
	}

	return ParseText(strings.NewReader(out.String()))
}

func slideFiles(zr *zip.Reader) []*zip.File {
	var slides []*zip.File
	for _, f := range zr.File {
		if pptxSlideRe.MatchString(f.Name) {
			slides = append(slides, f)
		}
	}
	sort.Slice(slides, func(i, j int) bool {
		ni, _ := strconv.Atoi(pptxSlideRe.FindStringSubmatch(slides[i].Name)[1])
		nj, _ := strconv.Atoi(pptxSlideRe.FindStringSubmatch(slides[j].Name)[1])
		return ni < nj
	})
	return slides
}
