package api

import (
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	"go.uber.org/zap"

	"PulseSend/internal/csvparser"
	"PulseSend/internal/db"
	"PulseSend/internal/models"
)

type Handler struct {
	Store *db.Store
	Jobs  chan<- models.EmailJob
	Log   *zap.Logger
}

func (h *Handler) SendEmail(w http.ResponseWriter, r *http.Request) {
	var job models.EmailJob

	if err := json.NewDecoder(r.Body).Decode(&job); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	job.Status = models.StatusPending

	ctx := r.Context()

	if err := h.Store.InsertEmail(ctx, &job); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	select {
	case h.Jobs <- job:
	case <-ctx.Done():
		http.Error(w, "request cancelled", http.StatusRequestTimeout)
		return
	}

	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id": job.ID,
	})
}

type bulkRecipient struct {
	To   string                 `json:"to"`
	Data map[string]interface{} `json:"data"`
}

type bulkSendRequest struct {
	Subject    string          `json:"subject"`
	Template   string          `json:"template"`
	Recipients []bulkRecipient `json:"recipients"`
}

type bulkSendResult struct {
	To    string `json:"to"`
	ID    int64  `json:"id,omitempty"`
	Error string `json:"error,omitempty"`
}

// SendBulk accepts JSON with a list of recipients and queues emails.
//
// POST /send-bulk
// {
//   "subject": "Hello",
//   "template": "email.html",
//   "recipients": [
//     {"to": "a@example.com", "data": {"Name":"A"}},
//     {"to": "b@example.com", "data": {"Name":"B"}}
//   ]
// }
func (h *Handler) SendBulk(w http.ResponseWriter, r *http.Request) {
	var req bulkSendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	req.Subject = strings.TrimSpace(req.Subject)
	req.Template = strings.TrimSpace(req.Template)
	if req.Subject == "" || req.Template == "" {
		http.Error(w, "subject and template are required", http.StatusBadRequest)
		return
	}
	if len(req.Recipients) == 0 {
		http.Error(w, "recipients is required", http.StatusBadRequest)
		return
	}
	if len(req.Recipients) > 1000 {
		http.Error(w, "too many recipients (max 1000)", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	results := make([]bulkSendResult, 0, len(req.Recipients))

	for _, rcpt := range req.Recipients {
		to := strings.TrimSpace(rcpt.To)
		if to == "" {
			results = append(results, bulkSendResult{To: rcpt.To, Error: "missing to"})
			continue
		}
		if rcpt.Data == nil {
			rcpt.Data = map[string]interface{}{}
		}

		job := models.EmailJob{
			To:       to,
			Subject:  req.Subject,
			Template: req.Template,
			Data:     rcpt.Data,
			Status:   models.StatusPending,
		}

		if err := h.Store.InsertEmail(ctx, &job); err != nil {
			results = append(results, bulkSendResult{To: to, Error: err.Error()})
			continue
		}

		select {
		case h.Jobs <- job:
			results = append(results, bulkSendResult{To: to, ID: job.ID})
		case <-ctx.Done():
			results = append(results, bulkSendResult{To: to, ID: job.ID, Error: "request cancelled"})
			// keep going to return what we have
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"results": results,
	})
}

// SendBulkCSV accepts a multipart CSV upload. The CSV must have a column named "Email" (case-insensitive).
//
// POST /send-bulk/csv (multipart/form-data)
// - file: <csv file>
// - subject: <email subject>
// - template: <template filename, e.g. email.html>
func (h *Handler) SendBulkCSV(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Limit request body size (CSV uploads).
	r.Body = http.MaxBytesReader(w, r.Body, 5<<20) // 5MB

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	subject := strings.TrimSpace(r.FormValue("subject"))
	template := strings.TrimSpace(r.FormValue("template"))
	if subject == "" || template == "" {
		http.Error(w, "subject and template are required", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "missing form file field 'file'", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Optional: maxRows override
	maxRows := 1000
	if s := strings.TrimSpace(r.FormValue("max_rows")); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 && n <= 10000 {
			maxRows = n
		}
	}

	records, err := parseRecipientsCSVUpload(file, header, maxRows)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	results := make([]bulkSendResult, 0, len(records))
	for _, rec := range records {
		job := models.EmailJob{
			To:       rec.To,
			Subject:  subject,
			Template: template,
			Data:     rec.Data,
			Status:   models.StatusPending,
		}

		if err := h.Store.InsertEmail(ctx, &job); err != nil {
			results = append(results, bulkSendResult{To: rec.To, Error: err.Error()})
			continue
		}

		select {
		case h.Jobs <- job:
			results = append(results, bulkSendResult{To: rec.To, ID: job.ID})
		case <-ctx.Done():
			results = append(results, bulkSendResult{To: rec.To, ID: job.ID, Error: "request cancelled"})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"results": results,
	})
}

type csvRecipientRecord struct {
	To   string
	Data map[string]interface{}
}

func parseRecipientsCSVUpload(file multipart.File, header *multipart.FileHeader, maxRows int) ([]csvRecipientRecord, error) {
	_ = header // reserved for future validations (filename, etc.)

	// Parse CSV using a dedicated helper in csvparser.
	rows, err := csvparser.ParseRecipientRows(file, maxRows)
	if err != nil {
		return nil, err
	}

	out := make([]csvRecipientRecord, 0, len(rows))
	for _, row := range rows {
		to := strings.TrimSpace(row.Email)
		if to == "" {
			continue
		}

		data := make(map[string]interface{}, len(row.Fields))
		for k, v := range row.Fields {
			data[k] = v
		}

		out = append(out, csvRecipientRecord{To: to, Data: data})
	}

	if len(out) == 0 {
		return nil, errors.New("no valid recipients found in csv (needs an Email column and at least one row)")
	}

	return out, nil
}
