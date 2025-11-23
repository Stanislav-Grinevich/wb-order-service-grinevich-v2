// Package kafkaconsumer реализует чтение сообщений из Kafka и их обработку.
package kafkaconsumer

import (
	"bytes"
	"context"
	"encoding/json"
	"log"

	"github.com/Stanislav-Grinevich/wb-order-service-grinevich/internal/cache"
	"github.com/Stanislav-Grinevich/wb-order-service-grinevich/internal/models"
	"github.com/Stanislav-Grinevich/wb-order-service-grinevich/internal/repo"

	"github.com/go-playground/validator/v10"
	kafka "github.com/segmentio/kafka-go"
)

// глобальный валидатор, чтобы не создавать его на каждое сообщение
var validateStruct = validator.New()

// Consumer обрабатывает сообщения.
type Consumer struct {
	reader *kafka.Reader
	repo   repo.OrdersStorage
	cache  cache.OrderCache
}

// Config задаёт параметры подключения к Kafka.
type Config struct {
	Brokers []string
	Topic   string
	GroupID string
}

// New создаёт консьюмера с ручным коммитом оффсетов.
func New(cfg Config, r repo.OrdersStorage, c cache.OrderCache) *Consumer {
	rd := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        cfg.Brokers,
		Topic:          cfg.Topic,
		GroupID:        cfg.GroupID,
		StartOffset:    kafka.LastOffset,
		CommitInterval: 0,
	})
	return &Consumer{reader: rd, repo: r, cache: c}
}

func (c *Consumer) Close() error { return c.reader.Close() }

// validate проверяет обязательные поля заказа с помощью тегов в models
// и пакета validator.v10.
func validate(o *models.Order) error {
	return validateStruct.Struct(o)
}

// processPayload парсит, валидирует и сохраняет заказ.
// Вынесено отдельно, чтобы можно было нормально тестить без Kafka.
func (c *Consumer) processPayload(ctx context.Context, payload []byte, offset int64) error {
	// убирается BOM, если есть
	payload = bytes.TrimPrefix(payload, []byte{0xEF, 0xBB, 0xBF})

	var o models.Order
	if err := json.Unmarshal(payload, &o); err != nil {
		log.Printf("[kafka] bad json (offset %d): %v", offset, err)
		return err
	}

	if err := validate(&o); err != nil {
		log.Printf("[kafka] invalid order (offset %d): %v", offset, err)
		return err
	}

	// запись в БД
	if err := c.repo.InsertOrUpdateOrder(ctx, o); err != nil {
		log.Printf("[kafka] db error (offset %d): %v", offset, err)
		return err
	}

	// обновление кэша
	c.cache.Set(o)

	log.Printf("[kafka] stored order %s (offset %d)", o.OrderUID, offset)
	return nil
}

// Run запускает бесконечный цикл чтения и обработки сообщений Kafka.
func (c *Consumer) Run(ctx context.Context) {
	log.Println("[kafka] consumer started")
	for {
		m, err := c.reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				log.Println("[kafka] stopped:", ctx.Err())
				return
			}
			log.Println("[kafka] read error:", err)
			continue
		}

		if err := c.processPayload(ctx, m.Value, m.Offset); err != nil {
			continue
		}

		// ручной коммит оффсета
		if err := c.reader.CommitMessages(ctx, m); err != nil {
			log.Printf("[kafka] commit error (offset %d): %v", m.Offset, err)
		}
	}
}
