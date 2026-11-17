package main

import (
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"

	"pr4_minenko/internal/handler"
	"pr4_minenko/internal/middleware"
	"pr4_minenko/internal/repository"
)

const addr = ":8080"

func main() {
	log.SetFlags(log.Ldate | log.Ltime)
	log.Println("=== Мережевий реєстр пристроїв (Device Registry API) ===")

	repo := repository.NewMemoryDeviceRepository()
	log.Println("[СЕРВЕР] Ініціалізація сховища даних (sync.RWMutex: active)")

	r := chi.NewRouter()
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(middleware.Logger)
	log.Println("[СЕРВЕР] Конвеєр Middleware підключено: RequestID, RealIP, Logger")

	devices := handler.NewDeviceHandler(repo)
	r.Mount("/api/v1/devices", devices.Routes())
	log.Println("[СЕРВЕР] Маршрути Chi змонтовано на префіксі /api/v1/devices")

	srv := &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
	}
	log.Printf("[СЕРВЕР] HTTP-сервер запущено на порту %s...", addr)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("[СЕРВЕР] Помилка: %v", err)
	}
}
