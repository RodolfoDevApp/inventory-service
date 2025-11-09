package domain

import (
	"context"

	"github.com/google/uuid"
)

type StockItemRepository interface {
	GetBySkus(ctx context.Context, skus []string) (map[string]*StockItem, error)
	UpsertMany(ctx context.Context, items []*StockItem) error
}

type StockReservationRepository interface {
	GetByOrderID(ctx context.Context, orderID uuid.UUID) (*StockReservation, error)
	Insert(ctx context.Context, r *StockReservation) error
	Update(ctx context.Context, r *StockReservation) error
}

type OutboxRepository interface {
	Insert(ctx context.Context, msg OutboxMessage) error
	GetPendingBatch(ctx context.Context, maxRetry, batchSize int) ([]OutboxMessage, error)
	Save(ctx context.Context, msg OutboxMessage) error
}

type OutboxMessage struct {
	ID             uuid.UUID
	Type           string
	PayloadJSON    string
	OccurredAtUtc  int64 // unix nano or seconds
	RetryCount     int
	ProcessedAtUtc *int64
}
