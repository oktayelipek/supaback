package appstate

import (
	"sync"

	"github.com/supaback/supaback/internal/config"
	"github.com/supaback/supaback/internal/destination"
)

// State holds mutable app-wide config and destination, safe for concurrent use.
type State struct {
	mu   sync.RWMutex
	cfg  *config.Config
	dest destination.Destination
}

func New(cfg *config.Config, dest destination.Destination) *State {
	return &State{cfg: cfg, dest: dest}
}

func (s *State) Get() (*config.Config, destination.Destination) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cfg, s.dest
}

func (s *State) Update(cfg *config.Config, dest destination.Destination) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cfg = cfg
	s.dest = dest
}
