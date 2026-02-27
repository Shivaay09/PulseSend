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

env
Database (SQLite file path)
DATABASE_URL=./pulsesend.db
SMTP
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=your-email@gmail.com
SMTP_PASSWORD=your-app-password
SMTP_FROM=your-email@gmail.com
Workers
WORKER_COUNT=5
RATE_LIMIT=10
RETRY_ATTEMPTS=3
HTTP
API_PORT=8080
METRICS_PORT=9090
> For local dev with Mailpit, set `SMTP_HOST=localhost`, `SMTP_PORT=1025` and run Mailpit separately.---### Running locally (no Docker)1. Ensure Go 1.25+ is installed.2. Create `.env` in the project root (see example above).3. Start the server:
bash
set -a
source .env
set +a
go run ./cmd/server
Endpoints:- API: `http://localhost:8080`- Metrics: `http://localhost:9090/metrics`---### Running with Docker + MailpitThe included `docker-compose.yml` spins up:- `app`: PulseSend (Go service)- `mailpit`: SMTP sink + UIFrom the project root:docker compose up --build
Then:
API: http://localhost:8080
Metrics: http://localhost:9090/metrics
Mailpit UI (see all emails): http://localhost:8025
API
1. POST /send – single email
Request
POST /sendContent-Type: application/json
{  "to": "recipient@example.com",  "subject": "Welcome to PulseSend",  "template": "email.html",  "data": {    "Name": "Shivam",    "City": "Delhi",    "Discount": "20%"  }}
Response
{  "id": 1}
The job is stored in the DB, then picked up and sent by workers in the background.
2. POST /send-bulk – bulk JSON
Send the same subject/template to multiple recipients, each with its own data.
POST /send-bulkContent-Type: application/json
{  "subject": "Bulk test from PulseSend",  "template": "email.html",  "recipients": [    {      "to": "person1@example.com",      "data": { "Name": "Person 1", "City": "Delhi", "Discount": "10%" }    },    {      "to": "person2@example.com",      "data": { "Name": "Person 2", "City": "Mumbai", "Discount": "15%" }    }  ]}
Response
{  "results": [    { "to": "person1@example.com", "id": 1 },    { "to": "person2@example.com", "id": 2 }  ]}
3. POST /send-bulk/csv – bulk via CSV upload
Upload a CSV where:
There must be an Email column (case-insensitive).
All other columns become template variables.
Example recipients.csv:
Name,Email,City,DiscountShivam,shivam@example.com,Delhi,20%Riya,riya@example.com,Mumbai,15%
Request (multipart/form-data)
POST /send-bulk/csvContent-Type: multipart/form-data
Form fields:
file: CSV file
subject: email subject
template: template filename (e.g. email.html)
max_rows (optional): numeric limit, default 1000
Response
{  "results": [    { "to": "shivam@example.com", "id": 1 },    { "to": "riya@example.com", "id": 2 }  ]}
Metrics
Prometheus-style metrics are exposed at:
GET /metrics
Some example metrics:
emails_sent_total
email_failures_total
These can be scraped by Prometheus / Grafana for dashboards and alerts.
Development notes
Database: SQLite is used for simplicity; swapping to Postgres/MySQL can be done by replacing the internal/db implementation.
Templates: HTML templates live in templates/. job.Template must match a file name there.
Safety:
Basic rate limiting via golang.org/x/time/rate.
Worker recovery from panics.
Timeouts per email send and for HTTP handlers.
CSV upload size and row counts are bounded.
License
PulseSend is released under the MIT License.
