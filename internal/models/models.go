package models

import (
	"fmt"
	"net/url"
	"sync"
)

type Server struct {
	URL   *url.URL
	Alive bool
	Mu    sync.RWMutex
}

// NewServers создает список серверов из переданных URL.
// Возвращает ошибку при некорректных URL.
func NewServers(URLs []string) ([]*Server, error) {
	servers := make([]*Server, len(URLs))
	for i, u := range URLs {
		ur, err := url.Parse(u)
		if err != nil {
			return nil, fmt.Errorf("failed to parse URL: %v", err)
		}
		servers[i] = &Server{URL: ur, Alive: true}
	}
	return servers, nil
}
func (s *Server) SetAlive(status bool) {
	s.Mu.Lock()
	defer s.Mu.Unlock()
	s.Alive = status
}
