package session

import (
	"atlas-channel/ring"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	logtest "github.com/sirupsen/logrus/hooks/test"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// This file pins Task 12 fix round 1's session-destroy edge for the ring
// pair cache (PRD FR-4), mirroring position_hook_test.go's pattern: it calls
// clearRingsOnDestroy directly rather than the full Destroy, which would
// require a live Kafka broker for the logout command and DESTROYED status
// event. Unlike position, ring.NewProcessor's cache has no exported seeding
// hook other than Populate (a real REST fetch, PRD FR-5) -- so these tests
// seed the real cache via Populate against an httptest.Server standing in
// for atlas-cashshop, and read it back through GetRingRecords, so the
// assertion is on real cache state rather than on a mock having been called.

// ringHookTestTenant creates a fresh tenant per test so seeded/cleared cache
// entries cannot leak into (or race with) other tests in this package or in
// the ring package's own tests.
func ringHookTestTenant(t *testing.T) tenant.Model {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	return tm
}

// ringsDoc is a minimal single-half JSON:API document for atlas-cashshop's
// GET /rings route (ring/processor_test.go:367-372's fixture, reused here
// since it is the module's established shape for this document).
func ringsDoc(characterId uint32, cashId int64) string {
	return fmt.Sprintf(
		`{"data":[{"id":"%s","type":"rings","attributes":{"pairId":"%s","characterId":%d,"partnerCharacterId":200,"assetId":1,"itemTemplateId":1112001,"ringType":"COUPLE","state":"ACTIVE","cashId":%d,"partnerCashId":2222,"partnerName":"Partner"}}],"meta":{"total":1,"page":{"number":1,"size":250,"last":1}}}`,
		uuid.New(), uuid.New(), characterId, cashId,
	)
}

// seedRingCache populates the real ring cache for characterId under ctx's
// tenant via a real Populate call against an httptest.Server standing in for
// atlas-cashshop.
func seedRingCache(t *testing.T, ctx context.Context, characterId uint32) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = fmt.Fprint(w, ringsDoc(characterId, 1111))
	}))
	t.Cleanup(srv.Close)
	t.Setenv("CASHSHOP_SERVICE_URL", srv.URL+"/api/")

	logger, _ := logtest.NewNullLogger()
	if err := ring.NewProcessor(logger, ctx).Populate(characterId); err != nil {
		t.Fatalf("seed setup: Populate: %v", err)
	}
}

// TestClearRingsOnDestroy_NonZeroCharacter_ClearsState pins the positive
// case: a destroyed session with a real character must drop that
// character's cached ring pair halves.
func TestClearRingsOnDestroy_NonZeroCharacter_ClearsState(t *testing.T) {
	tn := ringHookTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tn)
	characterId := uint32(884301)

	seedRingCache(t, ctx, characterId)

	logger, _ := logtest.NewNullLogger()
	rp := ring.NewProcessor(logger, ctx)
	if rr := rp.GetRingRecords(characterId); len(rr.Couple) == 0 {
		t.Fatal("test setup invalid: want a seeded ring record before the hook runs, got none")
	}

	clearRingsOnDestroy(logger, ctx, characterId)

	if rr := rp.GetRingRecords(characterId); len(rr.Couple) != 0 {
		t.Errorf("after clearRingsOnDestroy: want no cached ring records, got %+v", rr)
	}
}

// TestClearRingsOnDestroy_ZeroCharacter_NoOp pins the negative case: a
// session that never reached character selection (CharacterId == 0) must
// not touch the cache at all -- in particular it must not disturb an
// unrelated character's entry.
func TestClearRingsOnDestroy_ZeroCharacter_NoOp(t *testing.T) {
	tn := ringHookTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tn)
	characterId := uint32(884302)

	seedRingCache(t, ctx, characterId)

	logger, _ := logtest.NewNullLogger()
	clearRingsOnDestroy(logger, ctx, 0)

	rp := ring.NewProcessor(logger, ctx)
	if rr := rp.GetRingRecords(characterId); len(rr.Couple) == 0 {
		t.Error("clearRingsOnDestroy(l, ctx, 0) must be a no-op: want character 884302's entry untouched, got it cleared")
	}
}
