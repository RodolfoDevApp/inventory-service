package db

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/RodolfoDevApp/eventshop-inventory-go/internal/domain"
)

type PgOutboxRepository struct {
	db *sql.DB
}

func NewPgOutboxRepository(db *sql.DB) *PgOutboxRepository {
	return &PgOutboxRepository{db: db}
}

func (r *PgOutboxRepository) Insert(
	ctx context.Context,
	msg domain.OutboxMessage,
) error {
	if msg.ID == uuid.Nil {
		msg.ID = uuid.New()
	}
	if msg.OccurredAtUtc == 0 {
		msg.OccurredAtUtc = time.Now().UTC().Unix()
	}

	q := `
        insert into outbox_messages
        (id, type, payload_json, occurred_at_utc, retry_count, processed_at_utc)
        values ($1,$2,$3,to_timestamp($4),$5,null)
    `
	_, err := r.db.ExecContext(
		ctx, q,
		msg.ID,
		msg.Type,
		msg.PayloadJSON,
		msg.OccurredAtUtc,
		msg.RetryCount,
	)
	return err
}

func (r *PgOutboxRepository) GetPendingBatch(
	ctx context.Context,
	maxRetry, batchSize int,
) ([]domain.OutboxMessage, error) {
	q := `
        select id, type, payload_json,
               extract(epoch from occurred_at_utc) as occurred_at_sec,
               retry_count,
               processed_at_utc
        from outbox_messages
        where processed_at_utc is null
          and retry_count < $1
        order by occurred_at_utc asc
        limit $2
    `
	rows, err := r.db.QueryContext(ctx, q, maxRetry, batchSize)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.OutboxMessage
	for rows.Next() {
		var msg domain.OutboxMessage
		var processedAt sql.NullTime
		var occurredSec float64
		if err := rows.Scan(
			&msg.ID,
			&msg.Type,
			&msg.PayloadJSON,
			&occurredSec,
			&msg.RetryCount,
			&processedAt,
		); err != nil {
			return nil, err
		}
		msg.OccurredAtUtc = int64(occurredSec)
		if processedAt.Valid {
			t := processedAt.Time.Unix()
			msg.ProcessedAtUtc = &t
		}
		result = append(result, msg)
	}
	return result, nil
}

func (r *PgOutboxRepository) Save(
	ctx context.Context,
	msg domain.OutboxMessage,
) error {
	if msg.ID == uuid.Nil {
		return errors.New("outbox message id is empty")
	}

	// usamos NullFloat64 para que el driver siempre sepa el tipo del parámetro $3
	var processed sql.NullFloat64
	if msg.ProcessedAtUtc != nil {
		processed.Float64 = float64(*msg.ProcessedAtUtc) // segundos desde epoch
		processed.Valid = true
	} else {
		processed.Valid = false // manda NULL tipado (double precision)
	}

	q := `
        update outbox_messages
        set retry_count = $2,
            processed_at_utc = coalesce(to_timestamp($3), processed_at_utc)
        where id = $1
    `
	_, err := r.db.ExecContext(
		ctx, q,
		msg.ID,
		msg.RetryCount,
		processed, // <– nunca es "nil pelón", siempre tiene tipo
	)
	return err
}
