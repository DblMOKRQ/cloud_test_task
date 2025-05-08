package healthcheck

import (
	"context"
	"net/http"
	"time"

	"github.com/DblMOKRQ/cloud_test_task/internal/models"
	logger "github.com/DblMOKRQ/cloud_test_task/pkg"
	"go.uber.org/zap"
)

// HealthChecker реализует проверку состояния backend-серверов.
type HealthChecker struct {
	interval time.Duration
	timeout  time.Duration
	backends []*models.Server
	log      *logger.Logger
}

// NewHealthChecker создает новый экземпляр HealthChecker.
// Принимает интервал проверки, таймаут и список серверов.
func NewHealthChecker(interval time.Duration, timeout time.Duration, backends []*models.Server, log *logger.Logger) *HealthChecker {
	return &HealthChecker{
		interval: interval,
		timeout:  timeout,
		backends: backends,
		log:      log,
	}
}

// Run запускает периодические проверки состояния серверов.
// Работает до отмены контекста.
func (hc *HealthChecker) Run(ctx context.Context) {
	ticker := time.NewTicker(hc.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			for _, backend := range hc.backends {
				select {
				case <-ctx.Done():
					hc.log.Info("Healthchecker stopped")
					return
				default:
					hc.check(backend)
				}
			}
		case <-ctx.Done():
			hc.log.Info("Healthchecker stopped")
			return
		}
	}
}

func (hc *HealthChecker) check(backend *models.Server) {
	client := http.Client{
		Timeout: hc.interval,
	}
	resp, err := client.Get(backend.URL.String() + "/healthcheck")
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil || resp.StatusCode != http.StatusOK {
		hc.log.Error("Healthcheck failed for backend: ", zap.String("backend", backend.URL.String()))
		backend.SetAlive(false)
	} else {
		hc.log.Debug("Healthcheck passed for backend: ", zap.String("backend", backend.URL.String()))
		backend.SetAlive(true)
	}
}
