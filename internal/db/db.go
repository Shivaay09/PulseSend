package db

import (
	"context"
	"encoding/json"

	"PulseSend/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	Pool *pgxpool.Pool
}

func New(conn string) (*Store, error) {
	pool, err := pgxpool.New(context.Background(), conn)
	if err != nil {
		return nil, err
	}

	return &Store{Pool: pool}, nil
}

func (s *Store) Close() {
	s.Pool.Close()
}

func (s *Store) InsertEmail(ctx context.Context, job *models.EmailJob) error {

	dataJSON, err := json.Marshal(job.Data)
	if err != nil {
		return err
	}

	return s.Pool.QueryRow(ctx,
		`INSERT INTO email_jobs 
		 (to_email, subject, template, data, status, retries, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,0,NOW(),NOW())
		 RETURNING id`,
		job.To,
		job.Subject,
		job.Template,
		dataJSON,
		models.StatusPending,
	).Scan(&job.ID)
}

func (s *Store) UpdateStatus(
	ctx context.Context,
	id int64,
	status models.EmailStatus,
) error {

	_, err := s.Pool.Exec(ctx,
		`UPDATE email_jobs
		 SET status=$1,
		     updated_at=NOW()
		 WHERE id=$2`,
		status,
		id,
	)

	return err
}

func (s *Store) UpdateFailure(
	ctx context.Context,
	id int64,
	errorMsg string,
) error {

	_, err := s.Pool.Exec(ctx,
		`UPDATE email_jobs
		 SET status=$1,
		     retries = retries + 1,
		     error_msg=$2,
		     updated_at=NOW()
		 WHERE id=$3`,
		models.StatusFailed,
		errorMsg,
		id,
	)

	return err
}
