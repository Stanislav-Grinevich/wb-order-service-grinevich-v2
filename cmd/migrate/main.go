// Package main представляет собой небольшой помощник для наката/отката миграций.
// Использует POSTGRES_DSN и файлы в папке migrations.
package main

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"

	"github.com/jackc/pgx/v5"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("нужно указать команду: up или down")
	}

	cmd := os.Args[1]
	if cmd != "up" && cmd != "down" {
		log.Fatalf("неизвестная команда %q, нужно up или down", cmd)
	}

	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		log.Fatal("env POSTGRES_DSN не задан (см. .env.example)")
	}

	ctx := context.Background()

	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		log.Fatalf("ошибка подключения к postgres: %v", err)
	}
	defer conn.Close(ctx)

	dir := "migrations"
	pattern := "*.up.sql"
	if cmd == "down" {
		pattern = "*.down.sql"
	}

	paths, err := readMigrationFiles(dir, pattern)
	if err != nil {
		log.Fatalf("ошибка чтения миграций: %v", err)
	}

	if len(paths) == 0 {
		log.Fatalf("в папке %s нет файлов %s", dir, pattern)
	}

	// для up: от меньшего к большему
	// для down: откатываем в обратном порядке
	if cmd == "down" {
		for i, j := 0, len(paths)-1; i < j; i, j = i+1, j-1 {
			paths[i], paths[j] = paths[j], paths[i]
		}
	}

	log.Printf("выполняем миграции (%s):", cmd)
	for _, p := range paths {
		if err := runFile(ctx, conn, p); err != nil {
			log.Fatalf("ошибка в миграции %s: %v", p, err)
		}
	}
	log.Println("готово")
}

// readMigrationFiles собирает список файлов миграций по маске.
func readMigrationFiles(dir, pattern string) ([]string, error) {
	var files []string

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		match, err := filepath.Match(pattern, filepath.Base(path))
		if err != nil {
			return err
		}

		if match {
			files = append(files, path)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Strings(files)
	return files, nil
}

// runFile читает файл и выполняет его целиком как один запрос.
func runFile(ctx context.Context, conn *pgx.Conn, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	log.Printf("-> %s", filepath.Base(path))
	_, err = conn.Exec(ctx, string(data))
	if err != nil {
		return fmt.Errorf("sql error: %w", err)
	}

	return nil
}
