package csvparser

import (
	"PulseSend/internal/models"
	"encoding/csv"
	"errors"
	"os"
)

func Parse(path string) ([]models.EmailJob, error) {

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	reader := csv.NewReader(f)

	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	if len(records) < 2 {
		return nil, errors.New("csv must contain header and at least one row")
	}

	headers := records[0]

	var emailJobs []models.EmailJob

	for _, row := range records[1:] {

		if len(row) != len(headers) {
			continue // skip malformed row
		}

		job := models.EmailJob{
			To:       row[0],
			Subject:  row[1],
			Template: row[2],
			Data:     make(map[string]interface{}),
			Status:   models.StatusPending,
		}

		// Everything after first 3 columns becomes template data
		for i := 3; i < len(headers); i++ {
			job.Data[headers[i]] = row[i]
		}

		emailJobs = append(emailJobs, job)
	}

	return emailJobs, nil
}
