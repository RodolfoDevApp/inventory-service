package domain

import (
	"time"

	"github.com/google/uuid"
)

type ReservationStatus string

const (
	ReservationActive   ReservationStatus = "ACTIVE"
	ReservationReleased ReservationStatus = "RELEASED"
)

type ReservationLine struct {
	ID            uuid.UUID
	ReservationID uuid.UUID
	Sku           string
	Quantity      int
}

type StockReservation struct {
	ID            uuid.UUID
	OrderID       uuid.UUID
	UserID        uuid.UUID
	Status        ReservationStatus
	ReservedAtUtc time.Time
	ReleasedAtUtc *time.Time
	Lines         []ReservationLine
}

func NewStockReservation(orderID, userID uuid.UUID, lines []ReservationLine) *StockReservation {
	now := time.Now().UTC()
	return &StockReservation{
		ID:            uuid.New(),
		OrderID:       orderID,
		UserID:        userID,
		Status:        ReservationActive,
		ReservedAtUtc: now,
		ReleasedAtUtc: nil,
		Lines:         lines,
	}
}

func (r *StockReservation) MarkReleased() {
	if r.Status == ReservationReleased {
		return
	}
	now := time.Now().UTC()
	r.Status = ReservationReleased
	r.ReleasedAtUtc = &now
}
