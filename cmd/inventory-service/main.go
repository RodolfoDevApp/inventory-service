package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/RodolfoDevApp/eventshop-inventory-go/internal/api"
	"github.com/RodolfoDevApp/eventshop-inventory-go/internal/application"
	"github.com/RodolfoDevApp/eventshop-inventory-go/internal/config"
	"github.com/RodolfoDevApp/eventshop-inventory-go/internal/infrastructure/db"
	"github.com/RodolfoDevApp/eventshop-inventory-go/internal/infrastructure/messaging"
	outboxinfra "github.com/RodolfoDevApp/eventshop-inventory-go/internal/infrastructure/outbox"
)

func main() {
	cfg := config.Load()
	log.Printf("Starting inventory service on port %s", cfg.HttpPort)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dbConn, err := sql.Open("pgx", cfg.PgDsn)
	if err != nil {
		log.Fatalf("failed to open postgres: %v", err)
	}
	defer dbConn.Close()

	if err := dbConn.PingContext(ctx); err != nil {
		log.Fatalf("failed to ping postgres: %v", err)
	}

	// Repos
	stockRepo := db.NewPgStockItemRepository(dbConn)
	reservationRepo := db.NewPgStockReservationRepository(dbConn)
	outboxRepo := db.NewPgOutboxRepository(dbConn)

	// Event buses
	buses := messaging.NewEventBusPair(cfg.RabbitUri, "inventory.orders-events.v1")
	catalogBus := messaging.NewCatalogEventBus(cfg.RabbitUri, "inventory.catalog-events.v1")

	// Outbox writer + dispatcher + scheduler
	outboxWriter := application.NewOutboxWriter(outboxRepo)
	dispatcher := outboxinfra.NewDispatcher(
		outboxRepo,
		buses.Producer,
		cfg.OutboxMaxRetry,
		cfg.OutboxBatchSize,
	)
	scheduler := outboxinfra.NewScheduler(dispatcher, cfg.OutboxIntervalSec)
	scheduler.Start(ctx)

	// Application services
	reserveSvc := application.NewReserveStockService(stockRepo, reservationRepo, outboxWriter)
	releaseSvc := application.NewReleaseReservationService(stockRepo, reservationRepo, outboxWriter)

	// Handlers de eventos de Orders
	orderPlacedHandler := application.NewOrderPlacedHandler(reserveSvc)
	orderCancelledHandler := application.NewOrderCancelledHandler(releaseSvc)

	// Handler de eventos de Catalog
	productCreatedHandler := application.NewProductCreatedHandler(stockRepo, outboxWriter)

	// Suscripciones
	if err := messaging.RegisterOrderSubscriptions(
		ctx,
		buses.OrdersConsumer,
		orderPlacedHandler,
		orderCancelledHandler,
	); err != nil {
		log.Fatalf("failed to start orders subscriptions: %v", err)
	}

	if err := messaging.RegisterCatalogSubscriptions(
		ctx,
		catalogBus,
		productCreatedHandler,
	); err != nil {
		log.Fatalf("failed to start catalog subscriptions: %v", err)
	}

	// HTTP API
	mux := http.NewServeMux()
	apiServer := api.NewServer(cfg, stockRepo, reservationRepo)
	apiServer.RegisterRoutes(mux)

	httpSrv := &http.Server{
		Addr:    ":" + cfg.HttpPort,
		Handler: mux,
	}

	go func() {
		log.Printf("HTTP listening on :%s", cfg.HttpPort)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http server error: %v", err)
		}
	}()

	// Esperar señal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh
	log.Printf("Shutting down inventory service, signal: %s", sig.String())

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := httpSrv.Shutdown(shutdownCtx); err != nil {
		log.Printf("http shutdown error: %v", err)
	}
}
