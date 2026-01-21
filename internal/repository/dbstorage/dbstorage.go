package dbstorage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/config"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/handler/pinghandler"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/model"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/retry"
	"github.com/kazakovdmitriy/go-musthave-metrics/internal/service"
	"go.uber.org/zap"
)

var _ service.Storage = (*dbstorage)(nil)
var _ pinghandler.HealthChecker = (*dbstorage)(nil)

type dbstorage struct {
	db       *pgxpool.Pool
	log      *zap.Logger
	retryCfg retry.RetryConfig
}

func NewDBStorage(db *pgxpool.Pool, log *zap.Logger, cfg *config.ServerFlags) (*dbstorage, error) {
	retryDelays, err := cfg.GetRetryDelaysAsDuration()
	if err != nil {
		return nil, err
	}

	storage := &dbstorage{
		db:  db,
		log: log,
		retryCfg: retry.RetryConfig{
			MaxRetries:    cfg.MaxRetries,
			Delays:        retryDelays,
			IsRetryableFn: isConnectionError,
		},
	}
	return storage, nil
}

func (db *dbstorage) UpdateGauge(ctx context.Context, name string, value float64) error {
	query := `
		INSERT INTO metrics (id, mtype, value)
		VALUES ($1, 'gauge', $2)
		ON CONFLICT (id) DO UPDATE
		SET value = EXCLUDED.value;
	`

	err := retry.Do(ctx, db.retryCfg, func() error {
		_, execErr := db.db.Exec(ctx, query, name, value)
		return execErr
	})

	if err != nil {
		db.log.Error(
			"failed to update gauge",
			zap.Error(err),
			zap.String("metric name", name),
			zap.Float64("value", value),
		)
		return err
	}

	return nil
}

func (db *dbstorage) UpdateCounter(ctx context.Context, name string, value int64) error {
	query := `
		INSERT INTO metrics (id, mtype, delta)
		VALUES ($1, 'counter', $2)
		ON CONFLICT (id) DO UPDATE
		SET delta = metrics.delta + $2;
	`

	err := retry.Do(ctx, db.retryCfg, func() error {
		_, execErr := db.db.Exec(ctx, query, name, value)
		return execErr
	})

	if err != nil {
		db.log.Error(
			"failed to update counter",
			zap.Error(err),
			zap.String("metric name", name),
			zap.Int64("metric value", value),
		)

		return err
	}

	return nil
}

func (db *dbstorage) UpdateMetrics(ctx context.Context, metrics []model.Metrics) error {
	err := retry.Do(ctx, db.retryCfg, func() error {
		tx, txErr := db.db.Begin(ctx)
		if txErr != nil {
			return txErr
		}
		defer tx.Rollback(ctx)

		gaugeQuery := `
			INSERT INTO metrics (id, mtype, value)
			VALUES ($1, 'gauge', $2)
			ON CONFLICT (id) DO UPDATE
			SET value = EXCLUDED.value;`

		counterQuery := `
			INSERT INTO metrics (id, mtype, delta)
			VALUES ($1, 'counter', $2)
			ON CONFLICT (id) DO UPDATE
			SET delta = metrics.delta + EXCLUDED.delta;`

		for _, metric := range metrics {
			switch metric.MType {
			case model.Gauge:
				if metric.Value == nil {
					db.log.Warn("gauge metric value is nil, skipping",
						zap.String("metric_id", metric.ID))
					continue
				}
				_, err := tx.Exec(ctx, gaugeQuery, metric.ID, *metric.Value)
				if err != nil {
					db.log.Error("failed to update gauge metric in batch",
						zap.Error(err),
						zap.String("metric_id", metric.ID),
						zap.Float64("value", *metric.Value))
					return fmt.Errorf("failed to update gauge metric %s: %w", metric.ID, err)
				}

			case model.Counter:
				if metric.Delta == nil {
					db.log.Warn("counter metric delta is nil, skipping",
						zap.String("metric_id", metric.ID))
					continue
				}
				_, err := tx.Exec(ctx, counterQuery, metric.ID, *metric.Delta)
				if err != nil {
					db.log.Error("failed to update counter metric in batch",
						zap.Error(err),
						zap.String("metric_id", metric.ID),
						zap.Int64("delta", *metric.Delta))
					return fmt.Errorf("failed to update counter metric %s: %w", metric.ID, err)
				}

			default:
				db.log.Warn("unknown metric type, skipping",
					zap.String("metric_type", metric.MType),
					zap.String("metric_id", metric.ID))
			}
		}

		return tx.Commit(ctx)
	})

	if err != nil {
		db.log.Error("failed to update metrics batch after retries", zap.Error(err))
		return err
	}

	db.log.Info("successfully updated metrics batch",
		zap.Int("metrics_count", len(metrics)))
	return nil
}

func (db *dbstorage) GetGauge(ctx context.Context, name string) (float64, bool) {
	var value float64
	var found bool

	err := retry.Do(ctx, db.retryCfg, func() error {
		err := db.db.QueryRow(ctx, `SELECT value FROM metrics WHERE id = $1 AND mtype = 'gauge';`, name).Scan(&value)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil
			}
			return err
		}
		found = true
		return nil
	})

	if err != nil {
		db.log.Error("failed to get gauge after retries", zap.Error(err), zap.String("metric_name", name))
		return 0.0, false
	}

	if !found {
		return 0.0, false
	}
	return value, true
}

func (db *dbstorage) GetCounter(ctx context.Context, name string) (int64, bool) {
	var delta int64

	err := retry.Do(ctx, db.retryCfg, func() error {
		err := db.db.QueryRow(ctx, `SELECT delta FROM metrics WHERE id = $1 AND mtype = 'counter';`, name).Scan(&delta)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil
			}
			return err
		}
		return nil
	})

	if err != nil {
		db.log.Error("failed to get counter after retries", zap.Error(err), zap.String("metric_name", name))
		return 0, false
	}

	return delta, true
}

func (db *dbstorage) GetAllMetrics(ctx context.Context) (string, error) {
	var builder strings.Builder

	err := retry.Do(ctx, db.retryCfg, func() error {
		rows, err := db.db.Query(ctx, `SELECT id, mtype, delta, value FROM metrics;`)
		if err != nil {
			return err
		}
		defer rows.Close()

		found := false
		builder.WriteString("<ul>\n")

		for rows.Next() {
			found = true
			var id, mtype string
			var delta sql.NullInt64
			var value sql.NullFloat64

			if err := rows.Scan(&id, &mtype, &delta, &value); err != nil {
				return fmt.Errorf("failed to scan metric row: %w", err)
			}

			// Минимизируем WriteString вызовы
			builder.WriteString("<li>")
			builder.WriteString(id)
			builder.WriteString(" = ")

			if mtype == "counter" && delta.Valid {
				// Используем AppendInt для минимальной аллокации
				buf := strconv.AppendInt(make([]byte, 0, 20), delta.Int64, 10)
				builder.Write(buf)
			} else if mtype == "gauge" && value.Valid {
				// Форматируем с фиксированной точностью
				buf := strconv.AppendFloat(make([]byte, 0, 32), value.Float64, 'f', 2, 64)
				builder.Write(buf)
			}
			builder.WriteString("</li>\n")
		}

		if err = rows.Err(); err != nil {
			return fmt.Errorf("row iteration error: %w", err)
		}

		if !found {
			return fmt.Errorf("no metrics found")
		}

		builder.WriteString("</ul>\n")
		return nil
	})

	if err != nil {
		db.log.Error("failed to get all metrics after retries", zap.Error(err))
		return "", err
	}

	return builder.String(), nil
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

func isConnectionError(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == pgerrcode.AdminShutdown ||
			pgErr.Code == pgerrcode.CannotConnectNow ||
			pgErr.Code == pgerrcode.ConnectionException ||
			pgErr.Code == pgerrcode.ConnectionDoesNotExist ||
			pgErr.Code == pgerrcode.ConnectionFailure ||
			pgErr.Code == pgerrcode.SQLClientUnableToEstablishSQLConnection ||
			pgErr.Code == pgerrcode.SQLServerRejectedEstablishmentOfSQLConnection ||
			pgErr.Code == pgerrcode.TransactionResolutionUnknown
	}
	return strings.Contains(err.Error(), "connection refused") ||
		strings.Contains(err.Error(), "network") ||
		strings.Contains(err.Error(), "timeout")
}
