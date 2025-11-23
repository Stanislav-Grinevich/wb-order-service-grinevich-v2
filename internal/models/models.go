package models

import "time"

type Order struct {
	OrderUID          string    `json:"order_uid" validate:"required,min=8,max=64"`
	TrackNumber       string    `json:"track_number" validate:"required,min=5,max=32"`
	Entry             string    `json:"entry" validate:"required,min=2,max=8"`
	Locale            string    `json:"locale" validate:"required,min=2,max=8"`
	InternalSignature string    `json:"internal_signature"`
	CustomerID        string    `json:"customer_id" validate:"required,min=1,max=64"`
	DeliveryService   string    `json:"delivery_service" validate:"required,min=1,max=64"`
	ShardKey          string    `json:"shardkey" validate:"required,min=1,max=8"`
	SmID              int       `json:"sm_id" validate:"required,gte=0"`
	DateCreated       time.Time `json:"date_created" validate:"required"`
	OofShard          string    `json:"oof_shard" validate:"required,min=1,max=8"`

	Delivery Delivery `json:"delivery" validate:"required"`
	Payment  Payment  `json:"payment" validate:"required"`
	Items    []Item   `json:"items" validate:"required,min=1,dive"`
}

type Delivery struct {
	Name    string `json:"name" validate:"required,min=1,max=128"`
	Phone   string `json:"phone" validate:"required,e164"`
	Zip     string `json:"zip" validate:"required,min=3,max=16"`
	City    string `json:"city" validate:"required,min=1,max=64"`
	Address string `json:"address" validate:"required,min=1,max=256"`
	Region  string `json:"region" validate:"required,min=1,max=64"`
	Email   string `json:"email" validate:"required,email"`
}

type Payment struct {
	Transaction  string `json:"transaction" validate:"required,min=8,max=64"`
	RequestID    string `json:"request_id"`
	Currency     string `json:"currency" validate:"required,len=3"`
	Provider     string `json:"provider" validate:"required,min=1,max=32"`
	Amount       int    `json:"amount" validate:"required,gte=0"`
	PaymentDt    int64  `json:"payment_dt" validate:"required,gt=0"`
	Bank         string `json:"bank" validate:"required,min=1,max=32"`
	DeliveryCost int    `json:"delivery_cost" validate:"required,gte=0"`
	GoodsTotal   int    `json:"goods_total" validate:"required,gte=0"`
	CustomFee    int    `json:"custom_fee" validate:"gte=0"`
}

type Item struct {
	ChrtID      int64  `json:"chrt_id" validate:"required,gt=0"`
	TrackNumber string `json:"track_number" validate:"required,min=5,max=32"`
	Price       int    `json:"price" validate:"required,gte=0"`
	Rid         string `json:"rid" validate:"required,min=4,max=64"`
	Name        string `json:"name" validate:"required,min=1,max=128"`
	Sale        int    `json:"sale" validate:"gte=0,lte=100"`
	Size        string `json:"size" validate:"required,max=16"`
	TotalPrice  int    `json:"total_price" validate:"required,gte=0"`
	NmID        int64  `json:"nm_id" validate:"required,gt=0"`
	Brand       string `json:"brand" validate:"required,min=1,max=64"`
	Status      int    `json:"status" validate:"required,gte=0"`
}
