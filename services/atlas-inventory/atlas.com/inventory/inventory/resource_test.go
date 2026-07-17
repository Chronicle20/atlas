package inventory

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	server "github.com/Chronicle20/atlas/libs/atlas-rest/server"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// --- fake driver whose every query fails with a fixed error ---

type failConn struct{ err error }

func (failConn) Prepare(string) (driver.Stmt, error) { return nil, errors.New("not implemented") }
func (failConn) Close() error                        { return nil }
func (failConn) Begin() (driver.Tx, error)           { return nil, errors.New("not implemented") }
func (c failConn) QueryContext(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
	return nil, c.err
}

func (c failConn) ExecContext(context.Context, string, []driver.NamedValue) (driver.Result, error) {
	return nil, c.err
}

type failConnector struct{ err error }

func (f failConnector) Connect(context.Context) (driver.Conn, error) {
	return failConn(f), nil
}
func (f failConnector) Driver() driver.Driver { return nil }

func failingDB(t *testing.T, queryErr error) *gorm.DB {
	t.Helper()
	sqlDB := sql.OpenDB(failConnector{err: queryErr})
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	return db
}

type testSI struct{}

func (testSI) GetBaseURL() string { return "http://localhost" }
func (testSI) GetPrefix() string  { return "" }

func serveGetInventory(t *testing.T, db *gorm.DB) *httptest.ResponseRecorder {
	t.Helper()
	l := logrus.New()
	l.SetLevel(logrus.PanicLevel)
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	ctx := tenant.WithContext(context.Background(), ten)

	d := server.NewHandlerDependency(l, ctx)
	c := server.NewHandlerContext(testSI{})
	router := mux.NewRouter()
	router.HandleFunc("/characters/{characterId}/inventory", handleGetInventory(db)(&d, &c))

	req := httptest.NewRequest(http.MethodGet, "/characters/42/inventory", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	return rec
}

func TestGetInventoryTransientDBErrorIs503(t *testing.T) {
	server.RegisterTransientErrorClassifier(func(err error) bool {
		if database.IsTransientConnectionError(err) {
			database.CountTransient(err)
			return true
		}
		return false
	})
	defer server.RegisterTransientErrorClassifier(nil)

	rec := serveGetInventory(t, failingDB(t, &pgconn.PgError{Code: "53300"}))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d (body: %s)", rec.Code, rec.Body.String())
	}
	if rec.Header().Get("Retry-After") != "1" {
		t.Fatalf("expected Retry-After: 1, got %q", rec.Header().Get("Retry-After"))
	}
	if !strings.Contains(rec.Body.String(), "temporarily unavailable") {
		t.Fatalf("expected JSON:API 503 body, got: %s", rec.Body.String())
	}
}

func TestGetInventoryNonTransientDBErrorIs500(t *testing.T) {
	server.RegisterTransientErrorClassifier(database.IsTransientConnectionError)
	defer server.RegisterTransientErrorClassifier(nil)

	rec := serveGetInventory(t, failingDB(t, errors.New("real bug")))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
}
