package config

import (
	"log"
	"os"
	"strconv"
)

type Config struct {
	HttpPort          string
	PgDsn             string
	RabbitUri         string
	OutboxBatchSize   int
	OutboxMaxRetry    int
	OutboxIntervalSec int
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func atoiEnv(key string, def int) int {
	v := getenv(key, "")
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		log.Printf("invalid int env %s=%s, using default %d", key, v, def)
		return def
	}
	return n
}

func Load() Config {
	return Config{
		HttpPort: getenv("HTTP_PORT", "8083"),
		// Nota: coincide con 03-create-inventory-db.sql
		PgDsn:             getenv("PG_DSN", "postgres://inventory:inventory@localhost:5432/inventory_db?sslmode=disable"),
		RabbitUri:         getenv("RABBITMQ_URI", "amqp://user:password@localhost:5672/"),
		OutboxBatchSize:   atoiEnv("OUTBOX_BATCH_SIZE", 100),
		OutboxMaxRetry:    atoiEnv("OUTBOX_MAX_RETRY", 5),
		OutboxIntervalSec: atoiEnv("OUTBOX_INTERVAL_SEC", 5),
	}
}
