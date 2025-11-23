// Package repo — интерфейсы для работы с БД.
package repo

import (
	"context"

	"github.com/Stanislav-Grinevich/wb-order-service-grinevich/internal/models"
)

// OrdersStorage описывает, что нам нужно от хранилища заказов.
type OrdersStorage interface {
	InsertOrUpdateOrder(ctx context.Context, o models.Order) error
	LoadAllOrders(ctx context.Context, limit int) ([]models.Order, error)
	GetOrder(ctx context.Context, id string) (models.Order, error)
	InsertTestOrder(ctx context.Context) error
}
