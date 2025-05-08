package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/DblMOKRQ/cloud_test_task/internal/config"
	"github.com/DblMOKRQ/cloud_test_task/internal/models"
	"github.com/DblMOKRQ/cloud_test_task/internal/router"

	"github.com/DblMOKRQ/cloud_test_task/internal/router/backend/balancer"
	"github.com/DblMOKRQ/cloud_test_task/internal/router/backend/healthcheck"
	logger "github.com/DblMOKRQ/cloud_test_task/pkg"
	"go.uber.org/zap"
)

func main() {

	log, err := logger.NewLogger(false)
	if err != nil {
		panic(err)
	}
	defer log.Sync()

	cfg := config.MustLoad()

	servers, err := models.NewServers(cfg.Backends)
	if err != nil {
		log.Error("Failed to create servers", zap.Error(err))
		return
	}
	algorithm, err := balancer.GetAlgorithm(cfg.Balancer.Algorithm, servers)

	if err != nil {
		log.Error("Failed to create balancer", zap.Error(err))
		return
	}

	hc := healthcheck.NewHealthChecker(
		cfg.HealthChecker.Interval,
		cfg.HealthChecker.Timeout,
		servers,
		log,
	)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	go hc.Run(ctx)

	rout, err := router.NewRouter(cfg, algorithm, log, hc)
	if err != nil {
		log.Error("Failed to create router", zap.Error(err))
		return
	}

	rout.Run()

	log.Info("Server Stopped")

}
