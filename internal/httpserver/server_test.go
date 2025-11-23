package httpserver

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Stanislav-Grinevich/wb-order-service-grinevich/internal/models"
)

//МОКИ

type fakeCache struct {
	m map[string]models.Order
}

func (f *fakeCache) Get(id string) (models.Order, bool) {
	o, ok := f.m[id]
	return o, ok
}
func (f *fakeCache) Set(o models.Order)     { f.m[o.OrderUID] = o }
func (f *fakeCache) Load(os []models.Order) {}
func (f *fakeCache) Size() int              { return len(f.m) }

type fakeRepo struct {
	data map[string]models.Order
}

func (f *fakeRepo) InsertOrUpdateOrder(ctx context.Context, o models.Order) error {
	f.data[o.OrderUID] = o
	return nil
}
func (f *fakeRepo) LoadAllOrders(ctx context.Context, limit int) ([]models.Order, error) {
	return nil, nil
}
func (f *fakeRepo) GetOrder(ctx context.Context, id string) (models.Order, error) {
	o, ok := f.data[id]
	if !ok {
		return models.Order{}, errors.New("not found")
	}
	return o, nil
}
func (f *fakeRepo) InsertTestOrder(ctx context.Context) error { return nil }

func TestGetOrderFromCache(t *testing.T) {
	o := minimalOrder("id1")

	c := &fakeCache{m: map[string]models.Order{
		"id1": o,
	}}
	r := &fakeRepo{data: map[string]models.Order{}}

	s := New(c, r)

	req := httptest.NewRequest(http.MethodGet, "/order/id1", nil)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", rr.Code)
	}
}

// ТЕСТЫ
func TestGetOrderFromRepoWhenCacheMiss(t *testing.T) {
	o := minimalOrder("id2")

	c := &fakeCache{m: map[string]models.Order{}}
	r := &fakeRepo{data: map[string]models.Order{
		"id2": o,
	}}

	s := New(c, r)

	req := httptest.NewRequest(http.MethodGet, "/order/id2", nil)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", rr.Code)
	}
}

func TestGetOrderNotFound(t *testing.T) {
	c := &fakeCache{m: map[string]models.Order{}}
	r := &fakeRepo{data: map[string]models.Order{}}

	s := New(c, r)

	req := httptest.NewRequest(http.MethodGet, "/order/missing", nil)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 got %d", rr.Code)
	}
}

func minimalOrder(id string) models.Order {
	now := time.Now()
	return models.Order{
		OrderUID:        id,
		TrackNumber:     "t",
		Entry:           "WBIL",
		Locale:          "en",
		CustomerID:      "c",
		DeliveryService: "d",
		ShardKey:        "s",
		SmID:            1,
		DateCreated:     now,
		OofShard:        "o",
		Delivery:        models.Delivery{Name: "n", Phone: "p", Zip: "z", City: "c", Address: "a", Region: "r", Email: "e"},
		Payment:         models.Payment{Transaction: id, Currency: "USD", Provider: "wbpay", Amount: 1, PaymentDt: 1, Bank: "alpha", DeliveryCost: 1, GoodsTotal: 1, CustomFee: 0},
		Items:           []models.Item{{ChrtID: 1, TrackNumber: "t", Price: 1, Rid: "r", Name: "i", Sale: 1, Size: "0", TotalPrice: 1, NmID: 1, Brand: "b", Status: 1}},
	}
}
