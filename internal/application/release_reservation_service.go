package application

import (
	"context"

	"github.com/google/uuid"

	"github.com/RodolfoDevApp/eventshop-inventory-go/internal/domain"
)

type ReleaseReservationService struct {
	stockRepo       domain.StockItemRepository
	reservationRepo domain.StockReservationRepository
	outbox          OutboxWriter
}

func NewReleaseReservationService(
	stockRepo domain.StockItemRepository,
	reservationRepo domain.StockReservationRepository,
	outbox OutboxWriter,
) *ReleaseReservationService {
	return &ReleaseReservationService{
		stockRepo:       stockRepo,
		reservationRepo: reservationRepo,
		outbox:          outbox,
	}
}

func (s *ReleaseReservationService) HandleOrderCancelled(
	ctx context.Context,
	orderID uuid.UUID,
) error {
	res, err := s.reservationRepo.GetByOrderID(ctx, orderID)
	if err != nil {
		return err
	}
	if res == nil || res.Status == domain.ReservationReleased {
		// idempotente
		return nil
	}

	// Cargar stock items
	skus := make([]string, 0, len(res.Lines))
	for _, l := range res.Lines {
		skus = append(skus, l.Sku)
	}

	stockMap, err := s.stockRepo.GetBySkus(ctx, skus)
	if err != nil {
		return err
	}

	// Liberar a inventario
	for _, l := range res.Lines {
		item, ok := stockMap[l.Sku]
		if !ok {
			continue
		}
		item.Release(l.Quantity)
	}

	items := make([]*domain.StockItem, 0, len(stockMap))
	for _, it := range stockMap {
		items = append(items, it)
	}

	if err := s.stockRepo.UpsertMany(ctx, items); err != nil {
		return err
	}

	res.MarkReleased()
	if err := s.reservationRepo.Update(ctx, res); err != nil {
		return err
	}

	// Emitir CatalogStockAdjusted por cada SKU afectado
	for _, item := range items {
		adjEv := domain.NewCatalogStockAdjustedEvent(
			item.Sku,
			item.Available,
			item.Reserved,
			"ORDER_RELEASED",
		)
		if err := s.outbox.Enqueue(ctx, adjEv); err != nil {
			return err
		}
	}

	return nil
}
