// Package cache реализует потокобезопасный in-memory кэш.
// Используется для более быстрого доступа к данным: сначала проверяется кэш,
// а при промахе выполняется запрос в базу.
package cache

import (
	"sync"

	"github.com/Stanislav-Grinevich/wb-order-service-grinevich/internal/models"
)

// Cache хранит заказы в памяти.
// Добавлен простой лимит на количество записей, чтобы кэш не разрастался бесконечно.
type Cache struct {
	mu      sync.RWMutex
	data    map[string]models.Order
	order   []string // порядок добавления, нужен для инвалидации
	maxSize int
}

// defaultMaxSize — максимально количество заказов в кэше.
// Если заказов больше, самые старые будут выкидываться.
const defaultMaxSize = 1000

// New создаёт новый кэш с лимитом по умолчанию.
func New() *Cache {
	return &Cache{
		data:    make(map[string]models.Order),
		order:   make([]string, 0, defaultMaxSize),
		maxSize: defaultMaxSize,
	}
}

// NewWithLimit создаёт кэш с заданным лимитом.
// Оставил на будущее — удобно для тестов или кастомной настройки.
func NewWithLimit(max int) *Cache {
	if max <= 0 {
		max = defaultMaxSize
	}
	return &Cache{
		data:    make(map[string]models.Order),
		order:   make([]string, 0, max),
		maxSize: max,
	}
}

// Get возвращает заказ по ID из кэша.
func (c *Cache) Get(id string) (models.Order, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	o, ok := c.data[id]
	return o, ok
}

// Set добавляет или обновляет заказ в кэше по его OrderUID.
// Если записей становится больше лимита, то выкидывается самый старый.
func (c *Cache) Set(o models.Order) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.data[o.OrderUID]; !exists {
		c.order = append(c.order, o.OrderUID)
	}
	c.data[o.OrderUID] = o

	if c.maxSize > 0 && len(c.order) > c.maxSize {
		// выкидываем самый старый
		oldestID := c.order[0]
		c.order = c.order[1:]
		delete(c.data, oldestID)
	}
}

// Load загружает список заказов в кэш.
// Используется при прогреве кэша при старте сервиса.
func (c *Cache) Load(os []models.Order) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data = make(map[string]models.Order, len(os))
	c.order = c.order[:0]

	for _, o := range os {
		c.data[o.OrderUID] = o
		c.order = append(c.order, o.OrderUID)
	}

	// Если из БД пришло больше, чем maxSize, то оставляем только последние maxSize.
	if c.maxSize > 0 && len(c.order) > c.maxSize {
		start := len(c.order) - c.maxSize
		for _, id := range c.order[:start] {
			delete(c.data, id)
		}
		c.order = c.order[start:]
	}
}

// Size возвращает количество записей в кэше.
// Полезно для отладки и/или тестов.
func (c *Cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.data)
}
