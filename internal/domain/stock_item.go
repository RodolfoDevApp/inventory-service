package domain

import (
	"time"

	"github.com/google/uuid"
)

type StockItem struct {
	ID           uuid.UUID
	Sku          string
	Available    int
	Reserved     int
	UpdatedAtUtc time.Time
}

func NewStockItem(sku string, available int) *StockItem {
	return &StockItem{
		ID:           uuid.New(),
		Sku:          sku,
		Available:    available,
		Reserved:     0,
		UpdatedAtUtc: time.Now().UTC(),
	}
}

func (s *StockItem) CanReserve(qty int) bool {
	return qty > 0 && s.Available >= qty
}

func (s *StockItem) Reserve(qty int) {
	s.Available -= qty
	s.Reserved += qty
	s.UpdatedAtUtc = time.Now().UTC()
}

func (s *StockItem) Release(qty int) {
	s.Available += qty
	if s.Reserved >= qty {
		s.Reserved -= qty
	}
	s.UpdatedAtUtc = time.Now().UTC()
}
