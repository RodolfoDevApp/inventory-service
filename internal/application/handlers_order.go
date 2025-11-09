package application

import (
	"context"
	"encoding/json"
	"log"

	"github.com/rodolfodevapp/eventshop-messaging-go/core/primitives"

	"github.com/google/uuid"

	"github.com/RodolfoDevApp/eventshop-inventory-go/internal/domain"
)

type EventHandler interface {
	Handle(ctx context.Context, ev primitives.Event) error
}

// OrderPlacedHandler

type OrderPlacedHandler struct {
	service *ReserveStockService
}

func NewOrderPlacedHandler(s *ReserveStockService) *OrderPlacedHandler {
	return &OrderPlacedHandler{service: s}
}

func (h *OrderPlacedHandler) Handle(ctx context.Context, ev primitives.Event) error {
	env, ok := ev.(*primitives.IntegrationEventEnvelope)
	if !ok {
		log.Printf("OrderPlacedHandler: invalid event type %T", ev)
		return nil
	}
	if env.Type != "OrderPlacedEvent" {
		return nil
	}

	var payload domain.OrderPlacedPayload
	if err := json.Unmarshal([]byte(env.PayloadJSON), &payload); err != nil {
		log.Printf("OrderPlacedHandler: failed to unmarshal payload: %v", err)
		return nil
	}

	log.Printf("OrderPlacedHandler: received orderId=%s userId=%s",
		payload.OrderID.String(), payload.UserID.String())

	return h.service.HandleOrderPlaced(ctx, payload)
}

// OrderCancelledHandler

type OrderCancelledHandler struct {
	service *ReleaseReservationService
}

func NewOrderCancelledHandler(s *ReleaseReservationService) *OrderCancelledHandler {
	return &OrderCancelledHandler{service: s}
}

func (h *OrderCancelledHandler) Handle(ctx context.Context, ev primitives.Event) error {
	env, ok := ev.(*primitives.IntegrationEventEnvelope)
	if !ok {
		log.Printf("OrderCancelledHandler: invalid event type %T", ev)
		return nil
	}
	if env.Type != "OrderCancelledEvent" && env.Type != "OrderRejectedEvent" {
		return nil
	}

	var payload domain.OrderCancelledPayload
	if err := json.Unmarshal([]byte(env.PayloadJSON), &payload); err != nil {
		log.Printf("OrderCancelledHandler: failed to unmarshal payload: %v", err)
		return nil
	}

	if payload.OrderID == uuid.Nil {
		log.Printf("OrderCancelledHandler: missing orderId")
		return nil
	}

	log.Printf("OrderCancelledHandler: releasing reservation for orderId=%s", payload.OrderID.String())
	return h.service.HandleOrderCancelled(ctx, payload.OrderID)
}
