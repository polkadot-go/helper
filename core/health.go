package core

import (
	"context"
	"sync"
	"time"
)

type HealthStatus int

const (
	HealthUnknown HealthStatus = iota
	HealthHealthy
	HealthDegraded
	HealthUnhealthy
)

type HealthChecker interface {
	HealthCheck(ctx context.Context) (HealthStatus, error)
}

type HealthRegistry struct {
	mu       sync.RWMutex
	checkers map[string]HealthChecker
}

var healthRegistry = &HealthRegistry{
	checkers: make(map[string]HealthChecker),
}

func RegisterHealthCheck(name string, checker HealthChecker) {
	healthRegistry.mu.Lock()
	defer healthRegistry.mu.Unlock()
	healthRegistry.checkers[name] = checker
}

func CheckHealth(ctx context.Context) map[string]HealthResult {
	healthRegistry.mu.RLock()
	defer healthRegistry.mu.RUnlock()

	results := make(map[string]HealthResult)
	for name, checker := range healthRegistry.checkers {
		status, err := checker.HealthCheck(ctx)
		results[name] = HealthResult{
			Status: status,
			Error:  err,
			Time:   time.Now(),
		}
	}
	return results
}

type HealthResult struct {
	Status HealthStatus
	Error  error
	Time   time.Time
}
