package email

import (
	"PulseSend/internal/models"
	"bytes"
	"context"
	"fmt"
	"html/template"
	"path/filepath"
	"time"

	"github.com/cenkalti/backoff/v4"
	"gopkg.in/gomail.v2"
)

type Sender struct {
	Host string
	Port int
	From string
}

// Send renders the template and sends the email
func (s *Sender) Send(job models.EmailJob) error {

	// Build template path safely
	templatePath := filepath.Join("templates", job.Template)

	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return fmt.Errorf("template parse error: %w", err)
	}

	var body bytes.Buffer

	// Execute template with dynamic data
	if err := tmpl.Execute(&body, job.Data); err != nil {
		return fmt.Errorf("template execution error: %w", err)
	}

	m := gomail.NewMessage()
	m.SetHeader("From", s.From)
	m.SetHeader("To", job.To)
	m.SetHeader("Subject", job.Subject)
	m.SetBody("text/html", body.String())

	d := gomail.NewDialer(s.Host, s.Port, "", "")

	if err := d.DialAndSend(m); err != nil {
		return fmt.Errorf("smtp send error: %w", err)
	}

	return nil
}

// SendWithRetry retries email sending with exponential backoff
func (s *Sender) SendWithRetry(
	ctx context.Context,
	job models.EmailJob,
	retries int,
) error {

	operation := func() error {
		return s.Send(job)
	}

	b := backoff.NewExponentialBackOff()
	b.InitialInterval = 500 * time.Millisecond
	b.MaxElapsedTime = time.Duration(retries) * time.Second

	return backoff.Retry(operation, backoff.WithContext(b, ctx))
}
