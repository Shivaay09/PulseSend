package worker

import (
	"context"
	"sync"

	"go.uber.org/zap"
	"golang.org/x/time/rate"

	"PulseSend/internal/db"
	"PulseSend/internal/email"
	"PulseSend/internal/metrics"
	"PulseSend/internal/models"
)

func StartPool(
	ctx context.Context,
	wg *sync.WaitGroup,
	workers int,
	jobs <-chan models.EmailJob,
	sender *email.Sender,
	limiter *rate.Limiter,
	store *db.Store,
	logger *zap.Logger,
	retries int,
) {

	for i := 0; i < workers; i++ {
		wg.Add(1)

		go func(id int) {
			defer wg.Done()

			logger.Info("worker started", zap.Int("worker_id", id))

			for {
				select {

				case <-ctx.Done():
					logger.Info("worker shutting down", zap.Int("worker_id", id))
					return

				case job, ok := <-jobs:
					if !ok {
						logger.Info("job channel closed", zap.Int("worker_id", id))
						return
					}

					// ----------------------------
					// Rate Limit
					// ----------------------------
					if err := limiter.Wait(ctx); err != nil {
						logger.Warn("rate limiter stopped by context",
							zap.Int("worker_id", id),
							zap.Error(err),
						)
						return
					}

					// ----------------------------
					// Mark as Processing
					// ----------------------------
					if err := store.UpdateStatus(ctx, job.ID, models.StatusProcessing); err != nil {
						logger.Error("failed to update status to processing",
							zap.Int64("job_id", job.ID),
							zap.Error(err),
						)
						continue
					}

					// ----------------------------
					// Send Email
					// ----------------------------
					err := sender.SendWithRetry(ctx, job, retries)
					if err != nil {

						logger.Error("email send failed",
							zap.Int("worker_id", id),
							zap.String("to", job.To),
							zap.Error(err),
						)

						if dbErr := store.UpdateFailure(ctx, job.ID, err.Error()); dbErr != nil {
							logger.Error("failed to update failure status",
								zap.Int64("job_id", job.ID),
								zap.Error(dbErr),
							)
						}

						metrics.EmailFailures.Inc()
						continue
					}

					// ----------------------------
					// Mark as Sent
					// ----------------------------
					if err := store.UpdateStatus(ctx, job.ID, models.StatusSent); err != nil {
						logger.Error("failed to update sent status",
							zap.Int64("job_id", job.ID),
							zap.Error(err),
						)
					}

					logger.Info("email sent successfully",
						zap.Int("worker_id", id),
						zap.String("to", job.To),
					)

					metrics.EmailsSent.Inc()
				}
			}
		}(i)
	}
}
