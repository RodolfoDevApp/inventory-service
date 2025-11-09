package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/RodolfoDevApp/eventshop-inventory-go/internal/domain"
)

type ReserveStockService struct {
	stockRepo       domain.StockItemRepository
	reservationRepo domain.StockReservationRepository
	outbox          OutboxWriter
}

func NewReserveStockService(
	stockRepo domain.StockItemRepository,
	reservationRepo domain.StockReservationRepository,
	outbox OutboxWriter,
) *ReserveStockService {
	return &ReserveStockService{
		stockRepo:       stockRepo,
		reservationRepo: reservationRepo,
		outbox:          outbox,
	}
}

func (s *ReserveStockService) HandleOrderPlaced(
	ctx context.Context,
	payload domain.OrderPlacedPayload,
) error {
	if payload.OrderID == uuid.Nil {
		return errors.New("missing orderId")
	}

	// Idempotencia: si ya tenemos reservación, no hacemos nada
	if existing, _ := s.reservationRepo.GetByOrderID(ctx, payload.OrderID); existing != nil {
		return nil
	}

	if len(payload.Lines) == 0 {
		reason := "No lines in order"
		ev := domain.NewStockReservationFailedEvent(payload.OrderID, payload.UserID, reason)
		return s.outbox.Enqueue(ctx, ev)
	}

	skus := make([]string, 0, len(payload.Lines))
	for _, l := range payload.Lines {
		skus = append(skus, l.Sku)
	}

	stockMap, err := s.stockRepo.GetBySkus(ctx, skus)
	if err != nil {
		return err
	}

	// Validar disponibilidad
	for _, line := range payload.Lines {
		item, ok := stockMap[line.Sku]
		if !ok {
			reason := fmt.Sprintf("SKU %s not found", line.Sku)
			ev := domain.NewStockReservationFailedEvent(payload.OrderID, payload.UserID, reason)
			return s.outbox.Enqueue(ctx, ev)
		}
		if !item.CanReserve(line.Quantity) {
			reason := fmt.Sprintf("Not enough stock for sku %s", line.Sku)
			ev := domain.NewStockReservationFailedEvent(payload.OrderID, payload.UserID, reason)
			return s.outbox.Enqueue(ctx, ev)
		}
	}

	// Aplicar reservas en memoria
	for _, line := range payload.Lines {
		item := stockMap[line.Sku]
		item.Reserve(line.Quantity)
	}

	// Construir agregados de reservación
	resLines := make([]domain.ReservationLine, 0, len(payload.Lines))
	resID := uuid.New()
	for _, line := range payload.Lines {
		resLines = append(resLines, domain.ReservationLine{
			ID:            uuid.New(),
			ReservationID: resID,
			Sku:           line.Sku,
			Quantity:      line.Quantity,
		})
	}

	reservation := &domain.StockReservation{
		ID:            resID,
		OrderID:       payload.OrderID,
		UserID:        payload.UserID,
		Status:        domain.ReservationActive,
		ReservedAtUtc: time.Now().UTC(),
		ReleasedAtUtc: nil,
		Lines:         resLines,
	}

	// Persistir stock + reservación
	items := make([]*domain.StockItem, 0, len(stockMap))
	for _, item := range stockMap {
		items = append(items, item)
	}

	if err := s.stockRepo.UpsertMany(ctx, items); err != nil {
		return err
	}
	if err := s.reservationRepo.Insert(ctx, reservation); err != nil {
		return err
	}

	// Evento StockReserved
	evLines := make([]domain.StockReservedLine, 0, len(payload.Lines))
	for _, l := range payload.Lines {
		evLines = append(evLines, domain.StockReservedLine{
			Sku:      l.Sku,
			Quantity: l.Quantity,
		})
	}
	reservedEv := domain.NewStockReservedEvent(payload.OrderID, payload.UserID, evLines)
	if err := s.outbox.Enqueue(ctx, reservedEv); err != nil {
		return err
	}

	// Eventos CatalogStockAdjusted por cada SKU afectado
	for _, item := range items {
		adjEv := domain.NewCatalogStockAdjustedEvent(
			item.Sku,
			item.Available,
			item.Reserved,
			"ORDER_RESERVED",
		)
		if err := s.outbox.Enqueue(ctx, adjEv); err != nil {
			return err
		}
	}

	return nil
}
