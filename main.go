// Package main — точка входа в сервис wb-order-service.
// Здесь происходит инициализация базы данных, кэша, сервера и консьюмера.
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Stanislav-Grinevich/wb-order-service-grinevich/internal/cache"
	"github.com/Stanislav-Grinevich/wb-order-service-grinevich/internal/db"
	"github.com/Stanislav-Grinevich/wb-order-service-grinevich/internal/httpserver"
	"github.com/Stanislav-Grinevich/wb-order-service-grinevich/internal/kafkaconsumer"
	"github.com/Stanislav-Grinevich/wb-order-service-grinevich/internal/repo"
)

func main() {
	// контекст для graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// подключение к psql
	pool, err := db.NewPostgresPool(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	// репозиторий и кэш
	rp := repo.NewOrdersRepo(pool)
	cc := cache.New()

	// прогрев кэша
	orders, err := rp.LoadAllOrders(ctx, 200)
	if err != nil {
		log.Fatal(err)
	}

	// если БД пустая, то вставляем тестовый заказ
	if len(orders) == 0 {
		if err := rp.InsertTestOrder(ctx); err != nil {
			log.Fatal(err)
		}
		orders, _ = rp.LoadAllOrders(ctx, 1)
	}

	cc.Load(orders)
	log.Printf("cache warmup: %d orders", len(orders))

	// HTTP-сервер
	srv := httpserver.New(cc, rp)

	server := &http.Server{
		Addr:              ":8081",
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		fmt.Println("HTTP server listening on :8081")
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			// сравнение ошибок через errors.Is
			log.Fatal(err)
		}
	}()

	// консьюмер Kafka
	kcfg := kafkaconsumer.Config{
		Brokers: splitCSV(os.Getenv("KAFKA_BROKERS")),
		Topic:   os.Getenv("KAFKA_TOPIC"),
		GroupID: os.Getenv("KAFKA_GROUP"),
	}

	consumer := kafkaconsumer.New(kcfg, rp, cc)
	defer consumer.Close()

	go consumer.Run(ctx)

	// ожидание окончания работы (Ctrl+C)
	<-ctx.Done()
	log.Println("shutting down...")

	// аккуратная остановка HTTP сервера
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_ = server.Shutdown(shutdownCtx)
}

// splitCSV превращает строку с брокерами в массив
func splitCSV(s string) []string {
	if s == "" {
		return nil
	}

	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))

	for _, p := range parts {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}

	return out
}
