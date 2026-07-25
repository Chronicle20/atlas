package database

import (
	"context"
	"errors"
	"fmt"
	"net"
	"syscall"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

// connectRefusedError produces a real *pgconn.ConnectError by dialing a
// closed loopback port (pgconn.ConnectError's err field is unexported, so it
// cannot be constructed literally).
func connectRefusedError(t *testing.T) error {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err := pgconn.Connect(ctx, "postgres://user:pass@127.0.0.1:1/db")
	if err == nil {
		t.Fatal("expected connection to closed port to fail")
	}
	return err
}

func TestIsTransientConnectionError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"53300 too_many_connections", &pgconn.PgError{Code: "53300"}, true},
		{"57P03 cannot_connect_now", &pgconn.PgError{Code: "57P03"}, true},
		{"08001 connect failure", &pgconn.PgError{Code: "08001"}, true},
		{"08006 connection failure", &pgconn.PgError{Code: "08006"}, true},
		{"wrapped 53300", fmt.Errorf("acquire: %w", &pgconn.PgError{Code: "53300"}), true},
		{"net dial op error", &net.OpError{Op: "dial", Net: "tcp", Err: syscall.ECONNREFUSED}, true},
		{"wrapped ECONNREFUSED", fmt.Errorf("x: %w", syscall.ECONNREFUSED), true},
		{"wrapped ECONNRESET", fmt.Errorf("x: %w", syscall.ECONNRESET), true},
		{"23505 unique violation", &pgconn.PgError{Code: "23505"}, false},
		{"40001 serialization failure", &pgconn.PgError{Code: "40001"}, false},
		{"57014 statement timeout", &pgconn.PgError{Code: "57014"}, false},
		{"28P01 auth failure", &pgconn.PgError{Code: "28P01"}, false},
		{"bare context deadline", context.DeadlineExceeded, false},
		{"gorm record not found", gorm.ErrRecordNotFound, false},
		{"generic error", errors.New("boom"), false},
		{"net read op error", &net.OpError{Op: "read", Net: "tcp", Err: syscall.ECONNRESET}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsTransientConnectionError(tc.err); got != tc.want {
				t.Fatalf("IsTransientConnectionError(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

func TestIsTransientConnectionErrorConnectError(t *testing.T) {
	err := connectRefusedError(t)
	var ce *pgconn.ConnectError
	if !errors.As(err, &ce) {
		t.Skipf("dial to closed port did not yield *pgconn.ConnectError: %v", err)
	}
	if !IsTransientConnectionError(err) {
		t.Fatalf("real ConnectError not classified transient: %v", err)
	}
}

func TestTransientSQLState(t *testing.T) {
	if got := TransientSQLState(&pgconn.PgError{Code: "53300"}); got != "53300" {
		t.Fatalf("want 53300, got %q", got)
	}
	if got := TransientSQLState(&net.OpError{Op: "dial", Err: syscall.ECONNREFUSED}); got != "" {
		t.Fatalf("want empty for dial-shape error, got %q", got)
	}
	if got := TransientSQLState(&pgconn.PgError{Code: "23505"}); got != "" {
		t.Fatalf("want empty for non-transient SQLSTATE, got %q", got)
	}
}
