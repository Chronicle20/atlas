package database

import (
	"errors"
	"net"
	"syscall"

	"github.com/jackc/pgx/v5/pgconn"
)

// transientSQLStates are acquire-phase SQLSTATEs that are safe to retry: the
// server rejected the connection before any statement was sent. Any SQLSTATE
// produced after a statement began executing is deliberately absent.
var transientSQLStates = map[string]bool{
	"53300": true, // too_many_connections / reserved connection slots
	"57P03": true, // cannot_connect_now (server starting up / shutting down)
	"08001": true, // sqlclient_unable_to_establish_sqlconnection
	"08006": true, // connection_failure during establishment
}

// IsTransientConnectionError reports whether err is a connection-acquire-phase
// failure that is safe to retry (no statement was ever sent). Coded server
// errors are classified strictly by SQLSTATE — checked before the connect-error
// shape so an auth failure raised during connect is NOT transient. A bare
// context.DeadlineExceeded is ambiguous (could be mid-query) and is NOT
// transient; a deadline inside a *pgconn.ConnectError IS, because the connect
// provably never completed.
func IsTransientConnectionError(err error) bool {
	if err == nil {
		return false
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return transientSQLStates[pgErr.Code]
	}
	var connectErr *pgconn.ConnectError
	if errors.As(err, &connectErr) {
		return true
	}
	var opErr *net.OpError
	if errors.As(err, &opErr) && opErr.Op == "dial" {
		return true
	}
	if errors.Is(err, syscall.ECONNREFUSED) || errors.Is(err, syscall.ECONNRESET) {
		return true
	}
	return false
}

// TransientSQLState returns the SQLSTATE that classified err transient, or ""
// when classification came from a dial/connect error shape. Used for metric
// labels.
func TransientSQLState(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && transientSQLStates[pgErr.Code] {
		return pgErr.Code
	}
	return ""
}
