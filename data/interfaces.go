// data/interfaces.go
package data

import (
	"context"
	"database/sql"
	"time"
)

type Store interface {
	Connect(ctx context.Context) error
	Close() error
	Get(ctx context.Context, key string) (interface{}, error)
	Set(ctx context.Context, key string, value interface{}) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
}

type SQLStore interface {
	Store
	Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row
	Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	Begin(ctx context.Context) (*sql.Tx, error)
}

type CacheStore interface {
	Store
	SetWithTTL(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	GetMulti(ctx context.Context, keys []string) (map[string]interface{}, error)
	Increment(ctx context.Context, key string, delta int64) (int64, error)
	Decrement(ctx context.Context, key string, delta int64) (int64, error)
}

type StoreConfig interface {
	GetString(key string) string
	GetInt(key string) int
	GetBool(key string) bool
	GetDuration(key string) time.Duration
}
