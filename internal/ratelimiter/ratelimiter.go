package ratelimiter

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/DblMOKRQ/cloud_test_task/internal/router/errs"
	logger "github.com/DblMOKRQ/cloud_test_task/pkg"
	"github.com/go-redis/redis_rate/v10"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// RedisRateLimiter реализует ограничитель запросов на базе Redis.
type RedisRateLimiter struct {
	rdb          *redis.Client
	limiter      *redis_rate.Limiter
	defaultRate  int                         // Глобальный лимит по умолчанию
	defaultBurst int                         // Глобальный burst по умолчанию
	userLimits   map[string]redis_rate.Limit // Кэш индивидуальных лимитов
	mu           sync.RWMutex
	log          *logger.Logger
}

// InitRedisClient инициализирует Redis-клиент для ограничителя запросов.
// Возвращает готовый к работе экземпляр RedisRateLimiter.
func InitRedisClient(addr string, password string, log *logger.Logger, defaultRate int, defaultBurst int) *RedisRateLimiter {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       0,
	})

	limiter := redis_rate.NewLimiter(rdb)
	if _, err := rdb.Ping(context.Background()).Result(); err != nil {
		log.Fatal("Failed to connect to Redis", zap.Error(err))
	}
	return &RedisRateLimiter{
		rdb:          rdb,
		limiter:      limiter,
		defaultRate:  defaultRate,
		defaultBurst: defaultBurst,
		userLimits:   make(map[string]redis_rate.Limit),
		log:          log,
	}
}

// RateLimitMiddleware возвра middleware для ограничения запросов.
// Использует IP-адрес клиента как идентификатор.
func (rrl *RedisRateLimiter) Close() {
	if rrl.rdb != nil {
		_ = rrl.rdb.Close()
		rrl.rdb = nil
	}
}

// RateLimitMiddleware возвра middleware для ограничения запросов.
// Использует IP-адрес клиента как идентификатор.
func (rrl *RedisRateLimiter) RateLimitMiddleware(next http.Handler) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Идентификатор пользователя
		identifier, _, _ := net.SplitHostPort(r.RemoteAddr)

		rrl.mu.RLock()
		userLimit, exists := rrl.userLimits[identifier]
		rrl.mu.RUnlock()
		var limit redis_rate.Limit
		if exists {
			limit = userLimit
		} else {
			limit = redis_rate.Limit{
				Rate:   rrl.defaultRate,
				Period: time.Second,
				Burst:  rrl.defaultBurst,
			}
		}
		res, err := rrl.limiter.Allow(r.Context(), identifier, limit)
		if err != nil {
			_ = rrl.limiter.Reset(r.Context(), identifier)
			rrl.log.Error("Rate limit check failed",
				zap.String("identifier", identifier),
				zap.Error(err),
			)
			errs.JSONError(w, errs.ErrorResponse{Error: "Internal Server Error"}, http.StatusInternalServerError)
			return
		}

		// Логируем оставшиеся токены
		rrl.log.Debug("Rate limit status",
			zap.String("identifier", identifier),
			zap.Int("remaining", res.Remaining),
			zap.Int("limit", limit.Burst),
			zap.String("URL", r.URL.String()),
			zap.Duration("reset_in", res.ResetAfter),
		)

		if res.Allowed == 0 {
			rrl.log.Warn("Rate limit exceeded",
				zap.String("identifier", identifier),
				zap.Int("limit", rrl.defaultBurst),
				zap.String("URL", r.URL.String()),
			)
			errs.JSONError(w, errs.ErrorResponse{Error: "Rate limit exceeded"}, http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})

}

// SetUserLimit устанавливает кастомные лимиты для указанного пользователя.
// Возвращает ошибку при невалидных значениях лимитов.
func (rrl *RedisRateLimiter) SetUserLimit(userID string, newRate, newBurst int) error {
	if newRate <= 0 || newBurst <= 0 {
		return fmt.Errorf("rate and burst must be positive")
	}

	// Сохраняем новый лимит в кэше
	rrl.mu.Lock()
	defer rrl.mu.Unlock()
	rrl.userLimits[userID] = redis_rate.Limit{
		Rate:   newRate,
		Period: time.Second,
		Burst:  newBurst,
	}

	return nil
}
