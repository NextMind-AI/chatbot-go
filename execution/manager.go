package execution

import (
	"context"
	"sync"

	"github.com/rs/zerolog/log"
)

type UserExecution struct {
	ctx    context.Context
	cancel context.CancelFunc
}

type Manager struct {
	userExecutions map[string]*UserExecution
	mutex          sync.RWMutex
}

func NewManager() *Manager {
	return &Manager{
		userExecutions: make(map[string]*UserExecution),
	}
}

func (m *Manager) Start(userID string) context.Context {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if existingExecution, exists := m.userExecutions[userID]; exists {
		log.Info().Str("user_id", userID).Msg("Previous execution exists - cancelling it")
		existingExecution.cancel()
		// Give a brief moment for the previous execution to clean up
		// before starting the new one
	}

	ctx, cancel := context.WithCancel(context.Background())
	m.userExecutions[userID] = &UserExecution{
		ctx:    ctx,
		cancel: cancel,
	}

	log.Info().Str("user_id", userID).Msg("Started new execution context")
	return ctx
}

func (m *Manager) Cleanup(userID string, ctx context.Context) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if execution, exists := m.userExecutions[userID]; exists && execution.ctx == ctx {
		delete(m.userExecutions, userID)
	}
}
