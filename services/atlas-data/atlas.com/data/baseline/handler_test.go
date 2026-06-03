package baseline

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"atlas-data/rest"
	minio "atlas-data/storage/minio"

	"github.com/Chronicle20/atlas/libs/atlas-rest/server"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// nonNilSentinelClient returns a non-nil *minio.Client used to bypass the
// nil-mc 503 gate without touching network. The handler MUST NOT dereference
// it before the operator gate fires.
func nonNilSentinelClient() *minio.Client { return &minio.Client{} }

// invokeInner is a small helper that drives the typed inner handler directly,
// bypassing the RegisterInputHandler/JSON:API decode path. It lets us assert
// the operator gate and nil-mc gate in isolation.
func newDeps() (rest.HandlerDependency, rest.HandlerContext) {
	return server.NewHandlerDependency(logrus.New(), context.Background()),
		server.NewHandlerContext(nil)
}

func TestPublishNilMcReturns503(t *testing.T) {
	d, c := newDeps()
	h := publishInner(nil, nil, logrus.New())(&d, &c, PublishInputModel{Region: "GMS", MajorVersion: 83, MinorVersion: 1})
	req := httptest.NewRequest(http.MethodPost, "/api/data/baseline/publish", nil)
	rr := httptest.NewRecorder()
	h(rr, req)
	// mc is nil -> 503, before the operator check.
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rr.Code)
	}
}

func TestRestoreNilMcReturns503(t *testing.T) {
	d, c := newDeps()
	h := restoreInner(nil, nil, logrus.New())(&d, &c, RestoreInputModel{})
	req := httptest.NewRequest(http.MethodPost, "/api/data/baseline/restore", nil)
	rr := httptest.NewRecorder()
	h(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", rr.Code)
	}
}

// TestPublishRefusesNonOperator exercises the operator gate. Because we can't
// construct a real *minio.Client without env, we use a sentinel non-nil value
// to bypass the 503 gate; the request lacks X-Atlas-Operator: 1 so the handler
// must short-circuit with 403 BEFORE touching the (sentinel) client.
//
// publishInner only dereferences `mc` inside the Publisher constructor, which
// is reached only after the operator gate, so a non-nil unused pointer is safe.
func TestPublishRefusesNonOperator(t *testing.T) {
	d, c := newDeps()
	mc := nonNilSentinelClient()
	h := publishInner(nil, mc, logrus.New())(&d, &c, PublishInputModel{Region: "GMS", MajorVersion: 83, MinorVersion: 1})
	req := httptest.NewRequest(http.MethodPost, "/api/data/baseline/publish", nil)
	rr := httptest.NewRecorder()
	h(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rr.Code)
	}
}

func TestRestoreRefusesNonOperator(t *testing.T) {
	d, c := newDeps()
	mc := nonNilSentinelClient()
	h := restoreInner(nil, mc, logrus.New())(&d, &c, RestoreInputModel{Region: "GMS", MajorVersion: 83, MinorVersion: 1, TenantID: uuid.New()})
	req := httptest.NewRequest(http.MethodPost, "/api/data/baseline/restore", nil)
	rr := httptest.NewRecorder()
	h(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rr.Code)
	}
}

// TestPublishInputModelJsonApiIdentity asserts the JSON:API identity surface so
// RegisterInputHandler[PublishInputModel] can decode an inbound document.
func TestPublishInputModelJsonApiIdentity(t *testing.T) {
	var m PublishInputModel
	if m.GetName() != "baselinePublishes" {
		t.Fatalf("GetName = %s", m.GetName())
	}
	if err := m.SetID("ignored"); err != nil {
		t.Fatalf("SetID: %v", err)
	}
	if m.GetID() != "ignored" {
		t.Fatalf("GetID = %s", m.GetID())
	}
}

func TestRestoreInputModelJsonApiIdentity(t *testing.T) {
	var m RestoreInputModel
	if m.GetName() != "baselineRestores" {
		t.Fatalf("GetName = %s", m.GetName())
	}
	if err := m.SetID("ignored"); err != nil {
		t.Fatalf("SetID: %v", err)
	}
	if m.GetID() != "ignored" {
		t.Fatalf("GetID = %s", m.GetID())
	}
}

func TestPublishOutputModelIdShape(t *testing.T) {
	if got := PublishOutputId("GMS", 83, 1); got != "GMS/83.1" {
		t.Fatalf("PublishOutputId = %s", got)
	}
}
