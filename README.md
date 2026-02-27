## PulseSend

PulseSend is a lightweight email job queue and sender written in Go.  
It exposes a simple HTTP API for enqueueing emails, processes them with a worker pool, and reports Prometheus metrics. It supports both single and bulk sends (JSON and CSV) and can run either locally or fully containerized with Mailpit for local testing.

---

### Features

- **SQLite-backed job queue**  
  - Simple file-based storage (`pulsesend.db`) with auto-migrated schema.
- **Worker pool with retries**  
  - Configurable worker count, rate limiting, and retry attempts.
- **SMTP integration**  
  - Works with Mailpit for local dev and real SMTP (e.g. Gmail, SES, SendGrid) in production.
- **Bulk sending**  
  - JSON endpoint for multiple recipients.
  - CSV upload endpoint that maps columns to template data.
- **Templated emails**  
  - HTML templates rendered with dynamic data.
- **Metrics**  
  - Prometheus metrics at `/metrics` (emails sent, failures, etc.).
- **Dockerized**  
  - `Dockerfile` and `docker-compose.yml` for one-command local stack.

---

### Configuration

PulseSend is configured via environment variables (see `.env`):
