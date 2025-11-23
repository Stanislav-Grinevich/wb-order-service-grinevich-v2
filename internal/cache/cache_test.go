package cache

import (
	"testing"
	"time"

	"github.com/Stanislav-Grinevich/wb-order-service-grinevich/internal/models"
)

func TestCacheSetGet(t *testing.T) {
	c := NewWithLimit(10)

	o := models.Order{
		OrderUID:        "id1",
		TrackNumber:     "t1",
		Entry:           "WBIL",
		Locale:          "en",
		CustomerID:      "test",
		DeliveryService: "meest",
		ShardKey:        "9",
		SmID:            99,
		DateCreated:     time.Now(),
		OofShard:        "1",
		Delivery:        models.Delivery{Name: "n", Phone: "p", Zip: "z", City: "c", Address: "a", Region: "r", Email: "e"},
		Payment:         models.Payment{Transaction: "id1", Currency: "USD", Provider: "wbpay", Amount: 1, PaymentDt: 1, Bank: "alpha", DeliveryCost: 1, GoodsTotal: 1, CustomFee: 0},
		Items:           []models.Item{{ChrtID: 1, TrackNumber: "t1", Price: 1, Rid: "r", Name: "i", Sale: 1, Size: "0", TotalPrice: 1, NmID: 1, Brand: "b", Status: 1}},
	}

	c.Set(o)

	got, ok := c.Get("id1")
	if !ok {
		t.Fatalf("expected ok=true")
	}
	if got.OrderUID != "id1" {
		t.Fatalf("expected id1, got %s", got.OrderUID)
	}
}

func TestCacheEviction(t *testing.T) {
	c := NewWithLimit(2)

	o1 := models.Order{OrderUID: "1", TrackNumber: "t", Entry: "e", Locale: "en", CustomerID: "c", DeliveryService: "d", ShardKey: "s", SmID: 1, DateCreated: time.Now(), OofShard: "o",
		Delivery: models.Delivery{Name: "n", Phone: "p", Zip: "z", City: "c", Address: "a", Region: "r", Email: "e"},
		Payment:  models.Payment{Transaction: "1", Currency: "USD", Provider: "wbpay", Amount: 1, PaymentDt: 1, Bank: "alpha", DeliveryCost: 1, GoodsTotal: 1, CustomFee: 0},
		Items:    []models.Item{{ChrtID: 1, TrackNumber: "t", Price: 1, Rid: "r", Name: "i", Sale: 1, Size: "0", TotalPrice: 1, NmID: 1, Brand: "b", Status: 1}},
	}
	o2 := o1
	o2.OrderUID = "2"
	o2.Payment.Transaction = "2"
	o3 := o1
	o3.OrderUID = "3"
	o3.Payment.Transaction = "3"

	c.Set(o1)
	c.Set(o2)
	c.Set(o3) // должно вытолкнуть o1

	if c.Size() != 2 {
		t.Fatalf("expected size=2, got %d", c.Size())
	}
	if _, ok := c.Get("1"); ok {
		t.Fatalf("expected order 1 to be evicted")
	}
	if _, ok := c.Get("2"); !ok {
		t.Fatalf("expected order 2 to exist")
	}
	if _, ok := c.Get("3"); !ok {
		t.Fatalf("expected order 3 to exist")
	}
}
