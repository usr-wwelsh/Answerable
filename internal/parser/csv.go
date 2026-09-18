package parser

import (
	"encoding/csv"
	"fmt"
	"io"
)

type Record map[string]string

func ParseCSV(r io.Reader) ([]Record, error) {
	cr := csv.NewReader(r)

	header, err := cr.Read()
	if err != nil {
		return nil, fmt.Errorf("read csv header: %w", err)
	}

	var records []Record
	for {
		row, err := cr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read csv row: %w", err)
		}

		rec := make(Record, len(header))
		for i, key := range header {
			rec[key] = row[i]
		}
		records = append(records, rec)
	}

	return records, nil
}
