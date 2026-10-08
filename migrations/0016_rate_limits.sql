-- +goose Up

CREATE TABLE platform_rate_limits (
    key          TEXT NOT NULL,
    window_start TIMESTAMPTZ NOT NULL,
    count        INT NOT NULL DEFAULT 0,
    PRIMARY KEY (key, window_start)
);

CREATE INDEX platform_rate_limits_cleanup_idx
    ON platform_rate_limits (window_start);

-- +goose Down

DROP TABLE platform_rate_limits;