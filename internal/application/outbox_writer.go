package application

import (
	"context"
	"encoding/json"
	"reflect"
	"time"

	"github.com/google/uuid"
	"github.com/rodolfodevapp/eventshop-messaging-go/core/primitives"

	"github.com/RodolfoDevApp/eventshop-inventory-go/internal/domain"
)

type OutboxWriter interface {
	Enqueue(ctx context.Context, ev primitives.Event) error
}

type outboxWriter struct {
	repo domain.OutboxRepository
}

func NewOutboxWriter(repo domain.OutboxRepository) OutboxWriter {
	return &outboxWriter{repo: repo}
}

func (w *outboxWriter) Enqueue(ctx context.Context, ev primitives.Event) error {
	payload, err := json.Marshal(ev)
	if err != nil {
		return err
	}

	eventType := ev.GetRoutingKey()
	if eventType == "" {
		eventType = typeNameOf(ev)
	}

	now := time.Now().UTC().Unix()
	msg := domain.OutboxMessage{
		ID:             uuid.New(),
		Type:           eventType,
		PayloadJSON:    string(payload),
		OccurredAtUtc:  now,
		RetryCount:     0,
		ProcessedAtUtc: nil,
	}
	return w.repo.Insert(ctx, msg)
}

func typeNameOf(ev primitives.Event) string {
	if ev == nil {
		return ""
	}
	t := reflect.TypeOf(ev)
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t.Name()
}
