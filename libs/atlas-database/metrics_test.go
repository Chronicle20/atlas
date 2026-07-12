package database

import (
	"net"
	"syscall"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestCountTransientLabelsBySQLState(t *testing.T) {
	before := testutil.ToFloat64(transientErrorsTotal.WithLabelValues("53300"))
	CountTransient(&pgconn.PgError{Code: "53300"})
	after := testutil.ToFloat64(transientErrorsTotal.WithLabelValues("53300"))
	if after-before != 1 {
		t.Fatalf("expected counter delta 1, got %v", after-before)
	}
}

func TestCountTransientDialShapeUsesEmptyLabel(t *testing.T) {
	before := testutil.ToFloat64(transientErrorsTotal.WithLabelValues(""))
	CountTransient(&pgconn.PgError{Code: "53300"}) // wrong label, should not affect ""
	beforeAfterWrong := testutil.ToFloat64(transientErrorsTotal.WithLabelValues(""))
	if beforeAfterWrong != before {
		t.Fatalf("unexpected empty-label increment")
	}

	// A dial-shape error is classified transient but carries no SQLSTATE, so it
	// must land on the "" label.
	dialErr := &net.OpError{Op: "dial", Net: "tcp", Err: syscall.ECONNREFUSED}
	CountTransient(dialErr)
	afterDial := testutil.ToFloat64(transientErrorsTotal.WithLabelValues(""))
	if afterDial-beforeAfterWrong != 1 {
		t.Fatalf("expected empty-label counter delta 1 for dial-shape error, got %v", afterDial-beforeAfterWrong)
	}
}
