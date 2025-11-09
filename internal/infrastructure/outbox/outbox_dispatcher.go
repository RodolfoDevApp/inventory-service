package outbox

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/rodolfodevapp/eventshop-messaging-go/core/abstractions"
	"github.com/rodolfodevapp/eventshop-messaging-go/core/primitives"

	"github.com/RodolfoDevApp/eventshop-inventory-go/internal/domain"
)

type Dispatcher struct {
	repo      domain.OutboxRepository
	eventBus  abstractions.EventBus
	maxRetry  int
	batchSize int
}

func NewDispatcher(
	repo domain.OutboxRepository,
	eventBus abstractions.EventBus,
	maxRetry, batchSize int,
) *Dispatcher {
	return &Dispatcher{
		repo:      repo,
		eventBus:  eventBus,
		maxRetry:  maxRetry,
		batchSize: batchSize,
	}
}

func (d *Dispatcher) DispatchOnce(ctx context.Context) (int, error) {
	msgs, err := d.repo.GetPendingBatch(ctx, d.maxRetry, d.batchSize)
	if err != nil {
		return 0, err
	}
	if len(msgs) == 0 {
		return 0, nil
	}

	processed := 0
	for i := range msgs {
		msg := &msgs[i]

		var generic map[string]interface{}
		if err := json.Unmarshal([]byte(msg.PayloadJSON), &generic); err != nil {
			log.Printf("Outbox: failed to unmarshal payload: %v", err)
			msg.RetryCount++
			if err := d.repo.Save(ctx, *msg); err != nil {
				log.Printf("Outbox: failed to save message: %v", err)
			}
			continue
		}

		eventType := msg.Type // ej. "StockReserved" / "StockReservationFailed"
		payloadStr := msg.PayloadJSON

		// Envelope estándar
		envelope := primitives.NewIntegrationEventEnvelope(eventType, payloadStr)

		envelope.SetRoutingKey(eventType)

		if err := d.eventBus.Publish(ctx, &envelope); err != nil {
			log.Printf("Outbox: failed to publish %s: %v", msg.Type, err)
			msg.RetryCount++
		} else {
			now := time.Now().UTC().Unix()
			msg.ProcessedAtUtc = &now
			processed++
		}

		if err := d.repo.Save(ctx, *msg); err != nil {
			log.Printf("Outbox: failed to save message: %v", err)
		}
	}

	return processed, nil
}
