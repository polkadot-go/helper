// core/logger_init.go
package core

import (
	"context"
)

type loggerComponent struct{}

func (l *loggerComponent) Name() string {
	return "logger"
}

func (l *loggerComponent) Dependencies() []string {
	return []string{"config"}
}

func (l *loggerComponent) Init() error {
	// Get config component directly
	if comp := GetComponent("config"); comp != nil {
		// We'll set log level after config is loaded
		// For now, default to info
		SetLogLevel("info")
	}

	return nil
}

func (l *loggerComponent) Shutdown(ctx context.Context) error {
	return nil
}

func init() {
	Register(&loggerComponent{})
}
