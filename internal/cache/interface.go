// Package cache — интерфейсы для работы с кэшем.
package cache

import "github.com/Stanislav-Grinevich/wb-order-service-grinevich/internal/models"

// OrderCache описывает, что нам нужно от кэша заказов.
type OrderCache interface {
	Get(id string) (models.Order, bool)
	Set(o models.Order)
	Load(os []models.Order)
	Size() int
}
