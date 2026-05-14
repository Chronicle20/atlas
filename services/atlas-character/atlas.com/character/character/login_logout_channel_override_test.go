package character_test

import (
	"atlas-character/character"
	"atlas-character/kafka/message"
	character2 "atlas-character/kafka/message/character"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
)

// locationStub responds at GET /characters/{id}/location with a JSON:API
// payload using the supplied stale channelId. The point of the override
// fix is that this stale channelId must NOT leak into emitted LOGIN/LOGOUT
// events.
func locationStub(t *testing.T, staleChannelId channel.Id, mapId _map.Id, instance uuid.UUID) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		if !strings.Contains(r.URL.Path, "/location") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		// Path is .../characters/{id}/location — pluck the id out for echo.
		_, _ = fmt.Fprintf(w, `{"data":{"type":"character-locations","id":"1","attributes":{"worldId":0,"channelId":%d,"mapId":%d,"instance":"%s"}}}`,
			staleChannelId, mapId, instance.String())
	}))
	return srv
}

// TestLogin_OverridesStaleChannelId verifies that when location.GetField
// returns a stored channelId from a prior session, Login emits a LOGIN
// event carrying the *current* connection's channelId instead. Without
// this, MAP_STATUS / MONSTER_STATUS events route to the wrong channel and
// MonsterControl packets never reach the client.
func TestLogin_OverridesStaleChannelId(t *testing.T) {
	tctx := tenant.WithContext(context.Background(), testTenant())
	db := testDatabase(t)

	const staleChannel = channel.Id(0) // what's stored in character_locations
	const liveChannel = channel.Id(1)  // the channel the user is logging in to
	const stubMapId = _map.Id(50000)
	stubInstance := uuid.New()

	srv := locationStub(t, staleChannel, stubMapId, stubInstance)
	defer srv.Close()
	t.Setenv("MAPS_SERVICE_URL", srv.URL+"/")

	// Seed a character so processor.ByIdProvider resolves.
	input := character.NewModelBuilder().SetAccountId(1000).SetWorldId(0).SetName("LoginCh").SetLevel(1).Build()
	processor := character.NewProcessor(testLogger(), tctx, db)
	created, err := processor.Create(message.NewBuffer())(uuid.New(), input, _map.Id(0))
	if err != nil {
		t.Fatalf("Failed to seed character: %v", err)
	}

	transactionId := uuid.New()
	mb := message.NewBuffer()
	if err := processor.Login(mb)(transactionId, created.Id(), channel.NewModel(world.Id(0), liveChannel)); err != nil {
		t.Fatalf("Login emitted error: %v", err)
	}

	statusMessages, ok := mb.GetAll()[character2.EnvEventTopicCharacterStatus]
	if !ok || len(statusMessages) != 1 {
		t.Fatalf("expected exactly 1 character status message, got %d", len(statusMessages))
	}

	var event character2.StatusEvent[character2.StatusEventLoginBody]
	if err := json.Unmarshal(statusMessages[0].Value, &event); err != nil {
		t.Fatalf("failed to unmarshal LOGIN event: %v", err)
	}

	if event.Type != character2.StatusEventTypeLogin {
		t.Errorf("expected Type %s, got %s", character2.StatusEventTypeLogin, event.Type)
	}
	if event.Body.ChannelId != liveChannel {
		t.Errorf("LOGIN.Body.ChannelId: stale=%d live=%d, got %d (override did not fire)",
			staleChannel, liveChannel, event.Body.ChannelId)
	}
	if event.Body.MapId != stubMapId {
		t.Errorf("LOGIN.Body.MapId: expected %d (preserved from storage), got %d", stubMapId, event.Body.MapId)
	}
	if event.Body.Instance != stubInstance {
		t.Errorf("LOGIN.Body.Instance: expected %s (preserved from storage), got %s", stubInstance, event.Body.Instance)
	}
}

// TestLogout_OverridesStaleChannelId is the symmetric guard for Logout —
// downstream services rely on the channelId in LOGOUT events to release
// per-channel state on the right channel.
func TestLogout_OverridesStaleChannelId(t *testing.T) {
	tctx := tenant.WithContext(context.Background(), testTenant())
	db := testDatabase(t)

	const staleChannel = channel.Id(0)
	const liveChannel = channel.Id(2) // arbitrary live channel ≠ stored
	const stubMapId = _map.Id(60000)
	stubInstance := uuid.New()

	srv := locationStub(t, staleChannel, stubMapId, stubInstance)
	defer srv.Close()
	t.Setenv("MAPS_SERVICE_URL", srv.URL+"/")

	input := character.NewModelBuilder().SetAccountId(1001).SetWorldId(0).SetName("LogoutCh").SetLevel(1).Build()
	processor := character.NewProcessor(testLogger(), tctx, db)
	created, err := processor.Create(message.NewBuffer())(uuid.New(), input, _map.Id(0))
	if err != nil {
		t.Fatalf("Failed to seed character: %v", err)
	}

	transactionId := uuid.New()
	mb := message.NewBuffer()
	if err := processor.Logout(mb)(transactionId, created.Id(), channel.NewModel(world.Id(0), liveChannel)); err != nil {
		t.Fatalf("Logout emitted error: %v", err)
	}

	statusMessages, ok := mb.GetAll()[character2.EnvEventTopicCharacterStatus]
	if !ok || len(statusMessages) != 1 {
		t.Fatalf("expected exactly 1 character status message, got %d", len(statusMessages))
	}

	var event character2.StatusEvent[character2.StatusEventLogoutBody]
	if err := json.Unmarshal(statusMessages[0].Value, &event); err != nil {
		t.Fatalf("failed to unmarshal LOGOUT event: %v", err)
	}

	if event.Type != character2.StatusEventTypeLogout {
		t.Errorf("expected Type %s, got %s", character2.StatusEventTypeLogout, event.Type)
	}
	if event.Body.ChannelId != liveChannel {
		t.Errorf("LOGOUT.Body.ChannelId: stale=%d live=%d, got %d (override did not fire)",
			staleChannel, liveChannel, event.Body.ChannelId)
	}
	if event.Body.MapId != stubMapId {
		t.Errorf("LOGOUT.Body.MapId: expected %d (preserved from storage), got %d", stubMapId, event.Body.MapId)
	}
	if event.Body.Instance != stubInstance {
		t.Errorf("LOGOUT.Body.Instance: expected %s (preserved from storage), got %s", stubInstance, event.Body.Instance)
	}
}
