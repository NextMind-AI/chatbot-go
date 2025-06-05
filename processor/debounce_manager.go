package processor

import (
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// DebounceManager manages debounced message processing for users.
// For each user, it waits 15 seconds before processing their latest message.
// If a new message arrives during the wait period, the timer resets.
type DebounceManager struct {
	userTimers map[string]*userTimer
	mutex      sync.RWMutex
}

// userTimer holds the timer and cancellation function for a specific user
type userTimer struct {
	timer  *time.Timer
	cancel context.CancelFunc
}

// NewDebounceManager creates a new instance of DebounceManager
func NewDebounceManager() *DebounceManager {
	return &DebounceManager{
		userTimers: make(map[string]*userTimer),
	}
}

// ProcessMessage schedules message processing with a 15-second debounce.
// If another message comes from the same user within 15 seconds, the timer resets.
func (dm *DebounceManager) ProcessMessage(userID string, processor func()) {
	dm.mutex.Lock()
	defer dm.mutex.Unlock()

	// Cancel existing timer if it exists
	if existingTimer, exists := dm.userTimers[userID]; exists {
		log.Info().
			Str("user_id", userID).
			Msg("Resetting debounce timer - new message received")

		existingTimer.timer.Stop()
		existingTimer.cancel()
	}

	// Create a new context for cancellation
	ctx, cancel := context.WithCancel(context.Background())

	// Create a new timer for 15 seconds
	timer := time.AfterFunc(15*time.Second, func() {
		select {
		case <-ctx.Done():
			// Timer was cancelled, don't process
			log.Info().
				Str("user_id", userID).
				Msg("Debounce timer cancelled")
			return
		default:
			// Timer expired naturally, process the message
			log.Info().
				Str("user_id", userID).
				Msg("Debounce timer expired - processing message")

			// Clean up the timer from the map
			dm.cleanupTimer(userID)

			// Execute the processor function
			processor()
		}
	})

	// Store the timer and cancel function
	dm.userTimers[userID] = &userTimer{
		timer:  timer,
		cancel: cancel,
	}

	log.Info().
		Str("user_id", userID).
		Msg("Started 15-second debounce timer")
}

// cleanupTimer removes a timer from the map (called after timer expires or is cancelled)
func (dm *DebounceManager) cleanupTimer(userID string) {
	dm.mutex.Lock()
	defer dm.mutex.Unlock()
	delete(dm.userTimers, userID)
}

// CancelTimer cancels any pending timer for a user
func (dm *DebounceManager) CancelTimer(userID string) {
	dm.mutex.Lock()
	defer dm.mutex.Unlock()

	if timer, exists := dm.userTimers[userID]; exists {
		log.Info().
			Str("user_id", userID).
			Msg("Manually cancelling debounce timer")

		timer.timer.Stop()
		timer.cancel()
		delete(dm.userTimers, userID)
	}
}

// GetActiveTimersCount returns the number of active timers (for monitoring/debugging)
func (dm *DebounceManager) GetActiveTimersCount() int {
	dm.mutex.RLock()
	defer dm.mutex.RUnlock()
	return len(dm.userTimers)
}
