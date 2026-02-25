package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"go.uber.org/zap"
	"golang.org/x/time/rate"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"PulseSend/internal/api"
	"PulseSend/internal/config"
	"PulseSend/internal/db"
	"PulseSend/internal/email"
	"PulseSend/internal/metrics"
	"PulseSend/internal/models"
	"PulseSend/internal/worker"
)

func main() {

	// ------------------------------------------------
	// Logger
	// ------------------------------------------------
	logger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}
	defer logger.Sync()

	// ------------------------------------------------
	// Config
	// ------------------------------------------------
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal("failed to load config", zap.Error(err))
	}

	// ------------------------------------------------
	// Root Context + Shutdown
	// ------------------------------------------------
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		logger.Info("shutdown signal received", zap.String("signal", sig.String()))
		cancel()
	}()

	// ------------------------------------------------
	// Database
	// ------------------------------------------------
	store, err := db.New(cfg.DatabaseURL)
	if err != nil {
		logger.Fatal("database connection failed", zap.Error(err))
	}
	defer store.Pool.Close()

	// ------------------------------------------------
	// Metrics
	// ------------------------------------------------
	metrics.Init()

	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.Handler())

	metricsServer := &http.Server{
		Addr:    ":" + cfg.MetricsPort,
		Handler: metricsMux,
	}

	go func() {
		logger.Info("metrics server started", zap.String("port", cfg.MetricsPort))
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("metrics server error", zap.Error(err))
		}
	}()

	// ------------------------------------------------
	// Job Channel (shared by API + workers)
	// ------------------------------------------------
	jobs := make(chan models.EmailJob, 100)

	// ------------------------------------------------
	// Email Sender
	// ------------------------------------------------
	sender := &email.Sender{
		Host: cfg.SMTPHost,
		Port: cfg.SMTPPort,
		From: "noreply@pulsesend.com",
	}

	// ------------------------------------------------
	// Rate Limiter
	// ------------------------------------------------
	limiter := rate.NewLimiter(rate.Limit(cfg.RateLimit), cfg.RateLimit)

	// ------------------------------------------------
	// Worker Pool
	// ------------------------------------------------
	var wg sync.WaitGroup

	worker.StartPool(
		ctx,
		&wg,
		cfg.WorkerCount,
		jobs,
		sender,
		limiter,
		logger,
		cfg.RetryAttempts,
		store, // pass DB to update status
	)

	// ------------------------------------------------
	// HTTP API Server
	// ------------------------------------------------
	apiHandler := &api.Handler{
		Store: store,
		Jobs:  jobs,
		Log:   logger,
	}

	apiMux := http.NewServeMux()
	apiMux.HandleFunc("/send", apiHandler.SendEmail)

	apiServer := &http.Server{
		Addr:    ":8080",
		Handler: apiMux,
	}

	go func() {
		logger.Info("api server started", zap.String("port", "8080"))
		if err := apiServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("api server error", zap.Error(err))
		}
	}()

	// ------------------------------------------------
	// Wait for shutdown
	// ------------------------------------------------
	<-ctx.Done()

	logger.Info("shutting down services...")

	// Stop accepting new jobs
	close(jobs)

	// Wait workers to finish
	wg.Wait()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := apiServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("api shutdown failed", zap.Error(err))
	}

	if err := metricsServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("metrics shutdown failed", zap.Error(err))
	}

	logger.Info("application shutdown complete")
}
