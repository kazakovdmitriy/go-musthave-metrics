CREATE TYPE metric_type AS ENUM ('counter', 'gauge');

CREATE TABLE metrics (
    id    TEXT PRIMARY KEY,
    mtype metric_type NOT NULL,
    delta BIGINT,
    value DOUBLE PRECISION,
    hash  TEXT,

    CONSTRAINT chk_metric CHECK (
        (mtype = 'counter' AND delta IS NOT NULL AND value IS NULL) OR
        (mtype = 'gauge'   AND delta IS NULL     AND value IS NOT NULL)
    )
);

CREATE INDEX idx_metrics_mtype ON metrics (mtype);