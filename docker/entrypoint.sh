#!/usr/bin/env bash
set -e

echo "waiting postgres..."
until pg_isready -h postgres -U wb_user -d wb_orders >/dev/null 2>&1; do
  sleep 1
done

echo "waiting kafka..."
until nc -z kafka 9092 >/dev/null 2>&1; do
  sleep 1
done

echo "run migrations..."
/app/migrate up

echo "start service..."
exec /app/wb-order-service
