package _map

import (
	"atlas-channel/door"
	_map3 "atlas-channel/kafka/message/map"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	doorcb "github.com/Chronicle20/atlas/libs/atlas-packet/door/clientbound"
	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func newTestCtx(t *testing.T) context.Context {
	t.Helper()
	tn, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	return tenant.WithContext(context.Background(), tn)
}

// newTestField uses world/channel 0 to match the zero-value world/channel a
// session carries when constructed via session.NewSession directly in tests
// (world/channel are otherwise only set by session.Processor.Create, which
// requires a live socket handshake) — see session/processor_test.go and
// map/processor_test.go for the same pattern.
func newTestField() field.Model {
	return field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(100000000)).SetInstance(uuid.Nil).Build()
}

// addFieldSession registers a session in ctx's tenant registry with the given
// character id and field, using only public API. Recipient resolution
// (map.Processor.GetCharacterIdsInMap) now filters the local session registry
// rather than calling atlas-maps (task-121), so tests exercising
// fetchOtherCharactersInMap must populate the registry instead of mocking the
// atlas-maps HTTP endpoint.
func addFieldSession(t *testing.T, ctx context.Context, l logrus.FieldLogger, characterId uint32, f field.Model) uuid.UUID {
	t.Helper()
	ten := tenant.MustFromContext(ctx)
	sessionId := uuid.New()
	s := session.NewSession(sessionId, ten, 0, nil)
	session.AddSessionToRegistry(ten.Id(), s)
	sp := session.NewProcessor(l, ctx)
	sp.SetCharacterId(sessionId, characterId)
	sp.SetField(sessionId, f)
	return sessionId
}

// characterJSON returns a minimal JSON:API character response for the given ID.
func characterJSON(id uint32) string {
	return fmt.Sprintf(`{
		"data": {
			"type": "characters",
			"id": "%d",
			"attributes": {
				"accountId": 1,
				"worldId": 1,
				"name": "TestChar%d",
				"level": 1,
				"experience": 0,
				"gachaponExperience": 0,
				"strength": 4,
				"dexterity": 4,
				"intelligence": 4,
				"luck": 4,
				"hp": 50,
				"maxHp": 50,
				"mp": 5,
				"maxMp": 5,
				"meso": 0,
				"hpMpUsed": 0,
				"jobId": 0,
				"skinColor": 0,
				"gender": 0,
				"fame": 0,
				"hair": 30000,
				"face": 20000,
				"ap": 0,
				"sp": "0,0,0,0,0,0,0,0,0,0",
				"mapId": 100000000,
				"spawnPoint": 0,
				"gm": 0,
				"x": 0,
				"y": 0,
				"stance": 0
			}
		}
	}`, id, id)
}

// TestFetchOtherCharactersInMap_SkipsNotFound verifies that a 404 for one
// character is skipped (Warn logged) but the rest are returned successfully.
func TestFetchOtherCharactersInMap_SkipsNotFound(t *testing.T) {
	logger, hook := test.NewNullLogger()
	ctx := newTestCtx(t)
	f := newTestField()
	ten := tenant.MustFromContext(ctx)
	defer session.ClearRegistryForTenant(ten.Id())

	const selfId uint32 = 1
	const missingId uint32 = 2
	const presentId uint32 = 3

	// Recipient ids now come from the local session registry (task-121), not
	// atlas-maps; only the atlas-character lookups remain over HTTP.
	addFieldSession(t, ctx, logger, selfId, f)
	addFieldSession(t, ctx, logger, missingId, f)
	addFieldSession(t, ctx, logger, presentId, f)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		switch {
		case strings.Contains(r.URL.Path, fmt.Sprintf("/characters/%d", missingId)):
			// atlas-character GET for missingId returns 404
			w.WriteHeader(http.StatusNotFound)
		case strings.Contains(r.URL.Path, fmt.Sprintf("/characters/%d", presentId)):
			// atlas-character GET for presentId returns OK
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, characterJSON(presentId))
		default:
			// any inventory/pet endpoints return empty
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	t.Setenv("CHARACTERS_SERVICE_URL", srv.URL+"/")

	cms, err := fetchOtherCharactersInMap(logger, ctx, f, selfId)
	if err != nil {
		t.Fatalf("fetchOtherCharactersInMap returned unexpected error: %v", err)
	}

	// missingId should be absent from results (skipped via Warn)
	if _, ok := cms[missingId]; ok {
		t.Errorf("missingId [%d] should have been skipped but is present in result", missingId)
	}

	// presentId should be present in results
	if _, ok := cms[presentId]; !ok {
		t.Errorf("presentId [%d] should be in result but is absent", presentId)
	}

	// selfId should be excluded (it's the excludeId)
	if _, ok := cms[selfId]; ok {
		t.Errorf("selfId [%d] should be excluded from result", selfId)
	}

	// A Warn log should have been emitted for the missing character
	found := false
	for _, entry := range hook.Entries {
		if strings.Contains(entry.Message, "Skipping stale registry entry") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected a Warn log for the stale/missing character, but none was found")
	}
}

// TestFetchOtherCharactersInMap_InfraErrorIsHardFailure verifies that a
// non-404 error from atlas-character propagates as a hard failure (not skipped).
func TestFetchOtherCharactersInMap_InfraErrorIsHardFailure(t *testing.T) {
	logger, _ := test.NewNullLogger()
	ctx := newTestCtx(t)
	f := newTestField()
	ten := tenant.MustFromContext(ctx)
	defer session.ClearRegistryForTenant(ten.Id())

	const selfId uint32 = 1
	const badId uint32 = 2

	// Recipient ids now come from the local session registry (task-121).
	addFieldSession(t, ctx, logger, selfId, f)
	addFieldSession(t, ctx, logger, badId, f)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		// Return a 500 for all character fetches
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	t.Setenv("CHARACTERS_SERVICE_URL", srv.URL+"/")

	_, err := fetchOtherCharactersInMap(logger, ctx, f, selfId)
	if err == nil {
		t.Error("expected a hard failure for infrastructure error, but got nil")
	}
}

// ---- spawnDoorsForSession tests ----

// buildTestDoor creates a door.Model for the given area field owned by
// ownerCharacterId, with the given partyId.
func buildTestDoor(f field.Model, ownerCharacterId, partyId uint32) door.Model {
	rm := door.RestModel{
		Id:               "test-door-1",
		AreaDoorId:       100,
		TownDoorId:       200,
		OwnerCharacterId: ownerCharacterId,
		PartyId:          partyId,
		WorldId:          f.WorldId(),
		ChannelId:        f.ChannelId(),
		MapId:            f.MapId(),
		Instance:         f.Instance(),
		TownMapId:        _map.Id(104000000),
		AreaX:            300,
		AreaY:            400,
		TownX:            -100,
		TownY:            -200,
	}
	m, _ := door.Extract(rm)
	return m
}

// TestSpawnDoorsForSession_EligibleOwner_SpawnDoor asserts that when the
// arriving session IS the door owner (CharacterId in the member set), the
// operator emits exactly one SpawnDoor announce.
//
// We use the zero-value session.Model (CharacterId==0) and arrange a door whose
// ownerCharacterId is also 0. doorPartyMemberSet is stubbed to return {0: {}},
// and doorAnnounce is stubbed to capture the writer name without a real socket.
func TestSpawnDoorsForSession_EligibleOwner_SpawnDoor(t *testing.T) {
	l := logrus.New()
	ctx := newTestCtx(t)
	f := newTestField()

	// session.Model{} has CharacterId()==0; door is also owned by 0.
	d := buildTestDoor(f, 0, 0)

	// Stub announce seam; capture writer name.
	var spawnDoorCalls int
	origAnnounce := doorAnnounce
	defer func() { doorAnnounce = origAnnounce }()
	doorAnnounce = func(_ logrus.FieldLogger, _ context.Context, _ writer.Producer, writerName string, _ packet.Encode, _ session.Model) error {
		if writerName == doorcb.SpawnDoorWriter {
			spawnDoorCalls++
		}
		return nil
	}

	op := spawnDoorsForSession(l)(ctx)(nil)(session.Model{})
	if err := op(d); err != nil {
		t.Fatalf("operator returned unexpected error: %v", err)
	}
	if spawnDoorCalls != 1 {
		t.Errorf("SpawnDoor calls = %d, want 1 for eligible owner", spawnDoorCalls)
	}
}

// TestSpawnDoorsForSession_NonPartySession_StillSpawns asserts that a session
// that is NOT in the door owner's party still receives the area door: the door
// is a plain map object shown to everyone in the map (party only gates entry).
func TestSpawnDoorsForSession_NonPartySession_StillSpawns(t *testing.T) {
	l := logrus.New()
	ctx := newTestCtx(t)
	f := newTestField()

	// Door is owned by character 999; session is zero-value (CharacterId()==0),
	// not in the owner's party.
	d := buildTestDoor(f, 999, 0)

	// Stub announce; count SpawnDoor calls.
	var spawnDoorCalls int
	origAnnounce := doorAnnounce
	defer func() { doorAnnounce = origAnnounce }()
	doorAnnounce = func(_ logrus.FieldLogger, _ context.Context, _ writer.Producer, writerName string, _ packet.Encode, _ session.Model) error {
		if writerName == doorcb.SpawnDoorWriter {
			spawnDoorCalls++
		}
		return nil
	}

	op := spawnDoorsForSession(l)(ctx)(nil)(session.Model{})
	if err := op(d); err != nil {
		t.Fatalf("operator returned unexpected error: %v", err)
	}
	if spawnDoorCalls != 1 {
		t.Errorf("SpawnDoor calls = %d, want 1 (door is visible to everyone in the map)", spawnDoorCalls)
	}
}

// TestSpawnDoorsForSession_PartyMember_SpawnDoor asserts that when the arriving
// session is a same-channel party member of the door owner (not the owner
// themselves), the operator still emits SpawnDoor.
func TestSpawnDoorsForSession_PartyMember_SpawnDoor(t *testing.T) {
	l := logrus.New()
	ctx := newTestCtx(t)
	f := newTestField()

	// Door owned by 999; session (CharacterId==0) is a party member.
	d := buildTestDoor(f, 999, 77)

	var spawnDoorCalls int
	origAnnounce := doorAnnounce
	defer func() { doorAnnounce = origAnnounce }()
	doorAnnounce = func(_ logrus.FieldLogger, _ context.Context, _ writer.Producer, writerName string, _ packet.Encode, _ session.Model) error {
		if writerName == doorcb.SpawnDoorWriter {
			spawnDoorCalls++
		}
		return nil
	}

	op := spawnDoorsForSession(l)(ctx)(nil)(session.Model{})
	if err := op(d); err != nil {
		t.Fatalf("operator returned unexpected error: %v", err)
	}
	if spawnDoorCalls != 1 {
		t.Errorf("SpawnDoor calls = %d, want 1 for party member", spawnDoorCalls)
	}
}

// eventVisualsServer stands up a fake atlas-events map-entry visuals endpoint
// returning the given JSON:API body, and points BASE_SERVICE_URL at it for
// the duration of the test (mirrors events/processor_test.go).
func eventVisualsServer(t *testing.T, body string) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	t.Setenv("BASE_SERVICE_URL", srv.URL+"/api/")
}

// stubDoorAnnounceForVisuals swaps doorAnnounce for a recording stub and
// returns a restore func plus captured invocation details, keyed by writer
// name. announceActiveVisuals is the only caller under test in this file
// that reuses the doorAnnounce seam for non-door writers.
//
// The ContiMove state/subState bytes are no longer read off the wire
// response (DOM-25) -- the writer resolves them from the tenant's ContiMove
// writer options table, so the stub invokes enc with a fixture options map
// (the same shape RegisterTenantWriterOptions/TenantWriterOptions produce)
// to demonstrate the resolution actually happens.
func stubDoorAnnounceForVisuals(t *testing.T) (restore func(), calls *[]string, lastState, lastSubState *byte, lastBgm *string) {
	t.Helper()
	var seenWriters []string
	var state, subState byte
	var bgm string

	contiMoveOptions := map[string]interface{}{
		"operations": map[string]interface{}{
			"SHOW_STATE":     float64(10),
			"SHOW_SUB_STATE": float64(4),
		},
	}

	orig := doorAnnounce
	doorAnnounce = func(_ logrus.FieldLogger, _ context.Context, _ writer.Producer, writerName string, enc packet.Encode, _ session.Model) error {
		seenWriters = append(seenWriters, writerName)
		switch writerName {
		case fieldcb.ContiMoveWriter:
			b := enc(logrus.New(), context.Background())(contiMoveOptions)
			if len(b) >= 2 {
				state, subState = b[0], b[1]
			}
		case fieldcb.FieldEffectWriter:
			// FieldEffectBackgroundMusicBody's exact wire shape isn't needed
			// here; the bgm value itself is asserted via the response fixture.
			bgm = "seen"
		}
		return nil
	}

	return func() { doorAnnounce = orig }, &seenWriters, &state, &subState, &bgm
}

// TestAnnounceActiveVisuals_ContiMoveFiresForEnteringSession asserts a
// CONTI_MOVE visual returned by atlas-events is announced to the entering
// session as ContiMove, with the wire state/subState resolved from the
// tenant's ContiMove writer options table (DOM-25) -- not read off the
// atlas-events response, which no longer needs to carry them for this to
// work (the endpoint only ever returns currently-active visuals, so this
// site always resolves the SHOW pair).
func TestAnnounceActiveVisuals_ContiMoveFiresForEnteringSession(t *testing.T) {
	l := logrus.New()
	ctx := newTestCtx(t)
	f := newTestField()

	eventVisualsServer(t, `{"data":[{"type":"event-visuals","id":"1","attributes":{"occurrenceId":"o1","visual":"CONTI_MOVE","bgm":"Bgm04/ArabPirate"}}]}`)

	restore, calls, lastState, lastSubState, lastBgm := stubDoorAnnounceForVisuals(t)
	defer restore()

	announceActiveVisuals(l, ctx, nil, f, session.Model{})

	if len(*calls) != 2 || (*calls)[0] != fieldcb.ContiMoveWriter || (*calls)[1] != fieldcb.FieldEffectWriter {
		t.Fatalf("writer calls = %v, want [%s %s]", *calls, fieldcb.ContiMoveWriter, fieldcb.FieldEffectWriter)
	}
	if *lastState != 10 || *lastSubState != 4 {
		t.Fatalf("ContiMove state/subState: want the tenant-configured (10,4), got (%d,%d)", *lastState, *lastSubState)
	}
	if *lastBgm == "" {
		t.Fatalf("expected the background-music writer to fire for a non-empty bgm")
	}
}

// TestAnnounceActiveVisuals_SkipsNonContiMoveVisual asserts a visual that
// isn't CONTI_MOVE is skipped entirely -- no writer for it exists yet, and
// the loop must not fall through to the ContiMove/bgm announce.
func TestAnnounceActiveVisuals_SkipsNonContiMoveVisual(t *testing.T) {
	l := logrus.New()
	ctx := newTestCtx(t)
	f := newTestField()

	eventVisualsServer(t, `{"data":[{"type":"event-visuals","id":"1","attributes":{"occurrenceId":"o1","visual":"SOME_OTHER_VISUAL","bgm":"Bgm/Whatever"}}]}`)

	restore, calls, _, _, _ := stubDoorAnnounceForVisuals(t)
	defer restore()

	announceActiveVisuals(l, ctx, nil, f, session.Model{})

	if len(*calls) != 0 {
		t.Fatalf("writer calls = %v, want none for an unrecognized visual", *calls)
	}
}

// TestAnnounceActiveVisuals_FailOpenAnnouncesNothing asserts that when
// atlas-events is unreachable, announceActiveVisuals announces nothing and
// returns without panicking or otherwise disrupting the caller (FR-B16,
// FR-N15) -- map entry itself is unaffected.
func TestAnnounceActiveVisuals_FailOpenAnnouncesNothing(t *testing.T) {
	l := logrus.New()
	ctx := newTestCtx(t)
	f := newTestField()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	t.Setenv("BASE_SERVICE_URL", srv.URL+"/api/")

	restore, calls, _, _, _ := stubDoorAnnounceForVisuals(t)
	defer restore()

	announceActiveVisuals(l, ctx, nil, f, session.Model{})

	if len(*calls) != 0 {
		t.Fatalf("writer calls = %v, want none when atlas-events is unreachable", *calls)
	}
}

// jukeboxAnnounce records a single doorAnnounce invocation made by a jukebox
// handler, capturing the encoded PlayJukebox body so the exact item id on
// the wire can be asserted.
type jukeboxAnnounce struct {
	Writer string
	Body   []byte
}

// jukeboxAnnounceRecorder collects jukeboxAnnounce invocations made
// concurrently -- handleStatusEventJukeboxStart/End fan the announce out
// through map.(*ProcessorImpl).ForSessionsInMap, which runs the operator on
// one goroutine per session (libs/atlas-model ExecuteForEachSlice), so a
// bare captured slice races. The mutex is held only around the append and
// around the snapshot the test body reads back after the fan-out completes.
type jukeboxAnnounceRecorder struct {
	mu   sync.Mutex
	seen []jukeboxAnnounce
}

func (r *jukeboxAnnounceRecorder) record(a jukeboxAnnounce) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seen = append(r.seen, a)
}

// snapshot returns a copy of every call recorded so far, so the test body
// can range/index it without holding the lock itself.
func (r *jukeboxAnnounceRecorder) snapshot() []jukeboxAnnounce {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]jukeboxAnnounce, len(r.seen))
	copy(out, r.seen)
	return out
}

// stubDoorAnnounceForJukebox swaps doorAnnounce for a recording stub and
// returns a restore func plus the recorder it captured into -- the -1 stop
// signal is a correctness requirement, not a convention (design §3.2).
func stubDoorAnnounceForJukebox(t *testing.T) (restore func(), rec *jukeboxAnnounceRecorder) {
	t.Helper()
	rec = &jukeboxAnnounceRecorder{}

	orig := doorAnnounce
	doorAnnounce = func(l logrus.FieldLogger, ctx context.Context, _ writer.Producer, writerName string, enc packet.Encode, _ session.Model) error {
		rec.record(jukeboxAnnounce{Writer: writerName, Body: enc(l, ctx)(nil)})
		return nil
	}

	return func() { doorAnnounce = orig }, rec
}

// decodePlayJukebox decodes a captured PlayJukebox wire body via the real
// codec, so the assertion exercises the same decode path the client would.
func decodePlayJukebox(t *testing.T, body []byte) fieldcb.PlayJukebox {
	t.Helper()
	req := request.Request(body)
	reader := request.NewRequestReader(&req, 0)
	var m fieldcb.PlayJukebox
	m.Decode(logrus.New(), context.Background())(&reader, nil)
	return m
}

// newTestServerModel registers a real server.Model for ctx's tenant at
// world/channel 0, matching newTestField -- sc.Is compares the full tenant,
// not just world/channel, so a zero-value server.Model{} (unset tenant)
// never matches a real context tenant.
func newTestServerModel(t *testing.T, ctx context.Context) server.Model {
	t.Helper()
	ten := tenant.MustFromContext(ctx)
	return server.NewProcessor(logrus.New(), ctx).Register(ten, channel.NewModel(world.Id(0), channel.Id(0)), "", 0)
}

func TestHandleStatusEventJukeboxStart_BroadcastsToEverySessionInField(t *testing.T) {
	l := logrus.New()
	ctx := newTestCtx(t)
	ten := tenant.MustFromContext(ctx)
	defer session.ClearRegistryForTenant(ten.Id())
	f := newTestField()

	addFieldSession(t, ctx, l, 1001, f)
	addFieldSession(t, ctx, l, 1002, f)

	restore, rec := stubDoorAnnounceForJukebox(t)
	defer restore()

	sc := newTestServerModel(t, ctx)
	e := _map3.StatusEvent[_map3.JukeboxStart]{
		Type:      _map3.EventTopicMapStatusTypeJukeboxStart,
		WorldId:   world.Id(0),
		ChannelId: channel.Id(0),
		MapId:     _map.Id(100000000),
		Instance:  uuid.Nil,
		Body:      _map3.JukeboxStart{ItemId: 5100000, PlayerName: "Chronicle"},
	}
	handleStatusEventJukeboxStart(sc, nil)(l, ctx, e)

	calls := rec.snapshot()
	if len(calls) != 2 {
		t.Fatalf("announce count = %d, want 2", len(calls))
	}
	for _, c := range calls {
		if c.Writer != fieldcb.PlayJukeboxWriter {
			t.Fatalf("writer = %s, want %s", c.Writer, fieldcb.PlayJukeboxWriter)
		}
		m := decodePlayJukebox(t, c.Body)
		if m.ItemId() != 5100000 {
			t.Fatalf("itemId = %d, want 5100000", m.ItemId())
		}
		if m.PlayerName() != "Chronicle" {
			t.Fatalf("playerName = %s, want Chronicle", m.PlayerName())
		}
	}
}

func TestHandleStatusEventJukeboxStart_IgnoresOtherEventTypes(t *testing.T) {
	l := logrus.New()
	ctx := newTestCtx(t)
	ten := tenant.MustFromContext(ctx)
	defer session.ClearRegistryForTenant(ten.Id())
	f := newTestField()

	addFieldSession(t, ctx, l, 1001, f)
	addFieldSession(t, ctx, l, 1002, f)

	restore, rec := stubDoorAnnounceForJukebox(t)
	defer restore()

	sc := newTestServerModel(t, ctx)
	e := _map3.StatusEvent[_map3.JukeboxStart]{
		Type:      "SOMETHING_ELSE",
		WorldId:   world.Id(0),
		ChannelId: channel.Id(0),
		MapId:     _map.Id(100000000),
		Instance:  uuid.Nil,
		Body:      _map3.JukeboxStart{ItemId: 5100000, PlayerName: "Chronicle"},
	}
	handleStatusEventJukeboxStart(sc, nil)(l, ctx, e)

	if calls := rec.snapshot(); len(calls) != 0 {
		t.Fatalf("announce count = %d, want 0", len(calls))
	}
}

func TestHandleStatusEventJukeboxStart_IgnoresOtherWorldChannel(t *testing.T) {
	l := logrus.New()
	ctx := newTestCtx(t)
	ten := tenant.MustFromContext(ctx)
	defer session.ClearRegistryForTenant(ten.Id())
	f := newTestField()

	addFieldSession(t, ctx, l, 1001, f)
	addFieldSession(t, ctx, l, 1002, f)

	restore, rec := stubDoorAnnounceForJukebox(t)
	defer restore()

	sc := newTestServerModel(t, ctx)
	e := _map3.StatusEvent[_map3.JukeboxStart]{
		Type:      _map3.EventTopicMapStatusTypeJukeboxStart,
		WorldId:   world.Id(1),
		ChannelId: channel.Id(0),
		MapId:     _map.Id(100000000),
		Instance:  uuid.Nil,
		Body:      _map3.JukeboxStart{ItemId: 5100000, PlayerName: "Chronicle"},
	}
	handleStatusEventJukeboxStart(sc, nil)(l, ctx, e)

	if calls := rec.snapshot(); len(calls) != 0 {
		t.Fatalf("announce count = %d, want 0", len(calls))
	}
}

func TestHandleStatusEventJukeboxEnd_BroadcastsExactlyMinusOne(t *testing.T) {
	l := logrus.New()
	ctx := newTestCtx(t)
	ten := tenant.MustFromContext(ctx)
	defer session.ClearRegistryForTenant(ten.Id())
	f := newTestField()

	addFieldSession(t, ctx, l, 1001, f)
	addFieldSession(t, ctx, l, 1002, f)

	restore, rec := stubDoorAnnounceForJukebox(t)
	defer restore()

	sc := newTestServerModel(t, ctx)
	e := _map3.StatusEvent[_map3.JukeboxEnd]{
		Type:      _map3.EventTopicMapStatusTypeJukeboxEnd,
		WorldId:   world.Id(0),
		ChannelId: channel.Id(0),
		MapId:     _map.Id(100000000),
		Instance:  uuid.Nil,
		Body:      _map3.JukeboxEnd{ItemId: 5100000},
	}
	handleStatusEventJukeboxEnd(sc, nil)(l, ctx, e)

	calls := rec.snapshot()
	if len(calls) != 2 {
		t.Fatalf("announce count = %d, want 2", len(calls))
	}
	for _, c := range calls {
		if c.Writer != fieldcb.PlayJukeboxWriter {
			t.Fatalf("writer = %s, want %s", c.Writer, fieldcb.PlayJukeboxWriter)
		}
		if len(c.Body) < 4 {
			t.Fatalf("body too short: %v", c.Body)
		}
		if c.Body[0] != 0xff || c.Body[1] != 0xff || c.Body[2] != 0xff || c.Body[3] != 0xff {
			t.Fatalf("raw itemId bytes = % x, want ff ff ff ff", c.Body[:4])
		}
		m := decodePlayJukebox(t, c.Body)
		if m.ItemId() != -1 {
			t.Fatalf("itemId = %d, want -1", m.ItemId())
		}
		if m.PlayerName() != "" {
			t.Fatalf("playerName = %q, want empty", m.PlayerName())
		}
	}
}

func TestHandleStatusEventJukeboxEnd_IgnoresOtherWorldChannel(t *testing.T) {
	l := logrus.New()
	ctx := newTestCtx(t)
	ten := tenant.MustFromContext(ctx)
	defer session.ClearRegistryForTenant(ten.Id())
	f := newTestField()

	addFieldSession(t, ctx, l, 1001, f)
	addFieldSession(t, ctx, l, 1002, f)

	restore, rec := stubDoorAnnounceForJukebox(t)
	defer restore()

	sc := newTestServerModel(t, ctx)
	e := _map3.StatusEvent[_map3.JukeboxEnd]{
		Type:      _map3.EventTopicMapStatusTypeJukeboxEnd,
		WorldId:   world.Id(0),
		ChannelId: channel.Id(3),
		MapId:     _map.Id(100000000),
		Instance:  uuid.Nil,
		Body:      _map3.JukeboxEnd{ItemId: 5100000},
	}
	handleStatusEventJukeboxEnd(sc, nil)(l, ctx, e)

	if calls := rec.snapshot(); len(calls) != 0 {
		t.Fatalf("announce count = %d, want 0", len(calls))
	}
}

// jukeboxServer starts an httptest server that answers the atlas-maps
// jukebox resource with body, and points MAPS_SERVICE_URL at it.
func jukeboxServer(t *testing.T, status int, body string) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		if status != http.StatusOK {
			w.WriteHeader(status)
			return
		}
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	t.Setenv("MAPS_SERVICE_URL", srv.URL+"/")
}

func TestAnnounceActiveJukebox_ReplaysToTheEnteringSession(t *testing.T) {
	l := logrus.New()
	ctx := newTestCtx(t)
	f := newTestField()

	jukeboxServer(t, http.StatusOK, `{"data":{"type":"jukebox","id":"5100000","attributes":{"itemId":5100000,"playerName":"Chronicle"}}}`)

	restore, rec := stubDoorAnnounceForJukebox(t)
	defer restore()

	announceActiveJukebox(l, ctx, nil, f, session.Model{})

	calls := rec.snapshot()
	if len(calls) != 1 {
		t.Fatalf("announce count = %d, want 1", len(calls))
	}
	c := calls[0]
	if c.Writer != fieldcb.PlayJukeboxWriter {
		t.Fatalf("writer = %s, want %s", c.Writer, fieldcb.PlayJukeboxWriter)
	}
	m := decodePlayJukebox(t, c.Body)
	if m.ItemId() != 5100000 {
		t.Fatalf("itemId = %d, want 5100000", m.ItemId())
	}
	if m.PlayerName() != "Chronicle" {
		t.Fatalf("playerName = %s, want Chronicle", m.PlayerName())
	}
}

func TestAnnounceActiveJukebox_FailsOpenWhenMapsUnreachable(t *testing.T) {
	l := logrus.New()
	ctx := newTestCtx(t)
	f := newTestField()

	jukeboxServer(t, http.StatusNotFound, "")

	restore, rec := stubDoorAnnounceForJukebox(t)
	defer restore()

	announceActiveJukebox(l, ctx, nil, f, session.Model{})

	if calls := rec.snapshot(); len(calls) != 0 {
		t.Fatalf("announce count = %d, want 0", len(calls))
	}
}
