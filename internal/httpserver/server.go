// Package httpserver поднимает HTTP API и раздаёт статику.
package httpserver

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/Stanislav-Grinevich/wb-order-service-grinevich/internal/cache"
	"github.com/Stanislav-Grinevich/wb-order-service-grinevich/internal/repo"

	"github.com/go-chi/chi/v5"
)

// Server инкапсулирует кэш, репозиторий и роутер.
type Server struct {
	cache cache.OrderCache
	repo  repo.OrdersStorage
	mux   *chi.Mux
}

// New создаёт новый http-сервер.
func New(c cache.OrderCache, r repo.OrdersStorage) *Server {
	s := &Server{
		cache: c,
		repo:  r,
		mux:   chi.NewRouter(),
	}
	s.routes()
	return s
}

// routes настраивает хендлеры.
func (s *Server) routes() {
	s.mux.Get("/", s.handleIndex)
	s.mux.Get("/order/{id}", s.handleGetOrder)
}

// handleIndex отдаёт простую html страницу.
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "web/index.html")
}

// handleGetOrder ищет заказ по order_uid.
func (s *Server) handleGetOrder(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "empty id", http.StatusBadRequest)
		return
	}

	// сначала ищем в кэше
	if o, ok := s.cache.Get(id); ok {
		writeJSON(w, o)
		return
	}

	// если в кэше нет, то идём в БД
	o, err := s.repo.GetOrder(r.Context(), id)
	if err != nil {
		log.Printf("get order %s error: %v", id, err)
		http.Error(w, "order not found", http.StatusNotFound)
		return
	}

	writeJSON(w, o)
}

// writeJSON возвращает объект в JSON с отступами.
func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

// Handler возвращает объект http.Handler для запуска сервера.
func (s *Server) Handler() http.Handler {
	return s.mux
}
