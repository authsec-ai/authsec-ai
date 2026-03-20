package services

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/sony/gobreaker/v2"
)

// Circuit breakers for external service calls.
// Each breaker opens after 5 consecutive failures and stays open for 30 seconds
// before allowing a single probe request.
var (
	hydraBreaker = gobreaker.NewCircuitBreaker[*http.Response](gobreaker.Settings{
		Name:        "hydra",
		MaxRequests: 1,
		Interval:    60 * time.Second,
		Timeout:     30 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			return counts.ConsecutiveFailures >= 5
		},
		OnStateChange: func(name string, from, to gobreaker.State) {
			log.Printf("Circuit breaker %s: %s → %s", name, from, to)
		},
	})
)

// httpClient is a shared client with sensible timeout for external calls.
var httpClient = &http.Client{Timeout: 30 * time.Second}

// CircuitDo executes an HTTP request through the Hydra circuit breaker.
func CircuitDoHydra(req *http.Request) (*http.Response, error) {
	return hydraBreaker.Execute(func() (*http.Response, error) {
		resp, err := httpClient.Do(req)
		if err != nil {
			return nil, err
		}
		// Treat 5xx as failures so the breaker can detect backend outages.
		if resp.StatusCode >= 500 {
			return resp, fmt.Errorf("hydra returned %d", resp.StatusCode)
		}
		return resp, nil
	})
}
