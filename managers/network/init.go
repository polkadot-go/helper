// managers/network/init.go
package network

import (
	"context"

	"github.com/polkadot-go/helper/core"
	"github.com/polkadot-go/helper/core/config"
	"github.com/polkadot-go/helper/data/mysql"
)

type networkComponent struct{}

func (c *networkComponent) Name() string {
	return "network_manager"
}

func (c *networkComponent) Dependencies() []string {
	return []string{"config", "logger", "mysql"}
}

func (c *networkComponent) Init() error {
	cfg := config.Get()

	mysqlStore := mysql.Get()
	instance = New(mysqlStore)

	interval := cfg.GetDuration("network", "check_interval")
	if interval > 0 {
		instance.interval = interval
	}

	instance.Start()

	core.RegisterHealthCheck("network_manager", instance)
	return nil
}

func (c *networkComponent) Shutdown(ctx context.Context) error {
	if instance != nil {
		instance.Stop()
	}
	return nil
}

func init() {
	config.Register("network", config.Schema{
		"check_interval": config.Field{
			Default:     "30s",
			Required:    false,
			Description: "Network check interval",
		},
		"timeout": config.Field{
			Default:     "10s",
			Required:    false,
			Description: "Network check timeout",
		},
		"max_retries": config.Field{
			Default:     3,
			Required:    false,
			Description: "Maximum retry attempts",
		},
	})

	core.Register(&networkComponent{})
}
