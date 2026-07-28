package database

import (
	"context"
	"database/sql/driver"
	"time"

	"github.com/sirupsen/logrus"

	retry "github.com/Chronicle20/atlas/libs/atlas-retry"
)

type retryConnector struct {
	l        logrus.FieldLogger
	base     driver.Connector
	attempts int
	cfg      retry.Config
}

// newRetryConnector wraps base so transient acquire-phase failures (per
// IsTransientConnectionError) are retried with jittered backoff.
// database/sql invokes Connector.Connect only when the pool needs a new
// physical connection — before any SQL is sent on it — so nothing retried
// here can double-apply work. DB_ACQUIRE_RETRY_ATTEMPTS <= 1 disables the
// wrapper entirely.
func newRetryConnector(l logrus.FieldLogger, base driver.Connector) driver.Connector {
	attempts := getIntEnv("DB_ACQUIRE_RETRY_ATTEMPTS", 3)
	if attempts <= 1 {
		return base
	}
	cfg := retry.DefaultConfig().
		WithMaxRetries(attempts).
		WithInitialDelay(getDurationEnv("DB_ACQUIRE_RETRY_INITIAL_DELAY", 100*time.Millisecond)).
		WithMaxDelay(getDurationEnv("DB_ACQUIRE_RETRY_MAX_DELAY", 400*time.Millisecond))
	return &retryConnector{l: l, base: base, attempts: attempts, cfg: cfg}
}

func (c *retryConnector) Connect(ctx context.Context) (driver.Conn, error) {
	var conn driver.Conn
	err := retry.Try(ctx, c.cfg, func(attempt int) (bool, error) {
		var err error
		conn, err = c.base.Connect(ctx)
		if err == nil {
			return false, nil
		}
		if !IsTransientConnectionError(err) {
			return false, err
		}
		CountTransient(err)
		if attempt < c.attempts {
			acquireRetriesTotal.WithLabelValues(TransientSQLState(err)).Inc()
			c.l.WithError(err).Warnf("Transient DB connection acquire failure (SQLSTATE [%s]); retrying.", TransientSQLState(err))
		}
		return true, err
	})
	if err != nil {
		return nil, err
	}
	return conn, nil
}

func (c *retryConnector) Driver() driver.Driver { return c.base.Driver() }
