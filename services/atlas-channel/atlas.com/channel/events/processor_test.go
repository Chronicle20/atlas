package events

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func testLogger(t *testing.T) logrus.FieldLogger {
	t.Helper()
	l := logrus.New()
	l.SetOutput(io.Discard)
	return l
}

func testCtx(t *testing.T) context.Context {
	t.Helper()
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	return tenant.WithContext(context.Background(), ten)
}

// FR-B16/FR-N15: an unreachable atlas-events costs the visual and nothing else.
// The lookup must never surface as an error that aborts map entry.
func TestActiveVisualsInMapFailsOpen(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	t.Setenv("BASE_SERVICE_URL", srv.URL+"/api/")

	_, err := NewProcessor(testLogger(t), testCtx(t)).ActiveVisualsInMap(field.NewBuilder(1, 4, 200090010).Build())
	if err == nil {
		t.Fatalf("expected the transport error to be returned so the CALLER can log and move on")
	}
}

func TestActiveVisualsInMapDecodesTheProjection(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"data":[{"type":"event-visuals","id":"1","attributes":{"occurrenceId":"o1","visual":"CONTI_MOVE","state":10,"subState":4,"bgm":"Bgm04/ArabPirate"}}]}`))
	}))
	defer srv.Close()
	t.Setenv("BASE_SERVICE_URL", srv.URL+"/api/")

	vs, err := NewProcessor(testLogger(t), testCtx(t)).ActiveVisualsInMap(field.NewBuilder(1, 4, 200090010).Build())
	if err != nil {
		t.Fatalf("ActiveVisualsInMap: %v", err)
	}
	if len(vs) != 1 || vs[0].State != 10 || vs[0].SubState != 4 || vs[0].Bgm != "Bgm04/ArabPirate" {
		t.Fatalf("decoded %+v", vs)
	}
}
