package database

import (
	"database/sql"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sirupsen/logrus"
)

var (
	acquireRetriesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "atlas_db_acquire_retries_total",
			Help: "Number of retried transient connection-acquire failures, by SQLSTATE (empty label for dial-shape errors).",
		},
		[]string{"sqlstate"},
	)

	transientErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "atlas_db_transient_errors_total",
			Help: "Number of errors classified as transient connection failures, by SQLSTATE (empty label for dial-shape errors).",
		},
		[]string{"sqlstate"},
	)
)

// CountTransient increments the transient-error counter for err. Call only
// after IsTransientConnectionError(err) has returned true.
func CountTransient(err error) {
	transientErrorsTotal.WithLabelValues(TransientSQLState(err)).Inc()
}

// registerDBStats exposes the standard sql.DBStats gauge family (go_sql_*)
// for db on the default Prometheus registry. Registration failure (e.g. a
// duplicate registration in tests) logs a warning and continues.
func registerDBStats(l logrus.FieldLogger, db *sql.DB, dbName string) {
	if err := prometheus.DefaultRegisterer.Register(collectors.NewDBStatsCollector(db, dbName)); err != nil {
		l.WithError(err).Warnf("Unable to register DB stats collector.")
	}
}
