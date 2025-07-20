// core/config/init.go
package config

import (
	"context"
	"fmt"
	"os"

	"github.com/polkadot-go/helper/core"
)

type configComponent struct {
	filename string
}

func (c *configComponent) Name() string {
	return "config"
}

func (c *configComponent) Dependencies() []string {
	return nil
}

func (c *configComponent) Init() error {
	if c.filename == "" {
		c.filename = "config.json"
	}

	err := Load(c.filename)
	if os.IsNotExist(err) {
		if err := SaveTemplate(c.filename); err != nil {
			return fmt.Errorf("failed to save config template: %w", err)
		}
		return Load(c.filename)
	}
	return err
}

func (c *configComponent) Shutdown(ctx context.Context) error {
	return nil
}

var component = &configComponent{}

func init() {
	Register("config", Schema{
		"log_level": Field{
			Default:     "info",
			Required:    false,
			Description: "Logging level",
			Validator: func(v interface{}) error {
				level, ok := v.(string)
				if !ok {
					return fmt.Errorf("log_level must be string")
				}
				valid := []string{"debug", "info", "warn", "error"}
				for _, vl := range valid {
					if level == vl {
						return nil
					}
				}
				return fmt.Errorf("invalid log_level: %s", level)
			},
		},
		"environment": Field{
			Default:     "development",
			Required:    false,
			Description: "Environment",
		},
		"shutdown_timeout": Field{
			Default:     "30s",
			Required:    false,
			Description: "Graceful shutdown timeout",
		},
	})

	core.Register(component)
}

func SetConfigFile(filename string) {
	component.filename = filename
}
