// Package main реализует простой продюсер, который читает JSON и публикует его в топик.
// По умолчанию отправляет все model*.json и broken*.json из текущей директории.
// Можно включить режим генерации случайных заказов флагом -gen.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/joho/godotenv"
	kafka "github.com/segmentio/kafka-go"

	"github.com/Stanislav-Grinevich/wb-order-service-grinevich/internal/models"
)

func main() {
	if err := godotenv.Overload("cmd/producer/.env.local"); err != nil {
		log.Printf("env load error: %v", err)
	}

	brokersEnv := os.Getenv("KAFKA_BROKERS")
	topic := os.Getenv("KAFKA_TOPIC")

	if brokersEnv == "" || topic == "" {
		log.Fatal("нужно выставить KAFKA_BROKERS и KAFKA_TOPIC")
	}

	brokers := splitCSV(brokersEnv)
	if len(brokers) == 0 {
		log.Fatal("KAFKA_BROKERS пустой после парсинга")
	}

	// флаги для генератора
	genMode := flag.Bool("gen", false, "генерировать случайные заказы вместо чтения json файлов")
	genN := flag.Int("n", 100, "сколько сообщений сгенерировать в режиме -gen")
	badRate := flag.Float64("badRate", 0.1, "доля невалидных сообщений (0..1) в режиме -gen")
	delay := flag.Duration("delay", 200*time.Millisecond, "задержка между сообщениями в режиме -gen")
	filesEnv := flag.String("files", "", "доп. список файлов через запятую, если хочется отправить конкретные файлы")

	flag.Parse()

	// writer для записи в Kafka.
	writer := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireAll,
		Async:        false,
	}
	defer writer.Close()

	ctx := context.Background()

	if *genMode {
		if *genN <= 0 {
			log.Fatal("в режиме -gen нужно чтобы -n был > 0")
		}
		if *badRate < 0 || *badRate > 1 {
			log.Fatal("badRate должен быть в диапазоне 0..1")
		}

		log.Printf("режим генерации: n=%d badRate=%.2f delay=%s", *genN, *badRate, delay.String())
		sendGenerated(ctx, writer, *genN, *badRate, *delay)
		return
	}

	// ищем тестовые json в текущей папке
	validFiles, err := filepath.Glob("model*.json")
	if err != nil {
		log.Fatalf("ошибка поиска model*.json: %v", err)
	}

	brokenFiles, err := filepath.Glob("broken*.json")
	if err != nil {
		log.Fatalf("ошибка поиска broken*.json: %v", err)
	}

	// можно дополнительно докинуть конкретные файлы через -files
	extraFiles := splitCSV(*filesEnv)
	if len(extraFiles) != 0 {
		validFiles = append(validFiles, extraFiles...)
	}

	if len(validFiles) == 0 && len(brokenFiles) == 0 {
		log.Fatal("не найдено ни одного json файла (model*.json или broken*.json)")
	}

	fmt.Printf("найдено валидных файлов: %d, битых файлов: %d\n", len(validFiles), len(brokenFiles))

	// сначала отправляются валидные
	for _, name := range validFiles {
		if err := sendFile(ctx, writer, name); err != nil {
			log.Fatalf("ошибка отправки %s: %v", name, err)
		}
	}

	// потом отправляются битые
	for _, name := range brokenFiles {
		if err := sendFile(ctx, writer, name); err != nil {
			log.Fatalf("ошибка отправки %s: %v", name, err)
		}
	}
}

// sendFile читает файл и отправляет его содержимое как одно сообщение в Kafka.
func sendFile(ctx context.Context, w *kafka.Writer, fileName string) error {
	data, err := os.ReadFile(fileName)
	if err != nil {
		return err
	}

	msg := kafka.Message{
		Key:   []byte(filepath.Base(fileName)),
		Value: data,
		Time:  time.Now(),
	}

	log.Printf("-> отправка %s", fileName)
	return w.WriteMessages(ctx, msg)
}

// sendGenerated генерирует поток заказов и отправляет их в Kafka.
func sendGenerated(ctx context.Context, w *kafka.Writer, n int, badRate float64, delay time.Duration) {
	rand.Seed(time.Now().UnixNano())

	for i := 0; i < n; i++ {
		var (
			key   string
			value []byte
		)

		// иногда шлём битое сообщение
		if rand.Float64() < badRate {
			key, value = makeBrokenMessage()
		} else {
			order := makeRandomOrder()
			key = order.OrderUID

			b, err := json.Marshal(order)
			if err != nil {
				log.Printf("не удалось сериализовать заказ: %v", err)
				continue
			}
			value = b
		}

		msg := kafka.Message{
			Key:   []byte(key),
			Value: value,
			Time:  time.Now(),
		}

		if err := w.WriteMessages(ctx, msg); err != nil {
			log.Printf("ошибка отправки сообщения %d: %v", i+1, err)
			continue
		}
		log.Printf("-> отправлено сообщение %d/%d (key=%s)", i+1, n, key)

		if delay > 0 {
			time.Sleep(delay)
		}
	}
}

// makeRandomOrder генерирует валидный заказ по модели.
func makeRandomOrder() models.Order {
	now := time.Now().UTC()

	uid := fmt.Sprintf("gen-%d-%d", now.UnixNano(), rand.Intn(1000))
	track := fmt.Sprintf("TRACK-%d", rand.Intn(1_000_000))

	itemsCount := rand.Intn(3) + 1
	items := make([]models.Item, 0, itemsCount)
	for i := 0; i < itemsCount; i++ {
		price := rand.Intn(2000) + 100
		sale := rand.Intn(50)
		total := price * (100 - sale) / 100

		items = append(items, models.Item{
			ChrtID:      int64(rand.Intn(10_000_000)),
			TrackNumber: track,
			Price:       price,
			Rid:         fmt.Sprintf("rid-%d", rand.Intn(10_000_000)),
			Name:        fmt.Sprintf("Item #%d", i+1),
			Sale:        sale,
			Size:        "0",
			TotalPrice:  total,
			NmID:        int64(rand.Intn(10_000_000)),
			Brand:       "DemoBrand",
			Status:      200 + rand.Intn(10),
		})
	}

	return models.Order{
		OrderUID:    uid,
		TrackNumber: track,
		Entry:       "WBIL",
		Delivery: models.Delivery{
			Name:    "Gen User",
			Phone:   "+79000000000",
			Zip:     fmt.Sprintf("%06d", rand.Intn(999999)),
			City:    "Moscow",
			Address: "Demo street 1",
			Region:  "Demo region",
			Email:   "gen@example.com",
		},
		Payment: models.Payment{
			Transaction:  uid,
			RequestID:    "",
			Currency:     "USD",
			Provider:     "wbpay",
			Amount:       1000 + rand.Intn(5000),
			PaymentDt:    now.Unix(),
			Bank:         "alpha",
			DeliveryCost: 1500,
			GoodsTotal:   317,
			CustomFee:    0,
		},
		Items:             items,
		Locale:            "en",
		InternalSignature: "",
		CustomerID:        "gen",
		DeliveryService:   "meest",
		ShardKey:          "9",
		SmID:              99,
		DateCreated:       now,
		OofShard:          "1",
	}
}

// makeBrokenMessage делает заведомо невалидное сообщение.
func makeBrokenMessage() (string, []byte) {
	uid := fmt.Sprintf("broken-%d", time.Now().UnixNano())
	bad := map[string]any{
		"track_number": "BADTRACK",
		"customer_id":  "bad",
		"items":        "not-an-array",
	}

	b, _ := json.Marshal(bad)
	return uid, b
}

// splitCSV разбивает строку через запятую.
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
