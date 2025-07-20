// data/mysql/init.go
package mysql

import (
	"context"
	"time"

	"github.com/polkadot-go/helper/core"
	"github.com/polkadot-go/helper/core/config"
)

type mysqlComponent struct{}

func (c *mysqlComponent) Name() string {
	return "mysql"
}

func (c *mysqlComponent) Dependencies() []string {
	return []string{"config", "logger"}
}

func (c *mysqlComponent) Init() error {
	cfg := config.Get()

	configAdapter := &mysqlConfig{cfg: cfg}
	instance = New(configAdapter)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := instance.Connect(ctx); err != nil {
		return err
	}

	core.RegisterHealthCheck("mysql", instance)
	return nil
}

func (c *mysqlComponent) Shutdown(ctx context.Context) error {
	if instance != nil {
		return instance.Close()
	}
	return nil
}

type mysqlConfig struct {
	cfg *config.Config
}

func (m *mysqlConfig) GetString(key string) string {
	return m.cfg.GetString("mysql", key)
}

func (m *mysqlConfig) GetInt(key string) int {
	return m.cfg.GetInt("mysql", key)
}

func (m *mysqlConfig) GetBool(key string) bool {
	return m.cfg.GetBool("mysql", key)
}

func (m *mysqlConfig) GetDuration(key string) time.Duration {
	return m.cfg.GetDuration("mysql", key)
}

func init() {
	config.Register("mysql", config.Schema{
		"host": config.Field{
			Default:     "localhost",
			Required:    true,
			Description: "MySQL host",
		},
		"port": config.Field{
			Default:     3306,
			Required:    false,
			Description: "MySQL port",
		},
		"user": config.Field{
			Default:     "root",
			Required:    true,
			Description: "MySQL user",
		},
		"password": config.Field{
			Default:     "",
			Required:    true,
			Description: "MySQL password",
		},
		"database": config.Field{
			Default:     "polkadot",
			Required:    true,
			Description: "MySQL database",
		},
		"max_connections": config.Field{
			Default:     25,
			Required:    false,
			Description: "Maximum connections",
		},
		"max_idle_connections": config.Field{
			Default:     5,
			Required:    false,
			Description: "Maximum idle connections",
		},
		"conn_max_lifetime": config.Field{
			Default:     "5m",
			Required:    false,
			Description: "Connection max lifetime",
		},
	})

	core.Register(&mysqlComponent{})
}
