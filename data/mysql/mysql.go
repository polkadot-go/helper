// data/mysql/mysql.go
package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/polkadot-go/helper/core"
	"github.com/polkadot-go/helper/data"
)

type MySQL struct {
	db     *sql.DB
	config data.StoreConfig
	logger *core.Logger
}

var instance *MySQL

func Get() *MySQL {
	return instance
}

func New(cfg data.StoreConfig) *MySQL {
	return &MySQL{
		config: cfg,
		logger: core.GetLogger("mysql"),
	}
}

func (m *MySQL) Connect(ctx context.Context) error {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&charset=utf8mb4",
		m.config.GetString("user"),
		m.config.GetString("password"),
		m.config.GetString("host"),
		m.config.GetInt("port"),
		m.config.GetString("database"))

	var err error
	m.db, err = sql.Open("mysql", dsn)
	if err != nil {
		return err
	}

	m.db.SetMaxOpenConns(m.config.GetInt("max_connections"))
	m.db.SetMaxIdleConns(m.config.GetInt("max_idle_connections"))
	m.db.SetConnMaxLifetime(m.config.GetDuration("conn_max_lifetime"))

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err = m.db.PingContext(ctx)
	if err != nil {
		m.db.Close()
		return err
	}

	core.IncrCounter("mysql.connections")
	m.logger.Info("Connected to MySQL at %s:%d", m.config.GetString("host"), m.config.GetInt("port"))
	return nil
}

func (m *MySQL) Close() error {
	if m.db != nil {
		return m.db.Close()
	}
	return nil
}

func (m *MySQL) Get(ctx context.Context, key string) (interface{}, error) {
	var value string
	err := m.db.QueryRowContext(ctx, "SELECT value FROM kv WHERE key = ?", key).Scan(&value)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return value, err
}

func (m *MySQL) Set(ctx context.Context, key string, value interface{}) error {
	_, err := m.db.ExecContext(ctx,
		"INSERT INTO kv (key, value) VALUES (?, ?) ON DUPLICATE KEY UPDATE value = ?",
		key, value, value)
	return err
}

func (m *MySQL) Delete(ctx context.Context, key string) error {
	_, err := m.db.ExecContext(ctx, "DELETE FROM kv WHERE key = ?", key)
	return err
}

func (m *MySQL) Exists(ctx context.Context, key string) (bool, error) {
	var count int
	err := m.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM kv WHERE key = ?", key).Scan(&count)
	return count > 0, err
}

func (m *MySQL) Query(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	start := time.Now()
	rows, err := m.db.QueryContext(ctx, query, args...)
	core.RecordDuration("mysql.query", start)
	if err != nil {
		core.IncrCounter("mysql.errors")
	}
	return rows, err
}

func (m *MySQL) QueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	start := time.Now()
	row := m.db.QueryRowContext(ctx, query, args...)
	core.RecordDuration("mysql.query", start)
	return row
}

func (m *MySQL) Exec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	start := time.Now()
	result, err := m.db.ExecContext(ctx, query, args...)
	core.RecordDuration("mysql.exec", start)
	if err != nil {
		core.IncrCounter("mysql.errors")
	}
	return result, err
}

func (m *MySQL) Begin(ctx context.Context) (*sql.Tx, error) {
	return m.db.BeginTx(ctx, nil)
}

func (m *MySQL) HealthCheck(ctx context.Context) (core.HealthStatus, error) {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	if err := m.db.PingContext(ctx); err != nil {
		return core.HealthUnhealthy, err
	}

	var count int
	if err := m.db.QueryRowContext(ctx, "SELECT 1").Scan(&count); err != nil {
		return core.HealthDegraded, err
	}

	return core.HealthHealthy, nil
}
