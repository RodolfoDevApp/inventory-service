package application

import (
	"context"
	"encoding/json"
	"log"

	"github.com/RodolfoDevApp/eventshop-inventory-go/internal/domain"
	"github.com/rodolfodevapp/eventshop-messaging-go/core/primitives"
)

type ProductCreatedHandler struct {
	stockRepo domain.StockItemRepository
	outbox    OutboxWriter
}

func NewProductCreatedHandler(
	stockRepo domain.StockItemRepository,
	outbox OutboxWriter,
) *ProductCreatedHandler {
	return &ProductCreatedHandler{
		stockRepo: stockRepo,
		outbox:    outbox,
	}
}

func (h *ProductCreatedHandler) Handle(ctx context.Context, ev primitives.Event) error {
	env, ok := ev.(*primitives.IntegrationEventEnvelope)
	if !ok {
		log.Printf("ProductCreatedHandler: invalid event type %T", ev)
		return nil
	}
	if env.Type != "ProductCreated" {
		return nil
	}

	var payload domain.ProductCreatedPayload
	if err := json.Unmarshal([]byte(env.PayloadJSON), &payload); err != nil {
		log.Printf("ProductCreatedHandler: failed to unmarshal payload: %v", err)
		return nil
	}

	if payload.Sku == "" {
		log.Printf("ProductCreatedHandler: missing sku")
		return nil
	}

	log.Printf("ProductCreatedHandler: received ProductCreated for sku=%s qty=%d",
		payload.Sku, payload.StockQuantity)

	skus := []string{payload.Sku}
	existing, err := h.stockRepo.GetBySkus(ctx, skus)
	if err != nil {
		return err
	}

	var item *domain.StockItem
	if current, ok := existing[payload.Sku]; ok {
		// si ya existe, sobreescribimos available por el inicial (policy)
		current.Available = payload.StockQuantity
		item = current
	} else {
		item = domain.NewStockItem(payload.Sku, payload.StockQuantity)
	}

	if err := h.stockRepo.UpsertMany(ctx, []*domain.StockItem{item}); err != nil {
		return err
	}

	adjEv := domain.NewCatalogStockAdjustedEvent(
		item.Sku,
		item.Available,
		item.Reserved,
		"INITIAL_LOAD",
	)
	return h.outbox.Enqueue(ctx, adjEv)
}
