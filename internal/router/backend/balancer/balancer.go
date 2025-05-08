package balancer

import (
	"errors"

	"github.com/DblMOKRQ/cloud_test_task/internal/models"
	random "github.com/DblMOKRQ/cloud_test_task/internal/router/backend/balancer/random_distribution"
	roundrobin "github.com/DblMOKRQ/cloud_test_task/internal/router/backend/balancer/round_robin"
)

type balancer interface {
	Next() *models.Server
}

func GetAlgorithm(algorithm string, servers []*models.Server) (balancer, error) {
	switch algorithm {
	case "roundrobin":
		return roundrobin.NewRoundRobin(servers)
	case "random":
		return random.NewRandom(servers)
	default:
		return nil, errors.New("invalid algorithm")
	}
}
