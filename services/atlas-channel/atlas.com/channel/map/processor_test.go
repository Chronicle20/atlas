package _map_test

import (
	_map "atlas-channel/map"
	"atlas-channel/session"
	"atlas-channel/test"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	mapid "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

func mapTestSetup() (*logrus.Logger, func()) {
	logger, _ := logtest.NewNullLogger()
	cleanup := func() {
		session.ClearRegistryForTenant(test.DefaultTenantId)
	}
	return logger, cleanup
}

// addFieldSession registers a session in the default tenant's registry with the
// given character id and field, using only public API. Mirrors the helper in
// session/processor_test.go (test packages cannot share unexported helpers).
func addFieldSession(t *testing.T, p session.Processor, characterId uint32, f field.Model) uuid.UUID {
	t.Helper()
	sessionId := uuid.New()
	ten := test.CreateDefaultMockTenant()
	s := session.NewSession(sessionId, ten, 0, nil)
	session.AddSessionToRegistry(ten.Id(), s)
	if characterId != 0 {
		p.SetCharacterId(sessionId, characterId)
	}
	p.SetField(sessionId, f)
	return sessionId
}

func idSet(ids []uint32) map[uint32]bool {
	r := make(map[uint32]bool)
	for _, id := range ids {
		r[id] = true
	}
	return r
}

// Regression proof that recipient resolution no longer performs REST: no MAPS
// service URL is configured in the test environment, so any HTTP attempt errors.
func TestGetCharacterIdsInMap_LocalResolutionNoHTTP(t *testing.T) {
	logger, cleanup := mapTestSetup()
	defer cleanup()
	ctx := test.CreateTestContext()
	sp := session.NewProcessor(logger, ctx)
	p := _map.NewProcessor(logger, ctx)

	f := field.NewBuilder(0, 0, mapid.Id(100000000)).Build()
	addFieldSession(t, sp, 100, f)
	addFieldSession(t, sp, 200, f)

	ids, err := p.GetCharacterIdsInMap(f)
	if err != nil {
		t.Fatalf("GetCharacterIdsInMap() unexpected error (REST still in the path?): %v", err)
	}
	set := idSet(ids)
	if len(ids) != 2 || !set[100] || !set[200] {
		t.Errorf("GetCharacterIdsInMap() = %v, want exactly {100, 200}", ids)
	}
}

func TestCharacterIdsInMapModelProvider_DedupsCharacterIds(t *testing.T) {
	logger, cleanup := mapTestSetup()
	defer cleanup()
	ctx := test.CreateTestContext()
	sp := session.NewProcessor(logger, ctx)
	p := _map.NewProcessor(logger, ctx)

	f := field.NewBuilder(0, 0, mapid.Id(100000000)).Build()
	// Stale socket + reconnect: two registry sessions carrying the same character id.
	addFieldSession(t, sp, 100, f)
	addFieldSession(t, sp, 100, f)

	ids, err := p.CharacterIdsInMapModelProvider(f)()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ids) != 1 || ids[0] != 100 {
		t.Errorf("CharacterIdsInMapModelProvider() = %v, want exactly [100]", ids)
	}
}

func TestOtherCharacterIdsInMapModelProvider_ExcludesReference(t *testing.T) {
	logger, cleanup := mapTestSetup()
	defer cleanup()
	ctx := test.CreateTestContext()
	sp := session.NewProcessor(logger, ctx)
	p := _map.NewProcessor(logger, ctx)

	f := field.NewBuilder(0, 0, mapid.Id(100000000)).Build()
	addFieldSession(t, sp, 100, f)
	addFieldSession(t, sp, 200, f)

	ids, err := p.OtherCharacterIdsInMapModelProvider(f, 100)()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ids) != 1 || ids[0] != 200 {
		t.Errorf("OtherCharacterIdsInMapModelProvider(f, 100) = %v, want exactly [200]", ids)
	}
}

func TestCharacterIdsInMapAllInstancesModelProvider_UnionsInstances(t *testing.T) {
	logger, cleanup := mapTestSetup()
	defer cleanup()
	ctx := test.CreateTestContext()
	sp := session.NewProcessor(logger, ctx)
	p := _map.NewProcessor(logger, ctx)

	fNil := field.NewBuilder(0, 0, mapid.Id(100000000)).Build()
	fInst := field.NewBuilder(0, 0, mapid.Id(100000000)).SetInstance(uuid.New()).Build()
	addFieldSession(t, sp, 100, fNil)
	addFieldSession(t, sp, 200, fInst)

	ids, err := p.CharacterIdsInMapAllInstancesModelProvider(0, 0, mapid.Id(100000000))()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	set := idSet(ids)
	if len(ids) != 2 || !set[100] || !set[200] {
		t.Errorf("CharacterIdsInMapAllInstancesModelProvider() = %v, want exactly {100, 200}", ids)
	}
}

// FR-4.2: a session warped from map A to map B stops receiving A-broadcasts and
// starts receiving B-broadcasts, with no state in which it is in both or neither.
func TestTransition_WarpMovesRecipientSetAtomically(t *testing.T) {
	logger, cleanup := mapTestSetup()
	defer cleanup()
	ctx := test.CreateTestContext()
	sp := session.NewProcessor(logger, ctx)
	p := _map.NewProcessor(logger, ctx)

	fA := field.NewBuilder(0, 0, mapid.Id(100000000)).Build()
	fB := field.NewBuilder(0, 0, mapid.Id(200000000)).Build()
	addFieldSession(t, sp, 100, fA) // stays in A
	bId := addFieldSession(t, sp, 200, fA)

	// Before the warp: both in A, none in B.
	idsA, err := p.GetCharacterIdsInMap(fA)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	setA := idSet(idsA)
	if len(idsA) != 2 || !setA[100] || !setA[200] {
		t.Fatalf("pre-warp map A = %v, want exactly {100, 200}", idsA)
	}
	idsB, err := p.GetCharacterIdsInMap(fB)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(idsB) != 0 {
		t.Fatalf("pre-warp map B = %v, want empty", idsB)
	}

	// Warp B's session — same call the MAP_CHANGED consumer makes
	// (kafka/consumer/character/consumer.go, SetField before dependent broadcasts).
	sp.SetField(bId, fB)

	idsA, err = p.GetCharacterIdsInMap(fA)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(idsA) != 1 || idsA[0] != 100 {
		t.Errorf("post-warp map A = %v, want exactly [100]", idsA)
	}
	idsB, err = p.GetCharacterIdsInMap(fB)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(idsB) != 1 || idsB[0] != 200 {
		t.Errorf("post-warp map B = %v, want exactly [200]", idsB)
	}
}
