# WB Order Service

## Предпосылки.
- Docker + Docker Compose.
- Go 1.24+.

## Запуск инфраструктуры.
docker compose up --build
(миграции применяются автоматически при старте контейнера)

## Миграции.
Ручной запуск при необходимости:
docker exec -it wb-order-service ./migrate up

## Переменные окружения.
См. .env.example.
При необходимости:
cp .env.example .env

## Запуск сервиса локально.
go run .
UI и API поднимутся на http://localhost:8081

## Отправка тестовых сообщений.
Producer лежит в cmd/producer и нормально запускается локально.
Примеры:
go run ./cmd/producer
go run ./cmd/producer -gen -n 200 -badRate 0.1 -delay 150ms

## API.
Если сервис в Docker Compose:
GET http://localhost:8082/order/<order_uid>
UI: http://localhost:8082/

Если сервис запущен локально (go run .):
GET http://localhost:8081/order/<order_uid>
UI: http://localhost:8081/

Пример тестового order_uid:
b563feb7b2b84b6test

## Кэш.
In-memory кэш с ограничением размера.
При старте выполняется загрузка данных из PostgreSQL.
При отсутствии записи в кэше производится выборка из БД.

## UI.
Статическая страница находится в каталоге web/ и раздаётся HTTP-сервером.

## Заметки.
Консьюмер обрезает BOM у входящих JSON.
Offset коммитится только после успешной записи в БД и обновления кэша.

## Troubleshooting.
При попытке подключения к [::1]:9092 — установить KAFKA_BROKERS=127.0.0.1:9092.
При использовании новой consumer group — требуется отправить сообщения повторно или изменить KAFKA_GROUP.
