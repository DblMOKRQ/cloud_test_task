package randomdistribution

import (
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/DblMOKRQ/cloud_test_task/internal/models"
)

// Random реализует алгоритм балансировки случайного распределения.
type Random struct {
	servers []*models.Server
	mu      sync.RWMutex
	seed    uint64
}

// NewRandom создает новый балансировщик случайного распределения.
func NewRandom(servers []*models.Server) (*Random, error) {
	r := &Random{
		servers: servers,
	}
	atomic.StoreUint64(&r.seed, uint64(time.Now().UnixNano()))
	return r, nil
}

// Next возвращает случайный доступный сервер.
func (r *Random) Next() *models.Server {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.servers) == 0 {
		return nil
	}

	// Атомарное чтение и обновление seed
	oldSeed := atomic.LoadUint64(&r.seed)
	newSeed := rand.New(rand.NewSource(int64(oldSeed))).Uint64()
	atomic.CompareAndSwapUint64(&r.seed, oldSeed, newSeed)

	localRand := rand.New(rand.NewSource(int64(oldSeed)))

	index := localRand.Intn(len(r.servers))

	for i := 0; i < len(r.servers); i++ {
		currentIndex := (index + i) % len(r.servers)
		server := r.servers[currentIndex]

		server.Mu.RLock()
		alive := server.Alive
		server.Mu.RUnlock()

		if alive {
			return server
		}
	}
	return nil
}
