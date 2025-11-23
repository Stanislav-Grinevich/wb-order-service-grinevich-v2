// Package repo — доступ к БД (Postgres) для сущности Order.
package repo

import (
	"context"
	"errors"
	"time"

	"github.com/Stanislav-Grinevich/wb-order-service-grinevich/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// OrdersRepo хранит пул подключений к БД.
type OrdersRepo struct {
	pool *pgxpool.Pool
}

// NewOrdersRepo создаёт репозиторий заказов.
func NewOrdersRepo(pool *pgxpool.Pool) *OrdersRepo {
	return &OrdersRepo{pool: pool}
}

// InsertOrUpdateOrder сохраняет заказ одной транзакцией.
func (r *OrdersRepo) InsertOrUpdateOrder(ctx context.Context, o models.Order) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// orders
	_, err = tx.Exec(ctx, `
		INSERT INTO orders
		  (order_uid, track_number, entry, locale, internal_signature, customer_id,
		   delivery_service, shardkey, sm_id, date_created, oof_shard)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		ON CONFLICT (order_uid) DO UPDATE SET
		  track_number=EXCLUDED.track_number,
		  entry=EXCLUDED.entry,
		  locale=EXCLUDED.locale,
		  internal_signature=EXCLUDED.internal_signature,
		  customer_id=EXCLUDED.customer_id,
		  delivery_service=EXCLUDED.delivery_service,
		  shardkey=EXCLUDED.shardkey,
		  sm_id=EXCLUDED.sm_id,
		  date_created=EXCLUDED.date_created,
		  oof_shard=EXCLUDED.oof_shard
	`, o.OrderUID, o.TrackNumber, o.Entry, o.Locale, o.InternalSignature,
		o.CustomerID, o.DeliveryService, o.ShardKey, o.SmID, o.DateCreated, o.OofShard)
	if err != nil {
		return err
	}

	// deliveries
	_, err = tx.Exec(ctx, `
		INSERT INTO deliveries
		  (order_uid, name, phone, zip, city, address, region, email)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT (order_uid) DO UPDATE SET
		  name=EXCLUDED.name,
		  phone=EXCLUDED.phone,
		  zip=EXCLUDED.zip,
		  city=EXCLUDED.city,
		  address=EXCLUDED.address,
		  region=EXCLUDED.region,
		  email=EXCLUDED.email
	`, o.OrderUID, o.Delivery.Name, o.Delivery.Phone, o.Delivery.Zip, o.Delivery.City,
		o.Delivery.Address, o.Delivery.Region, o.Delivery.Email)
	if err != nil {
		return err
	}

	// payments
	_, err = tx.Exec(ctx, `
		INSERT INTO payments
		  (order_uid, transaction, request_id, currency, provider, amount,
		   payment_dt, bank, delivery_cost, goods_total, custom_fee)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		ON CONFLICT (order_uid) DO UPDATE SET
		  transaction=EXCLUDED.transaction,
		  request_id=EXCLUDED.request_id,
		  currency=EXCLUDED.currency,
		  provider=EXCLUDED.provider,
		  amount=EXCLUDED.amount,
		  payment_dt=EXCLUDED.payment_dt,
		  bank=EXCLUDED.bank,
		  delivery_cost=EXCLUDED.delivery_cost,
		  goods_total=EXCLUDED.goods_total,
		  custom_fee=EXCLUDED.custom_fee
	`, o.OrderUID, o.Payment.Transaction, o.Payment.RequestID, o.Payment.Currency, o.Payment.Provider,
		o.Payment.Amount, o.Payment.PaymentDt, o.Payment.Bank, o.Payment.DeliveryCost, o.Payment.GoodsTotal, o.Payment.CustomFee)
	if err != nil {
		return err
	}

	// items
	_, err = tx.Exec(ctx, `DELETE FROM items WHERE order_uid = $1`, o.OrderUID)
	if err != nil {
		return err
	}
	for _, it := range o.Items {
		_, err = tx.Exec(ctx, `
			INSERT INTO items
			  (order_uid, chrt_id, track_number, price, rid, name, sale, size, total_price, nm_id, brand, status)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		`, o.OrderUID, it.ChrtID, it.TrackNumber, it.Price, it.Rid, it.Name, it.Sale, it.Size, it.TotalPrice, it.NmID, it.Brand, it.Status)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

// GetOrder возвращает заказ по order_uid из всех таблиц.
func (r *OrdersRepo) GetOrder(ctx context.Context, id string) (models.Order, error) {
	var o models.Order

	// orders
	err := r.pool.QueryRow(ctx, `
		SELECT order_uid, track_number, entry, locale, internal_signature, customer_id,
		       delivery_service, shardkey, sm_id, date_created, oof_shard
		FROM orders WHERE order_uid = $1
	`, id).
		Scan(&o.OrderUID, &o.TrackNumber, &o.Entry, &o.Locale, &o.InternalSignature, &o.CustomerID,
			&o.DeliveryService, &o.ShardKey, &o.SmID, &o.DateCreated, &o.OofShard)
	if err != nil {
		return models.Order{}, err
	}

	// deliveries
	err = r.pool.QueryRow(ctx, `
		SELECT name, phone, zip, city, address, region, email
		FROM deliveries WHERE order_uid = $1
	`, id).Scan(&o.Delivery.Name, &o.Delivery.Phone, &o.Delivery.Zip, &o.Delivery.City,
		&o.Delivery.Address, &o.Delivery.Region, &o.Delivery.Email)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return models.Order{}, err
	}

	// payments
	err = r.pool.QueryRow(ctx, `
		SELECT transaction, request_id, currency, provider, amount, payment_dt, bank, delivery_cost, goods_total, custom_fee
		FROM payments WHERE order_uid = $1
	`, id).Scan(&o.Payment.Transaction, &o.Payment.RequestID, &o.Payment.Currency, &o.Payment.Provider,
		&o.Payment.Amount, &o.Payment.PaymentDt, &o.Payment.Bank, &o.Payment.DeliveryCost, &o.Payment.GoodsTotal, &o.Payment.CustomFee)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return models.Order{}, err
	}

	// items
	rows, err := r.pool.Query(ctx, `
		SELECT chrt_id, track_number, price, rid, name, sale, size, total_price, nm_id, brand, status
		FROM items WHERE order_uid = $1
	`, id)
	if err != nil {
		return models.Order{}, err
	}
	defer rows.Close()

	o.Items = make([]models.Item, 0, 8)
	for rows.Next() {
		var it models.Item
		if err := rows.Scan(&it.ChrtID, &it.TrackNumber, &it.Price, &it.Rid, &it.Name, &it.Sale,
			&it.Size, &it.TotalPrice, &it.NmID, &it.Brand, &it.Status); err != nil {
			return models.Order{}, err
		}
		o.Items = append(o.Items, it)
	}
	return o, nil
}

// LoadAllOrders возвращает последние N заказов по order_uid (для прогрева кэша).
func (r *OrdersRepo) LoadAllOrders(ctx context.Context, limit int) ([]models.Order, error) {
	if limit <= 0 {
		limit = 1000
	}
	rows, err := r.pool.Query(ctx, `
		SELECT order_uid
		FROM orders
		ORDER BY date_created DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}

	out := make([]models.Order, 0, len(ids))
	for _, id := range ids {
		o, err := r.GetOrder(ctx, id)
		if err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, nil
}

// InsertTestOrder вставляет демонстрационный заказ.
func (r *OrdersRepo) InsertTestOrder(ctx context.Context) error {
	return r.InsertOrUpdateOrder(ctx, models.Order{
		OrderUID:          "b563feb7b2b84b6test",
		TrackNumber:       "WBILMTESTTRACK",
		Entry:             "WBIL",
		Locale:            "en",
		InternalSignature: "",
		CustomerID:        "test",
		DeliveryService:   "meest",
		ShardKey:          "9",
		SmID:              99,
		DateCreated:       time.Date(2021, 11, 26, 6, 22, 19, 0, time.UTC),
		OofShard:          "1",
		Delivery: models.Delivery{
			Name:    "Test Testov",
			Phone:   "+9720000000",
			Zip:     "2639809",
			City:    "Kiryat Mozkin",
			Address: "Ploshad Mira 15",
			Region:  "Kraiot",
			Email:   "test@gmail.com",
		},
		Payment: models.Payment{
			Transaction:  "b563feb7b2b84b6test",
			RequestID:    "",
			Currency:     "USD",
			Provider:     "wbpay",
			Amount:       1817,
			PaymentDt:    1637907727,
			Bank:         "alpha",
			DeliveryCost: 1500,
			GoodsTotal:   317,
			CustomFee:    0,
		},
		Items: []models.Item{{
			ChrtID:      9934930,
			TrackNumber: "WBILMTESTTRACK",
			Price:       453,
			Rid:         "ab4219087a764ae0btest",
			Name:        "Mascaras",
			Sale:        30,
			Size:        "0",
			TotalPrice:  317,
			NmID:        2389212,
			Brand:       "Vivienne Sabo",
			Status:      202,
		}},
	})
}
