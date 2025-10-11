package cache

import (
	"sync"
	"wb-orders-service/internal/models"
)

type OrderCache struct {
	mu     sync.RWMutex
	orders map[string]models.Order
}

func NewOrderCache() *OrderCache {
	return &OrderCache{
		orders: make(map[string]models.Order),
	}
}

func (c *OrderCache) Set(orderUID string, order models.Order) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.orders[orderUID] = order
}

func (c *OrderCache) Get(orderUID string) (models.Order, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	order, exists := c.orders[orderUID]
	return order, exists
}

func (c *OrderCache) LoadFromDB(orders []models.Order) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, order := range orders {
		c.orders[order.OrderUID] = order
	}
}

func (c *OrderCache) GetAll() []models.Order {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	orders := make([]models.Order, 0, len(c.orders))
	for _, order := range c.orders {
		orders = append(orders, order)
	}
	return orders
}