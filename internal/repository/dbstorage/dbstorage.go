package dbstorage

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/handler/ping"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/service"
)

var _ service.Storage = (*dbstorage)(nil)
var _ ping.HealthChecker = (*dbstorage)(nil)

type dbstorage struct {
	db *pgxpool.Pool
}

func NewDBStorage(db *pgxpool.Pool) *dbstorage {
	storage := &dbstorage{
		db: db,
	}

	return storage
}

func (db *dbstorage) UpdateGauge(name string, value float64) {
	// Позже будет реализация
}

func (db *dbstorage) UpdateCounter(name string, value int64) {
	// Позже будет реализация
}

func (db *dbstorage) GetGauge(name string) (float64, bool) {
	// Позже будет реализация
	return 0, false
}

func (db *dbstorage) GetCounter(name string) (int64, bool) {
	// Позже будет реализация
	return 0.0, false
}
func (db *dbstorage) GetAllMetrics() (string, error) {
	// Позже будет реализация
	return "", nil
}

func (db *dbstorage) Ping(ctx context.Context) error {
	if db == nil || db.db == nil {
		return fmt.Errorf("database not connected")
	}
	return db.db.Ping(ctx)
}

func (db *dbstorage) Close() error {
	// Позже будет реализация
	db.db.Close()
	return nil
}
