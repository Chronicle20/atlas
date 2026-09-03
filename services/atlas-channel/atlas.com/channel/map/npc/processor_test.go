package npc

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// TestInMapModelProvider_EmptyField asserts a field with no scripted NPCs
// yields an empty, non-error slice -- the overwhelmingly common case
// (every field except the five G2 seeds), which spawnScriptedNpcsForSession
// depends on being silent and cheap.
func TestInMapModelProvider_EmptyField(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(`{"data": []}`))
	}))
	defer srv.Close()
	defer SetBaseURLForTest(srv.URL)()

	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)

	f := field.NewBuilder(0, 1, 108010600).Build()

	ms, err := NewProcessor(logrus.New(), ctx).InMapModelProvider(f)()
	if err != nil {
		t.Fatalf("InMapModelProvider: %v", err)
	}
	if len(ms) != 0 {
		t.Fatalf("InMapModelProvider = %d models, want 0", len(ms))
	}
}
