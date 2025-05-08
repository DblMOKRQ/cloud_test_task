package roundrobin

import (
	"sync"
	"sync/atomic"

	"github.com/DblMOKRQ/cloud_test_task/internal/models"
)

// RoundRobin реализует алгоритм балансировки Round Robin.
type RoundRobin struct {
	servers []*models.Server // Список серверов
	index   uint32           // Текущий индекс (атомарный)
	mu      sync.RWMutex     // Мьютекс для безопасного обновления списка серверов
}

// NewRoundRobin создает новый экземпляр балансировщика RoundRobin.
// Принимает список серверов для балансировки нагрузки.
func NewRoundRobin(servers []*models.Server) (*RoundRobin, error) {
	rr := &RoundRobin{servers: servers}

	return rr, nil
}

// Next возвращает следующий доступный сервер из списка.
// Возвращает nil, если нет доступных серверов.
func (rr *RoundRobin) Next() *models.Server {
	rr.mu.RLock()
	defer rr.mu.RUnlock()

	if len(rr.servers) == 0 {
		return nil
	}

	index := atomic.AddUint32(&rr.index, 1) % uint32(len(rr.servers))
	for i := uint32(0); i < uint32(len(rr.servers)); i++ {
		if rr.servers[index%uint32(len(rr.servers))].Alive {
			return rr.servers[index%uint32(len(rr.servers))]
		}
		index = (index + 1) % uint32(len(rr.servers))
	}
	return nil
}
