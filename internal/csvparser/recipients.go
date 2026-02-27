package csvparser

import (
	"encoding/csv"
	"errors"
	"io"
	"strings"
)

// RecipientRow represents a single recipient extracted from a CSV.
// Email is taken from the "Email" column (case-insensitive).
// Fields contains all other columns (header -> value) for template data.
type RecipientRow struct {
	Email  string
	Fields map[string]string
}

// ParseRecipientRows parses a CSV from an io.Reader. The CSV must contain a header row
// with an "Email" column (case-insensitive). All other columns are returned as Fields.
//
// maxRows limits how many data rows are parsed (excluding header).
func ParseRecipientRows(r io.Reader, maxRows int) ([]RecipientRow, error) {
	reader := csv.NewReader(r)
	reader.TrimLeadingSpace = true

	headers, err := reader.Read()
	if err != nil {
		return nil, err
	}
	if len(headers) == 0 {
		return nil, errors.New("csv header row is empty")
	}

	emailIdx := -1
	normalized := make([]string, len(headers))
	for i, h := range headers {
		h = strings.TrimSpace(h)
		normalized[i] = h
		if strings.EqualFold(h, "email") {
			emailIdx = i
		}
	}
	if emailIdx == -1 {
		return nil, errors.New("csv must contain an Email column")
	}

	if maxRows <= 0 {
		maxRows = 1000
	}

	rows := make([]RecipientRow, 0)
	for len(rows) < maxRows {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if len(record) != len(headers) {
			// skip malformed row
			continue
		}

		email := strings.TrimSpace(record[emailIdx])
		if email == "" {
			continue
		}

		fields := make(map[string]string, len(headers)-1)
		for i := range record {
			if i == emailIdx {
				continue
			}
			key := normalized[i]
			if key == "" {
				continue
			}
			fields[key] = strings.TrimSpace(record[i])
		}

		rows = append(rows, RecipientRow{
			Email:  email,
			Fields: fields,
		})
	}

	if len(rows) == 0 {
		return nil, errors.New("csv must contain at least one data row")
	}

	return rows, nil
}

