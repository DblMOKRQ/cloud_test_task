package router

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/DblMOKRQ/cloud_test_task/internal/config"
	"github.com/DblMOKRQ/cloud_test_task/internal/models"
	"github.com/DblMOKRQ/cloud_test_task/internal/ratelimiter"
	"github.com/DblMOKRQ/cloud_test_task/internal/router/backend/healthcheck"
	"github.com/DblMOKRQ/cloud_test_task/internal/router/errs"
	"github.com/DblMOKRQ/cloud_test_task/internal/router/proxy"
	logger "github.com/DblMOKRQ/cloud_test_task/pkg"
	"go.uber.org/zap"
)

type balancer interface {
	Next() *models.Server
}

// Router обрабатывает HTTP-запросы и управляет балансировкой.
type Router struct {
	Host       string
	Port       string
	RL         *ratelimiter.RedisRateLimiter
	bal        balancer
	log        *logger.Logger
	server     *http.Server
	hc         *healthcheck.HealthChecker
	shutdownWg sync.WaitGroup
	cfg        *config.Config
}

// NewRouter создает новый экземпляр роутера с настройками из конфига
// Возвращает ошибку если не удалось инициализировать компоненты
func NewRouter(cfg *config.Config, bal balancer, log *logger.Logger, hc *healthcheck.HealthChecker) (*Router, error) {
	mux := http.NewServeMux()

	rt := &Router{
		Host: cfg.Host,
		Port: cfg.Port,
		RL: ratelimiter.InitRedisClient(
			fmt.Sprintf("%s:%d", cfg.Storage.Redis.Host, cfg.Storage.Redis.Port),
			cfg.Storage.Redis.Password,
			log,
			cfg.Rate_limiting.Rate_per_second,
			cfg.Rate_limiting.Capacity,
		),
		bal: bal,
		log: log,
		hc:  hc,
		cfg: cfg,
	}
	mux.HandleFunc("/", rt.HandleRequest)
	mux.HandleFunc("/edit", rt.HandleEdit)
	handler := rt.RL.RateLimitMiddleware(mux)
	rt.server = &http.Server{
		Addr:    fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
		Handler: handler,
	}

	return rt, nil

}

// HandleRequest обрабатывает входящие HTTP-запросы.
// Перенаправляет запросы через балансировщик на backend-серверы.
func (rt *Router) HandleRequest(w http.ResponseWriter, r *http.Request) {
	backend := rt.bal.Next()
	if backend == nil {
		rt.log.Error("No backend available")
		errs.JSONError(w, errs.ErrorResponse{Error: "Service is unavailable"}, http.StatusBadGateway)
		return
	}

	proxy.Proxy(backend.URL, rt.log).ServeHTTP(w, r)
	rt.log.Info("Request proxied to ", zap.Any("URL", backend.URL))
}

// HandleEdit обрабатывает запросы на изменение лимитов.
// Принимает JSON с новыми значениями rate limit.
func (rt *Router) HandleEdit(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		errs.JSONError(w, errs.ErrorResponse{Error: "Only POST method is allowed"}, http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		UserIP   string `json:"userIP"`
		NewRate  int    `json:"newRate"`
		NewBurst int    `json:"newBurst"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		rt.log.Error("Failed to decode request", zap.Error(err))
		errs.JSONError(w, errs.ErrorResponse{Error: "Invalid request format"}, http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Валидация данных
	if request.UserIP == "" {
		errs.JSONError(w, errs.ErrorResponse{Error: "userIP is required"}, http.StatusBadRequest)
		return
	}

	if request.NewRate <= 0 || request.NewBurst <= 0 {
		errs.JSONError(w, errs.ErrorResponse{Error: "newRate and newBurst must be positive integers"}, http.StatusBadRequest)
		return
	}

	// Обновляем лимит
	if err := rt.RL.SetUserLimit(request.UserIP, request.NewRate, request.NewBurst); err != nil {
		rt.log.Error("Failed to update rate limit",
			zap.String("userIP", request.UserIP),
			zap.Error(err),
		)
		errs.JSONError(w, errs.ErrorResponse{Error: "Failed to update rate limit"}, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("Rate limit updated for %s: %d/s (burst %d)", request.UserIP, request.NewRate, request.NewBurst)))
}

// Run запускает HTTP-сервер и healthchecker.
// Обрабатывает graceful shutdown при получении сигналов ОС.
func (rt *Router) Run() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Запуск health checker
	rt.shutdownWg.Add(1)
	go func() {
		defer rt.shutdownWg.Done()
		rt.hc.Run(ctx)
	}()

	// Запуск HTTP сервера
	rt.shutdownWg.Add(1)
	go func() {
		defer rt.shutdownWg.Done()
		rt.log.Info("Starting server", zap.String("address", rt.server.Addr))
		if err := rt.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			rt.log.Error("Server error", zap.Error(err))
			stop() // Инициируем shutdown при ошибке
		}
	}()

	// Ожидание сигнала завершения
	<-ctx.Done()
	rt.log.Info("Shutting down server...")

	// Graceful shutdown HTTP сервера
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := rt.server.Shutdown(shutdownCtx); err != nil {
		rt.log.Error("Server shutdown error", zap.Error(err))
	}

	// Закрытие Redis соединения
	rt.RL.Close()

	// Ожидание завершения всех компонентов
	done := make(chan struct{})
	go func() {
		rt.shutdownWg.Wait()
		close(done)
	}()

	select {
	case <-done:
		rt.log.Info("Server stopped gracefully")

	}
}

func (rt *Router) GracefulShutdown(ctx context.Context) error {
	rt.log.Info("Graceful shutdown")

	return rt.server.Shutdown(ctx)
}
