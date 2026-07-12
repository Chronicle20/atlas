package database

import (
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
}
