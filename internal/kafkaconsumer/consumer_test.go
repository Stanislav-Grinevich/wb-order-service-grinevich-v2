package kafkaconsumer

import (
	"context"
	"testing"
	"time"

	"github.com/Stanislav-Grinevich/wb-order-service-grinevich/internal/models"
)

// МОКИ ПОД ИНТЕРФЕЙСЫ

type fakeRepo struct {
	last  models.Order
	calls int
	fail  bool
}

func (f *fakeRepo) InsertOrUpdateOrder(ctx context.Context, o models.Order) error {
	if f.fail {
		return context.Canceled
	}
	f.last = o
	f.calls++
	return nil
}
func (f *fakeRepo) LoadAllOrders(ctx context.Context, limit int) ([]models.Order, error) {
	return nil, nil
}
func (f *fakeRepo) GetOrder(ctx context.Context, id string) (models.Order, error) {
	return models.Order{}, nil
}
func (f *fakeRepo) InsertTestOrder(ctx context.Context) error { return nil }

type fakeCache struct {
	last models.Order
	sets int
}

func (f *fakeCache) Get(id string) (models.Order, bool) { return models.Order{}, false }
func (f *fakeCache) Set(o models.Order) {
	f.last = o
	f.sets++
}
func (f *fakeCache) Load(os []models.Order) {}
func (f *fakeCache) Size() int              { return f.sets }

func TestProcessPayloadValid(t *testing.T) {
	r := &fakeRepo{}
	c := &fakeCache{}

	cons := &Consumer{repo: r, cache: c}

	// все поля валидные под текущие validate теги
	payload := []byte(`{
		"order_uid":"order123",
		"track_number":"WBILMTESTTRACK",
		"entry":"WBIL",
		"delivery":{"name":"n","phone":"+79000000000","zip":"12345","city":"c","address":"a","region":"r","email":"e@e.com"},
		"payment":{"transaction":"order123","request_id":"","currency":"USD","provider":"wbpay","amount":1,"payment_dt":1,"bank":"alpha","delivery_cost":1,"goods_total":1,"custom_fee":0},
		"items":[{"chrt_id":1,"track_number":"WBILMTESTTRACK","price":1,"rid":"rid1","name":"i","sale":1,"size":"0","total_price":1,"nm_id":1,"brand":"b","status":1}],
		"locale":"en",
		"internal_signature":"",
		"customer_id":"c",
		"delivery_service":"d",
		"shardkey":"s",
		"sm_id":1,
		"date_created":"2021-11-26T06:22:19Z",
		"oof_shard":"o"
	}`)

	if err := cons.processPayload(context.Background(), payload, 1); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if r.calls != 1 {
		t.Fatalf("expected 1 repo call got %d", r.calls)
	}
	if r.last.OrderUID != "order123" {
		t.Fatalf("repo stored wrong order")
	}
	if c.sets != 1 {
		t.Fatalf("cache should be updated")
	}
}

func TestProcessPayloadInvalidJSON(t *testing.T) {
	r := &fakeRepo{}
	c := &fakeCache{}

	cons := &Consumer{repo: r, cache: c}

	payload := []byte(`{bad json`)

	if err := cons.processPayload(context.Background(), payload, 1); err == nil {
		t.Fatalf("expected error")
	}

	if r.calls != 0 {
		t.Fatalf("repo should not be called on bad json")
	}
	if c.sets != 0 {
		t.Fatalf("cache should not be called on bad json")
	}
}

// Валидный JSON, но модель не проходит валидацию.
// Все поля валидные, кроме order_uid.
func TestProcessPayloadInvalidModel(t *testing.T) {
	r := &fakeRepo{}
	c := &fakeCache{}

	cons := &Consumer{repo: r, cache: c}

	payload := []byte(`{
		"order_uid":"",
		"track_number":"WBILMTESTTRACK",
		"entry":"WBIL",
		"delivery":{"name":"n","phone":"+79000000000","zip":"12345","city":"c","address":"a","region":"r","email":"e@e.com"},
		"payment":{"transaction":"order124","request_id":"","currency":"USD","provider":"wbpay","amount":1,"payment_dt":1,"bank":"alpha","delivery_cost":1,"goods_total":1,"custom_fee":0},
		"items":[{"chrt_id":1,"track_number":"WBILMTESTTRACK","price":1,"rid":"rid1","name":"i","sale":1,"size":"0","total_price":1,"nm_id":1,"brand":"b","status":1}],
		"locale":"en",
		"internal_signature":"",
		"customer_id":"c",
		"delivery_service":"d",
		"shardkey":"s",
		"sm_id":1,
		"date_created":"2021-11-26T06:22:19Z",
		"oof_shard":"o"
	}`)

	if err := cons.processPayload(context.Background(), payload, 2); err == nil {
		t.Fatalf("expected validation error")
	}

	if r.calls != 0 {
		t.Fatalf("repo should not be called on invalid model")
	}
	if c.sets != 0 {
		t.Fatalf("cache should not be updated on invalid model")
	}
}

func TestProcessPayloadRepoFail(t *testing.T) {
	r := &fakeRepo{fail: true}
	c := &fakeCache{}

	cons := &Consumer{repo: r, cache: c}

	payload := []byte(`{
		"order_uid":"order125",
		"track_number":"WBILMTESTTRACK",
		"entry":"WBIL",
		"delivery":{"name":"n","phone":"+79000000000","zip":"12345","city":"c","address":"a","region":"r","email":"e@e.com"},
		"payment":{"transaction":"order125","request_id":"","currency":"USD","provider":"wbpay","amount":1,"payment_dt":1,"bank":"alpha","delivery_cost":1,"goods_total":1,"custom_fee":0},
		"items":[{"chrt_id":1,"track_number":"WBILMTESTTRACK","price":1,"rid":"rid1","name":"i","sale":1,"size":"0","total_price":1,"nm_id":1,"brand":"b","status":1}],
		"locale":"en",
		"internal_signature":"",
		"customer_id":"c",
		"delivery_service":"d",
		"shardkey":"s",
		"sm_id":1,
		"date_created":"2021-11-26T06:22:19Z",
		"oof_shard":"o"
	}`)

	if err := cons.processPayload(context.Background(), payload, 3); err == nil {
		t.Fatalf("expected error")
	}

	if r.calls != 0 {
		t.Fatalf("repo should not be called on repo fail")
	}
	if c.sets != 0 {
		t.Fatalf("cache should not be updated on repo fail")
	}
}

var _ = time.Now
