package config

import "github.com/kelseyhightower/envconfig"

type Config struct {
	// ----------------------------
	// SMTP
	// ----------------------------
	SMTPHost     string `envconfig:"SMTP_HOST" default:"localhost"`
	SMTPPort     int    `envconfig:"SMTP_PORT" default:"1025"`
	SMTPUser     string `envconfig:"SMTP_USER" default:""`
	SMTPPassword string `envconfig:"SMTP_PASSWORD" default:""`
	SMTPFrom     string `envconfig:"SMTP_FROM" default:"noreply@pulsesend.com"`

	// ----------------------------
	// Workers
	// ----------------------------
	WorkerCount   int `envconfig:"WORKER_COUNT" default:"5"`
	RateLimit     int `envconfig:"RATE_LIMIT" default:"10"`
	RetryAttempts int `envconfig:"RETRY_ATTEMPTS" default:"3"`

	// ----------------------------
	// HTTP API
	// ----------------------------
	APIPort string `envconfig:"API_PORT" default:"8080"`

	// ----------------------------
	// Metrics
	// ----------------------------
	MetricsPort string `envconfig:"METRICS_PORT" default:"9090"`

	// ----------------------------
	// Database
	// ----------------------------
	DatabaseURL string `envconfig:"DATABASE_URL" required:"true"`
}

func Load() (*Config, error) {
	var cfg Config
	err := envconfig.Process("", &cfg)
	return &cfg, err
}
