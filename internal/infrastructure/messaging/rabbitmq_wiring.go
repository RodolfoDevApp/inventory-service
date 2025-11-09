package messaging

import (
	"context"
	"log"

	"github.com/RodolfoDevApp/eventshop-inventory-go/internal/application"
	messaging "github.com/rodolfodevapp/eventshop-messaging-go/rabbitmq"
)

type EventBusPair struct {
	OrdersConsumer *messaging.RabbitMqEventBus
	Producer       *messaging.RabbitMqEventBus
}

// Consumer para orders.events + Producer para inventory.events
func NewEventBusPair(
	rabbitUri string,
	ordersQueuePrefix string,
) EventBusPair {
	ordersOpts := messaging.RabbitMqOptions{
		URI:          rabbitUri,
		ExchangeName: "orders.events",
		QueuePrefix:  ordersQueuePrefix,
		Prefetch:     32,
		RetryDelayMs: 30000,
	}
	producerOpts := messaging.RabbitMqOptions{
		URI:          rabbitUri,
		ExchangeName: "inventory.events",
		QueuePrefix:  "inventory.dispatcher.v1",
		Prefetch:     32,
		RetryDelayMs: 30000,
	}

	ordersBus := messaging.NewRabbitMqEventBus(ordersOpts, nil, nil)
	producerBus := messaging.NewRabbitMqEventBus(producerOpts, nil, nil)

	return EventBusPair{
		OrdersConsumer: ordersBus,
		Producer:       producerBus,
	}
}

// Consumer para catalog.events
func NewCatalogEventBus(
	rabbitUri string,
	queuePrefix string,
) *messaging.RabbitMqEventBus {
	opts := messaging.RabbitMqOptions{
		URI:          rabbitUri,
		ExchangeName: "catalog.events",
		QueuePrefix:  queuePrefix,
		Prefetch:     32,
		RetryDelayMs: 30000,
	}
	return messaging.NewRabbitMqEventBus(opts, nil, nil)
}

func RegisterOrderSubscriptions(
	ctx context.Context,
	bus *messaging.RabbitMqEventBus,
	orderPlacedHandler application.EventHandler,
	orderCancelledHandler application.EventHandler,
) error {
	bus.Subscribe("OrderPlacedEvent", orderPlacedHandler)
	bus.Subscribe("OrderCancelledEvent", orderCancelledHandler)
	bus.Subscribe("OrderRejectedEvent", orderCancelledHandler)

	if err := bus.StartConsumers(ctx); err != nil {
		log.Printf("Error starting orders consumers: %v", err)
		return err
	}
	return nil
}

func RegisterCatalogSubscriptions(
	ctx context.Context,
	bus *messaging.RabbitMqEventBus,
	productCreatedHandler application.EventHandler,
) error {
	bus.Subscribe("ProductCreated", productCreatedHandler)

	if err := bus.StartConsumers(ctx); err != nil {
		log.Printf("Error starting catalog consumers: %v", err)
		return err
	}
	return nil
}
