// managers/network/network.go
package network

import (
	"context"
	"sync"
	"time"

	"github.com/polkadot-go/helper/core"
	"github.com/polkadot-go/helper/data"
)

type NetworkManager struct {
	store    data.SQLStore
	logger   *core.Logger
	stopCh   chan struct{}
	wg       sync.WaitGroup
	interval time.Duration
}

var instance *NetworkManager

func Get() *NetworkManager {
	return instance
}

func New(store data.SQLStore) *NetworkManager {
	return &NetworkManager{
		store:    store,
		logger:   core.GetLogger("network"),
		stopCh:   make(chan struct{}),
		interval: 30 * time.Second,
	}
}

func (n *NetworkManager) Start() {
	n.wg.Add(1)
	go n.monitor()
	n.logger.Info("Network manager started")
}

func (n *NetworkManager) Stop() {
	close(n.stopCh)
	n.wg.Wait()
	n.logger.Info("Network manager stopped")
}

func (n *NetworkManager) monitor() {
	defer n.wg.Done()
	ticker := time.NewTicker(n.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			n.checkNetwork()
		case <-n.stopCh:
			return
		}
	}
}

func (n *NetworkManager) checkNetwork() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	start := time.Now()

	// Example network check - verify database connection
	rows, err := n.store.Query(ctx, "SELECT 1")
	if err != nil {
		n.logger.Error("Network check failed: %v", err)
		core.IncrCounter("network.check.failed")
	} else {
		rows.Close()
	}

	core.RecordDuration("network.check", start)
	core.IncrCounter("network.checks")

	n.logger.Debug("Network check completed")
}

func (n *NetworkManager) HealthCheck(ctx context.Context) (core.HealthStatus, error) {
	// Check if monitoring is running
	select {
	case <-n.stopCh:
		return core.HealthUnhealthy, nil
	default:
		return core.HealthHealthy, nil
	}
}
