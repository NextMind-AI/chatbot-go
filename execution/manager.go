package execution

import (
	"context"
	"sync"
	"time"

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

		// Actually implement the delay mentioned in the comment
		// Give time for the previous execution to clean up properly
		log.Info().Str("user_id", userID).Msg("Waiting for previous execution cleanup")
		time.Sleep(100 * time.Millisecond)
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
		log.Info().Str("user_id", userID).Msg("Cleaning up execution context")
		delete(m.userExecutions, userID)
	} else {
		log.Warn().Str("user_id", userID).Msg("Cleanup called but execution not found or context mismatch")
	}
}
