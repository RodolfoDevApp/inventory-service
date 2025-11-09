package domain

import (
	"time"

	"github.com/google/uuid"
	"github.com/rodolfodevapp/eventshop-messaging-go/core/primitives"
)

// =========== Payloads de eventos entrantes ===========

// ProductCreated (desde catalog.events)
type ProductCreatedPayload struct {
	ProductID     uuid.UUID `json:"productId"`
	VendorID      uuid.UUID `json:"vendorId"`
	Sku           string    `json:"sku"`
	Name          string    `json:"name"`
	Slug          string    `json:"slug"`
	Price         float64   `json:"price"`
	StockQuantity int       `json:"stockQuantity"`
	CreatedAtUtc  time.Time `json:"createdAtUtc"`
	IsActive      bool      `json:"isActive"`
	Description   string    `json:"description"`
	MainImageURL  string    `json:"mainImageUrl"`
	ImageURLs     []string  `json:"imageUrls"`
}

// OrderPlaced (desde orders.events)
type OrderPlacedLine struct {
	Sku      string `json:"sku"`
	Quantity int    `json:"quantity"`
}

type OrderPlacedPayload struct {
	OrderID uuid.UUID         `json:"orderId"`
	UserID  uuid.UUID         `json:"userId"`
	Lines   []OrderPlacedLine `json:"lines"`
}

type OrderCancelledPayload struct {
	OrderID uuid.UUID `json:"orderId"`
	UserID  uuid.UUID `json:"userId"`
}

// =========== Eventos salientes Inventory -> otros ===========

// StockReserved (ya lo teníamos)
type StockReservedLine struct {
	Sku      string `json:"sku"`
	Quantity int    `json:"quantity"`
}

type StockReservedEvent struct {
	primitives.BaseEvent
	OrderID       uuid.UUID           `json:"orderId"`
	UserID        uuid.UUID           `json:"userId"`
	ReservedAtUtc time.Time           `json:"reservedAtUtc"`
	Lines         []StockReservedLine `json:"lines"`
}

func NewStockReservedEvent(orderID, userID uuid.UUID, lines []StockReservedLine) *StockReservedEvent {
	ev := &StockReservedEvent{
		BaseEvent:     primitives.NewBaseEvent(),
		OrderID:       orderID,
		UserID:        userID,
		ReservedAtUtc: time.Now().UTC(),
		Lines:         lines,
	}
	ev.SetRoutingKey("StockReserved")
	return ev
}

type StockReservationFailedEvent struct {
	primitives.BaseEvent
	OrderID     uuid.UUID `json:"orderId"`
	UserID      uuid.UUID `json:"userId"`
	Reason      string    `json:"reason"`
	FailedAtUtc time.Time `json:"failedAtUtc"`
}

func NewStockReservationFailedEvent(orderID, userID uuid.UUID, reason string) *StockReservationFailedEvent {
	ev := &StockReservationFailedEvent{
		BaseEvent:   primitives.NewBaseEvent(),
		OrderID:     orderID,
		UserID:      userID,
		Reason:      reason,
		FailedAtUtc: time.Now().UTC(),
	}
	ev.SetRoutingKey("StockReservationFailed")
	return ev
}

// CatalogStockAdjusted (evento para Catalog, Search, etc.)
type CatalogStockAdjustedEvent struct {
	primitives.BaseEvent
	Sku               string    `json:"sku"`
	AvailableQuantity int       `json:"availableQuantity"`
	ReservedQuantity  int       `json:"reservedQuantity"`
	Reason            string    `json:"reason"`
	OccurredAtUtc     time.Time `json:"occurredAtUtc"`
}

func NewCatalogStockAdjustedEvent(
	sku string,
	available, reserved int,
	reason string,
) *CatalogStockAdjustedEvent {
	ev := &CatalogStockAdjustedEvent{
		BaseEvent:         primitives.NewBaseEvent(),
		Sku:               sku,
		AvailableQuantity: available,
		ReservedQuantity:  reserved,
		Reason:            reason,
		OccurredAtUtc:     time.Now().UTC(),
	}
	ev.SetRoutingKey("CatalogStockAdjusted")
	return ev
}
