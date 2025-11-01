package dbstorage

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/handler/ping"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/service"
	"go.uber.org/zap"
)

var _ service.Storage = (*dbstorage)(nil)
var _ ping.HealthChecker = (*dbstorage)(nil)

type dbstorage struct {
	db  *pgxpool.Pool
	log *zap.Logger
}

func NewDBStorage(db *pgxpool.Pool, log *zap.Logger) *dbstorage {
	storage := &dbstorage{
		db:  db,
		log: log,
	}
	return storage
}

func (db *dbstorage) UpdateGauge(ctx context.Context, name string, value float64) {
	query := `
		INSERT INTO metrics (id, mtype, value)
		VALUES ($1, 'gauge', $2)
		ON CONFLICT (id) DO UPDATE
		SET value = EXCLUDED.value;
	`
	_, err := db.db.Exec(ctx, query, name, value)
	if err != nil {
		db.log.Error(
			"failed to update gauge",
			zap.Error(err),
			zap.String("metric name", name),
			zap.Float64("value", value),
		)
	}
}

func (db *dbstorage) UpdateCounter(ctx context.Context, name string, value int64) {
	query := `
		INSERT INTO metrics (id, mtype, delta)
		VALUES ($1, 'counter', $2)
		ON CONFLICT (id) DO UPDATE
		SET delta = metrics.delta + $2;
	`

	_, err := db.db.Exec(ctx, query, name, value)
	if err != nil {
		db.log.Error(
			"failed to update counter",
			zap.Error(err),
			zap.String("metric name", name),
			zap.Int64("metric value", value),
		)
	}
}

func (db *dbstorage) GetGauge(ctx context.Context, name string) (float64, bool) {
	query := `SELECT value FROM metrics WHERE id = $1 AND mtype = 'gauge';`
	var value float64
	err := db.db.QueryRow(ctx, query, name).Scan(&value)
	if err != nil {
		return 0.0, false
	}
	return value, true
}

func (db *dbstorage) GetCounter(ctx context.Context, name string) (int64, bool) {
	query := `SELECT delta FROM metrics WHERE id = $1 AND mtype = 'counter';`
	var delta int64
	err := db.db.QueryRow(ctx, query, name).Scan(&delta)
	if err != nil {
		return 0, false
	}
	return delta, true
}

func (db *dbstorage) GetAllMetrics(ctx context.Context) (string, error) {
	query := `SELECT id, mtype, delta, value FROM metrics;`
	rows, err := db.db.Query(ctx, query)
	if err != nil {
		return "", fmt.Errorf("failed to query all metrics: %w", err)
	}
	defer rows.Close()

	result := "<ul>\n"
	found := false

	for rows.Next() {
		found = true
		var id, mtype string
		var delta sql.NullInt64
		var value sql.NullFloat64

		if err := rows.Scan(&id, &mtype, &delta, &value); err != nil {
			return "", fmt.Errorf("failed to scan metric row: %w", err)
		}

		switch mtype {
		case "counter":
			if delta.Valid {
				result += fmt.Sprintf("<li>%s = %d</li>\n", id, delta.Int64)
			}
		case "gauge":
			if value.Valid {
				result += fmt.Sprintf("<li>%s = %f</li>\n", id, value.Float64)
			}
		}
	}

	if err = rows.Err(); err != nil {
		return "", fmt.Errorf("row iteration error: %w", err)
	}

	if !found {
		return "", fmt.Errorf("no metrics found")
	}

	result += "</ul>\n"
	return result, nil
}

func (db *dbstorage) Ping(ctx context.Context) error {
	if db == nil || db.db == nil {
		return fmt.Errorf("database not connected")
	}
	return db.db.Ping(ctx)
}

func (db *dbstorage) Close() error {
	db.db.Close()
	return nil
}
