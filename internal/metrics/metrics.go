package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	EmailsSent = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "emails_sent_total",
			Help: "Total emails sent",
		},
	)

	EmailFailures = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "email_failures_total",
			Help: "Total failed emails",
		},
	)
)

func Init() {
	prometheus.MustRegister(EmailsSent)
	prometheus.MustRegister(EmailFailures)
}
