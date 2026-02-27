package db

import (
	"context"
	"database/sql"
	"encoding/json"

	"PulseSend/internal/models"

	_ "github.com/mattn/go-sqlite3"
)

type Store struct {
	DB *sql.DB
}

func New(conn string) (*Store, error) {
	// For SQLite, conn is typically a file path, e.g. "./pulsesend.db"
	db, err := sql.Open("sqlite3", conn)
	if err != nil {
		return nil, err
	}

	// SQLite is file-based; keep connections small/simple.
	db.SetMaxOpenConns(1)

	// Ensure schema exists.
	schema := `
	CREATE TABLE IF NOT EXISTS email_jobs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		to_email   TEXT NOT NULL,
		subject    TEXT NOT NULL,
		template   TEXT NOT NULL,
		data       TEXT NOT NULL,
		status     TEXT NOT NULL,
		retries    INTEGER NOT NULL DEFAULT 0,
		error_msg  TEXT,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);`

	if _, err := db.Exec(schema); err != nil {
		_ = db.Close()
		return nil, err
	}

	return &Store{DB: db}, nil
}

func (s *Store) Close() {
	if s.DB != nil {
		_ = s.DB.Close()
	}
}

func (s *Store) InsertEmail(ctx context.Context, job *models.EmailJob) error {
	dataJSON, err := json.Marshal(job.Data)
	if err != nil {
		return err
	}

	res, err := s.DB.ExecContext(
		ctx,
		`INSERT INTO email_jobs 
		 (to_email, subject, template, data, status, retries, created_at, updated_at)
		 VALUES (?,?,?,?,?,0,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)`,
		job.To,
		job.Subject,
		job.Template,
		string(dataJSON),
		models.StatusPending,
	)
	if err != nil {
		return err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return err
	}

	job.ID = id
	return nil
}

func (s *Store) UpdateStatus(
	ctx context.Context,
	id int64,
	status models.EmailStatus,
) error {
	_, err := s.DB.ExecContext(
		ctx,
		`UPDATE email_jobs
		 SET status = ?,
		     updated_at = CURRENT_TIMESTAMP
		 WHERE id = ?`,
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
	_, err := s.DB.ExecContext(
		ctx,
		`UPDATE email_jobs
		 SET status = ?,
		     retries = retries + 1,
		     error_msg = ?,
		     updated_at = CURRENT_TIMESTAMP
		 WHERE id = ?`,
		models.StatusFailed,
		errorMsg,
		id,
	)
	return err
}
