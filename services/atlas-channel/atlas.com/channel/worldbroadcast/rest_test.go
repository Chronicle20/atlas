package worldbroadcast

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jtumidanski/api2go/jsonapi"
	"github.com/sirupsen/logrus"

	worldConstants "github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// TestRestModel_Unmarshal asserts the JSON:API stubs are wired so
// api2go.Unmarshal succeeds and every documented attribute round-trips into
// the wire model, mirroring monsterbook's TestCollectionRestModel_Unmarshal
// (services/atlas-channel/atlas.com/channel/monsterbook/rest_test.go).
func TestRestModel_Unmarshal(t *testing.T) {
	body := []byte(`{
		"data": {
			"type": "broadcast-queues",
			"id": "TV",
			"attributes": {
				"family": "TV",
				"activeRemainingSeconds": 7,
				"pendingCount": 2,
				"waitSeconds": 15
			}
		}
	}`)

	var rm RestModel
	if err := jsonapi.Unmarshal(body, &rm); err != nil {
		t.Fatalf("jsonapi.Unmarshal: %v", err)
	}

	if rm.GetID() != "TV" {
		t.Errorf("GetID() = %q, want %q", rm.GetID(), "TV")
	}
	if rm.Id != "TV" {
		t.Errorf("Id = %q, want %q", rm.Id, "TV")
	}
	if rm.Family != FamilyTV {
		t.Errorf("Family = %q, want %q", rm.Family, FamilyTV)
	}
	if rm.ActiveRemainingSeconds != 7 {
		t.Errorf("ActiveRemainingSeconds = %d, want 7", rm.ActiveRemainingSeconds)
	}
	if rm.PendingCount != 2 {
		t.Errorf("PendingCount = %d, want 2", rm.PendingCount)
	}
	if rm.WaitSeconds != 15 {
		t.Errorf("WaitSeconds = %d, want 15", rm.WaitSeconds)
	}
}

// TestRestModel_GetName asserts the JSON:API resource type matches the
// world-side broadcast/rest.go RestModel (task-123 Task 9).
func TestRestModel_GetName(t *testing.T) {
	var rm RestModel
	if rm.GetName() != "broadcast-queues" {
		t.Errorf("GetName() = %q, want %q", rm.GetName(), "broadcast-queues")
	}
}

// TestRestModel_SetID asserts SetID stores the family-keyed id verbatim
// (the resource id is the family string, e.g. "TV"/"AVATAR" - see
// world-side broadcast/rest.go's RestModel.Id doc comment).
func TestRestModel_SetID(t *testing.T) {
	var rm RestModel
	if err := rm.SetID(FamilyAvatar); err != nil {
		t.Fatalf("SetID: %v", err)
	}
	if rm.Id != FamilyAvatar {
		t.Errorf("Id = %q, want %q", rm.Id, FamilyAvatar)
	}
	if rm.GetID() != FamilyAvatar {
		t.Errorf("GetID() = %q, want %q", rm.GetID(), FamilyAvatar)
	}
}

func newTestTenant(t *testing.T) tenant.Model {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	return tm
}

// TestGetWaitSeconds_RoundTrip stands up an httptest server returning a
// canned broadcast-queues JSON:API document at the
// worlds/{worldId}/broadcast-queues/{family} path (matching atlas-world's
// broadcast/resource.go route from Task 9) and asserts
// NewProcessor.GetWaitSeconds decodes it and returns WaitSeconds.
func TestGetWaitSeconds_RoundTrip(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/worlds/1/broadcast-queues/TV") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(`{
			"data": {
				"type": "broadcast-queues",
				"id": "TV",
				"attributes": {
					"family": "TV",
					"activeRemainingSeconds": 0,
					"pendingCount": 3,
					"waitSeconds": 21
				}
			}
		}`))
	}))
	defer srv.Close()
	defer SetBaseURLForTest(srv.URL)()

	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	waitSeconds, err := NewProcessor(logrus.New(), ctx).GetWaitSeconds(worldConstants.Id(1), FamilyTV)
	if err != nil {
		t.Fatalf("GetWaitSeconds: %v", err)
	}
	if waitSeconds != 21 {
		t.Errorf("WaitSeconds = %d, want 21", waitSeconds)
	}
}

// TestGetWaitSeconds_NotFound asserts a transport/decode error from the
// upstream is RETURNED to the caller rather than swallowed or defaulted to
// 0 - the handler (Task 12) rejects conservatively on error (design §6
// "never consume-then-drop").
func TestGetWaitSeconds_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	defer SetBaseURLForTest(srv.URL)()

	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	_, err := NewProcessor(logrus.New(), ctx).GetWaitSeconds(worldConstants.Id(1), FamilyTV)
	if err == nil {
		t.Fatal("expected error on 404, got nil")
	}
	if !errors.Is(err, requests.ErrNotFound) {
		t.Fatalf("expected requests.ErrNotFound, got %T: %v", err, err)
	}
}
