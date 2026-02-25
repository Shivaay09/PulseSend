package api

import (
	"context"
	"encoding/json"
	"net/http"

	"go.uber.org/zap"

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

	ctx := context.Background()

	if err := h.Store.InsertEmail(ctx, &job); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.Jobs <- job

	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"id": job.ID,
	})
}
