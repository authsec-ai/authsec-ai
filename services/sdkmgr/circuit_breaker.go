package sdkmgr

import (
	"sync"
	"time"

	"github.com/sony/gobreaker/v2"
)

// BreakerSettings defines configuration for a named circuit breaker.
type BreakerSettings struct {
	MaxRequests  uint32        // half-open: max concurrent probes
	Interval     time.Duration // closed-state counter reset interval
	Timeout      time.Duration // open→half-open transition timeout
	MaxFailures  uint32        // failures before tripping
}

var (
	breakers   = make(map[string]*gobreaker.CircuitBreaker[any])
	breakersMu sync.RWMutex
)

// DefaultBreakers mirrors the Python sdk-manager's 5 named breakers.
var DefaultBreakers = map[string]BreakerSettings{
	"auth_manager":      {MaxRequests: 1, Interval: 60 * time.Second, Timeout: 60 * time.Second, MaxFailures: 5},
	"external_services": {MaxRequests: 1, Interval: 30 * time.Second, Timeout: 30 * time.Second, MaxFailures: 3},
	"openai":            {MaxRequests: 1, Interval: 45 * time.Second, Timeout: 45 * time.Second, MaxFailures: 5},
	"mcp_server":        {MaxRequests: 1, Interval: 30 * time.Second, Timeout: 30 * time.Second, MaxFailures: 3},
	"database":          {MaxRequests: 1, Interval: 120 * time.Second, Timeout: 120 * time.Second, MaxFailures: 10},
}

// GetBreaker returns or creates a circuit breaker with the given name.
func GetBreaker(name string) *gobreaker.CircuitBreaker[any] {
	breakersMu.RLock()
	cb, ok := breakers[name]
	breakersMu.RUnlock()
	if ok {
		return cb
	}

	settings, exists := DefaultBreakers[name]
	if !exists {
		settings = BreakerSettings{
			MaxRequests: 1,
			Interval:    30 * time.Second,
			Timeout:     30 * time.Second,
			MaxFailures: 3,
		}
	}

	breakersMu.Lock()
	defer breakersMu.Unlock()

	// Double-check after acquiring write lock.
	if cb, ok := breakers[name]; ok {
		return cb
	}

	maxFail := settings.MaxFailures
	cb = gobreaker.NewCircuitBreaker[any](gobreaker.Settings{
		Name:        name,
		MaxRequests: settings.MaxRequests,
		Interval:    settings.Interval,
		Timeout:     settings.Timeout,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= maxFail
		},
	})
	breakers[name] = cb
	return cb
}

// BreakerStatus returns the state of all registered circuit breakers.
func BreakerStatus() map[string]string {
	breakersMu.RLock()
	defer breakersMu.RUnlock()

	status := make(map[string]string, len(breakers))
	for name, cb := range breakers {
		status[name] = cb.State().String()
	}
	return status
}
