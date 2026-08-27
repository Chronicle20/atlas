package ring

import (
	"atlas-channel/asset"
	"atlas-channel/equipment"
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	testlog "github.com/sirupsen/logrus/hooks/test"

	eqslot "atlas-channel/equipment/slot"

	slot2 "github.com/Chronicle20/atlas/libs/atlas-constants/inventory/slot"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func newRingTestContext(t *testing.T) (context.Context, uuid.UUID) {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	return tenant.WithContext(context.Background(), tm), tm.Id()
}

func mustCashAsset(t *testing.T, cashId int64) asset.Model {
	t.Helper()
	return asset.NewBuilderWithId(1, uuid.New(), 1112001).SetCashId(cashId).MustBuild()
}

// positionOf looks up the real slot position for a slot type name from
// libs/atlas-constants/inventory/slot/constants.go, so tests exercise the
// same positions the production slot table defines.
func positionOf(t *testing.T, slotType string) slot2.Position {
	t.Helper()
	for _, s := range slot2.Slots {
		if string(s.Type) == slotType {
			return s.Position
		}
	}
	t.Fatalf("unknown slot type %q", slotType)
	return 0
}

// equipCash sets slotType's CashEquipable to an asset carrying cashId.
func equipCash(t *testing.T, eq equipment.Model, slotType string, cashId int64) {
	t.Helper()
	a := mustCashAsset(t, cashId)
	eq.Set(slot2.Type(slotType), eqslot.Model{Position: positionOf(t, slotType), CashEquipable: &a})
}

// equipNonCash sets slotType's Equipable (the non-cash sub-slot) to an
// asset carrying cashId -- used only to prove a cash ring never occupies
// the non-cash sub-slot.
func equipNonCash(t *testing.T, eq equipment.Model, slotType string, cashId int64) {
	t.Helper()
	a := mustCashAsset(t, cashId)
	eq.Set(slot2.Type(slotType), eqslot.Model{Position: positionOf(t, slotType), Equipable: &a})
}

func withUpstreamFn(t *testing.T, fn func(l logrus.FieldLogger, ctx context.Context, characterId uint32) ([]Model, error)) {
	t.Helper()
	prev := upstreamFn
	upstreamFn = fn
	t.Cleanup(func() { upstreamFn = prev })
}

// Fixture halves (task-10 brief): all for character 100, partner 200.
var (
	coupleActive = Model{
		characterId: 100, partnerCharacterId: 200,
		cashId: 1111, partnerCashId: 2222,
		ringType: TypeCouple, state: StateActive,
		itemTemplateId: 1112001, partnerName: "Partner",
	}
	friendshipActive = Model{
		characterId: 100, partnerCharacterId: 200,
		cashId: 3333, partnerCashId: 4444,
		ringType: TypeFriendship, state: StateActive,
		itemTemplateId: 1112800, partnerName: "Partner",
	}
	coupleBroken = Model{
		characterId: 100, partnerCharacterId: 200,
		cashId: 5555, partnerCashId: 6666,
		ringType: TypeCouple, state: StateBroken,
		itemTemplateId: 1112001, partnerName: "Partner",
	}
)

func populate(t *testing.T, p Processor, characterId uint32, halves []Model) {
	t.Helper()
	withUpstreamFn(t, func(_ logrus.FieldLogger, _ context.Context, _ uint32) ([]Model, error) {
		return halves, nil
	})
	if err := p.Populate(characterId); err != nil {
		t.Fatalf("Populate: %v", err)
	}
}

func TestGetRingSet(t *testing.T) {
	const characterId = uint32(100)

	t.Run("no halves", func(t *testing.T) {
		resetRingCache()
		t.Cleanup(resetRingCache)
		ctx, _ := newRingTestContext(t)
		p := NewProcessor(logrus.New(), ctx)

		rs := p.GetRingSet(characterId, equipment.NewModel())
		if rs.Couple != nil || rs.Friendship != nil || rs.Marriage != nil {
			t.Fatalf("GetRingSet() = %+v, want all-nil RingSet", rs)
		}
	})

	t.Run("couple equipped", func(t *testing.T) {
		resetRingCache()
		t.Cleanup(resetRingCache)
		ctx, _ := newRingTestContext(t)
		p := NewProcessor(logrus.New(), ctx)
		populate(t, p, characterId, []Model{coupleActive})

		eq := equipment.NewModel()
		equipCash(t, eq, "ring1", 1111)

		rs := p.GetRingSet(characterId, eq)
		if rs.Couple == nil {
			t.Fatal("Couple = nil, want non-nil")
		}
		if rs.Couple.OwnSN != 1111 || rs.Couple.PartnerSN != 2222 {
			t.Fatalf("Couple = %+v, want OwnSN=1111 PartnerSN=2222", rs.Couple)
		}
		if rs.Friendship != nil {
			t.Fatalf("Friendship = %+v, want nil", rs.Friendship)
		}
	})

	t.Run("friendship equipped", func(t *testing.T) {
		resetRingCache()
		t.Cleanup(resetRingCache)
		ctx, _ := newRingTestContext(t)
		p := NewProcessor(logrus.New(), ctx)
		populate(t, p, characterId, []Model{friendshipActive})

		eq := equipment.NewModel()
		equipCash(t, eq, "ring1", 3333)

		rs := p.GetRingSet(characterId, eq)
		if rs.Friendship == nil {
			t.Fatal("Friendship = nil, want non-nil")
		}
		if rs.Couple != nil {
			t.Fatalf("Couple = %+v, want nil", rs.Couple)
		}
	})

	t.Run("owned but not equipped (FR-14)", func(t *testing.T) {
		resetRingCache()
		t.Cleanup(resetRingCache)
		ctx, _ := newRingTestContext(t)
		p := NewProcessor(logrus.New(), ctx)
		populate(t, p, characterId, []Model{coupleActive})

		rs := p.GetRingSet(characterId, equipment.NewModel())
		if rs.Couple != nil || rs.Friendship != nil {
			t.Fatalf("GetRingSet() = %+v, want empty RingSet for an owned-but-unequipped ring", rs)
		}
	})

	t.Run("BROKEN discarded (FR-3)", func(t *testing.T) {
		resetRingCache()
		t.Cleanup(resetRingCache)
		ctx, _ := newRingTestContext(t)
		p := NewProcessor(logrus.New(), ctx)
		populate(t, p, characterId, []Model{coupleBroken})

		eq := equipment.NewModel()
		equipCash(t, eq, "ring1", 5555)

		rs := p.GetRingSet(characterId, eq)
		if rs.Couple != nil {
			t.Fatalf("Couple = %+v, want nil for a BROKEN half", rs.Couple)
		}
	})

	t.Run("equipped in a non-cash slot", func(t *testing.T) {
		resetRingCache()
		t.Cleanup(resetRingCache)
		ctx, _ := newRingTestContext(t)
		p := NewProcessor(logrus.New(), ctx)
		populate(t, p, characterId, []Model{coupleActive})

		eq := equipment.NewModel()
		equipNonCash(t, eq, "ring1", 1111)

		rs := p.GetRingSet(characterId, eq)
		if rs.Couple != nil {
			t.Fatalf("Couple = %+v, want nil: a cash ring never occupies the non-cash sub-slot", rs.Couple)
		}
	})

	t.Run("pet ring slot ignored", func(t *testing.T) {
		resetRingCache()
		t.Cleanup(resetRingCache)
		ctx, _ := newRingTestContext(t)
		p := NewProcessor(logrus.New(), ctx)
		populate(t, p, characterId, []Model{coupleActive})

		eq := equipment.NewModel()
		equipCash(t, eq, "petRing1", 1111)

		rs := p.GetRingSet(characterId, eq)
		if rs.Couple != nil {
			t.Fatalf("Couple = %+v, want nil: petRing1 is pet equipment, not a couple/friendship ring slot", rs.Couple)
		}
	})

	t.Run("two couple halves, lowest slot wins (FR-15)", func(t *testing.T) {
		resetRingCache()
		t.Cleanup(resetRingCache)
		ctx, _ := newRingTestContext(t)
		p := NewProcessor(logrus.New(), ctx)
		low := Model{characterId: 100, partnerCharacterId: 200, cashId: 7777, partnerCashId: 8888, ringType: TypeCouple, state: StateActive, itemTemplateId: 1112001, partnerName: "Partner"}
		high := Model{characterId: 100, partnerCharacterId: 200, cashId: 1111, partnerCashId: 2222, ringType: TypeCouple, state: StateActive, itemTemplateId: 1112001, partnerName: "Partner"}
		populate(t, p, characterId, []Model{low, high})

		eq := equipment.NewModel()
		equipCash(t, eq, "ring1", 7777) // ring1 = -12, ranks before ring2
		equipCash(t, eq, "ring2", 1111) // ring2 = -13

		rs := p.GetRingSet(characterId, eq)
		if rs.Couple == nil || rs.Couple.OwnSN != 7777 {
			t.Fatalf("Couple = %+v, want OwnSN=7777 (ring1 outranks ring2)", rs.Couple)
		}
	})

	t.Run("slot tie broken by cashId", func(t *testing.T) {
		resetRingCache()
		t.Cleanup(resetRingCache)
		ctx, _ := newRingTestContext(t)
		p := NewProcessor(logrus.New(), ctx)
		low := Model{characterId: 100, partnerCharacterId: 200, cashId: 1111, partnerCashId: 2222, ringType: TypeCouple, state: StateActive, itemTemplateId: 1112001, partnerName: "Partner"}
		high := Model{characterId: 100, partnerCharacterId: 200, cashId: 7777, partnerCashId: 8888, ringType: TypeCouple, state: StateActive, itemTemplateId: 1112001, partnerName: "Partner"}
		populate(t, p, characterId, []Model{low, high})

		// Two distinct slot keys pinned to the SAME position (-12) so the
		// selection rule's tie-break (lowest cashId) is exercised even
		// though no two real ring slots share a position.
		eq := equipment.NewModel()
		a1 := mustCashAsset(t, 1111)
		eq.Set(slot2.Type("ring1"), eqslot.Model{Position: -12, CashEquipable: &a1})
		a2 := mustCashAsset(t, 7777)
		eq.Set(slot2.Type("ringTieDefensive"), eqslot.Model{Position: -12, CashEquipable: &a2})

		rs := p.GetRingSet(characterId, eq)
		if rs.Couple == nil || rs.Couple.OwnSN != 1111 {
			t.Fatalf("Couple = %+v, want OwnSN=1111 (lower cashId wins a slot-position tie)", rs.Couple)
		}
	})

	t.Run("upstream error (FR-5)", func(t *testing.T) {
		resetRingCache()
		t.Cleanup(resetRingCache)
		ctx, _ := newRingTestContext(t)
		l, hook := testlog.NewNullLogger()
		p := NewProcessor(l, ctx)

		withUpstreamFn(t, func(_ logrus.FieldLogger, _ context.Context, _ uint32) ([]Model, error) {
			return nil, errors.New("cashshop unreachable")
		})
		if err := p.Populate(characterId); err != nil {
			t.Fatalf("Populate() = %v, want nil (fail-soft: a cashshop outage must not propagate)", err)
		}

		var warn *logrus.Entry
		for _, e := range hook.AllEntries() {
			if e.Level == logrus.WarnLevel {
				warn = e
				break
			}
		}
		if warn == nil {
			t.Fatal("expected one warn-level log entry on upstream failure, got none")
		}
		msg := warn.Message
		if !strings.Contains(msg, strconv.Itoa(int(characterId))) {
			t.Fatalf("warn log %q does not contain character id [%d]", msg, characterId)
		}

		eq := equipment.NewModel()
		equipCash(t, eq, "ring1", 1111)
		rs := p.GetRingSet(characterId, eq)
		if rs.Couple != nil || rs.Friendship != nil || rs.Marriage != nil {
			t.Fatalf("GetRingSet() = %+v after upstream failure, want empty RingSet (no panic, no error)", rs)
		}
	})
}

// TestGetRingRecords covers Task 11's history-view accessor: every ACTIVE
// half is listed regardless of equipment, BROKEN/EXPIRED halves are dropped,
// and a cache miss returns an empty RingRecords rather than issuing a REST
// call.
func TestGetRingRecords(t *testing.T) {
	const characterId = uint32(100)

	t.Run("cache miss returns empty records", func(t *testing.T) {
		resetRingCache()
		t.Cleanup(resetRingCache)
		ctx, _ := newRingTestContext(t)
		p := NewProcessor(logrus.New(), ctx)

		rr := p.GetRingRecords(characterId)

		if len(rr.Couple) != 0 || len(rr.Friend) != 0 || len(rr.Marriage) != 0 {
			t.Fatalf("GetRingRecords() = %+v on cache miss, want empty RingRecords", rr)
		}
	})

	t.Run("lists every ACTIVE half, unequipped included, broken excluded", func(t *testing.T) {
		resetRingCache()
		t.Cleanup(resetRingCache)
		ctx, _ := newRingTestContext(t)
		p := NewProcessor(logrus.New(), ctx)
		populate(t, p, characterId, []Model{coupleActive, friendshipActive, coupleBroken})

		rr := p.GetRingRecords(characterId)

		if len(rr.Couple) != 1 {
			t.Fatalf("Couple records = %+v, want exactly 1 (coupleActive; coupleBroken must be excluded)", rr.Couple)
		}
		got := rr.Couple[0]
		want := packetmodel.CoupleRecord{
			PairCharacterId:   coupleActive.PartnerCharacterId(),
			PairCharacterName: coupleActive.PartnerName(),
			OwnSN:             coupleActive.CashId(),
			PairSN:            coupleActive.PartnerCashId(),
		}
		if got != want {
			t.Errorf("Couple[0] = %+v, want %+v", got, want)
		}

		if len(rr.Friend) != 1 {
			t.Fatalf("Friend records = %+v, want exactly 1 (friendshipActive)", rr.Friend)
		}
		if rr.Friend[0].FriendItemId != friendshipActive.ItemTemplateId() {
			t.Errorf("Friend[0].FriendItemId = %d, want %d", rr.Friend[0].FriendItemId, friendshipActive.ItemTemplateId())
		}

		if len(rr.Marriage) != 0 {
			t.Errorf("Marriage records = %+v, want empty (marriage-ring acquisition is a PRD non-goal)", rr.Marriage)
		}
	})
}

// ringsDoc renders a JSON:API "rings" list document for one ring half
// belonging to characterId, carrying a cashId above 2^53 -- the carried
// finding from task 9's review: no test yet exercised the real
// jsonapi.Unmarshal wire-decode path (only Extract on a hand-built struct).
func ringsDoc(characterId uint32, cashId int64) string {
	return fmt.Sprintf(
		`{"data":[{"id":"%s","type":"rings","attributes":{"pairId":"%s","characterId":%d,"partnerCharacterId":200,"assetId":1,"itemTemplateId":1112001,"ringType":"COUPLE","state":"ACTIVE","cashId":%d,"partnerCashId":2222,"partnerName":"Partner"}}],"meta":{"total":1,"page":{"number":1,"size":250,"last":1}}}`,
		uuid.New(), uuid.New(), characterId, cashId,
	)
}

// TestPopulateDecodesRealJSONAPIDocument proves Populate's fetch goes
// through the real jsonapi.Unmarshal wire-decode path (requests.
// DrainProvider -> Extract), not just Extract called directly on a
// hand-built RestModel, and that a cashId above 2^53 (9007199254740993)
// survives the round trip without float64 precision loss.
func TestPopulateDecodesRealJSONAPIDocument(t *testing.T) {
	resetRingCache()
	t.Cleanup(resetRingCache)

	const characterId = uint32(100)
	const largeCashId = int64(9007199254740993)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = fmt.Fprint(w, ringsDoc(characterId, largeCashId))
	}))
	t.Cleanup(srv.Close)
	t.Setenv("CASHSHOP_SERVICE_URL", srv.URL+"/api/")

	ctx, _ := newRingTestContext(t)
	p := NewProcessor(logrus.New(), ctx)

	if err := p.Populate(characterId); err != nil {
		t.Fatalf("Populate: %v", err)
	}

	eq := equipment.NewModel()
	equipCash(t, eq, "ring1", largeCashId)

	rs := p.GetRingSet(characterId, eq)
	if rs.Couple == nil {
		t.Fatal("Couple = nil after populating from a real JSON:API document, want non-nil")
	}
	if rs.Couple.OwnSN != largeCashId {
		t.Fatalf("Couple.OwnSN = %d, want %d (cashId must survive the wire round trip without float64 precision loss)", rs.Couple.OwnSN, largeCashId)
	}
}

// TestRingCachePopulatedOnCharacterLoad proves Populate's idempotency
// guard (task-269 task 12): the character-load path may call Populate once
// per load without worrying about a duplicate delivery re-fetching -- a
// second Populate call for the same character while its cache entry is
// still present must not issue a second GET /rings.
func TestRingCachePopulatedOnCharacterLoad(t *testing.T) {
	resetRingCache()
	t.Cleanup(resetRingCache)

	const characterId = uint32(100)
	calls := 0
	ctx, _ := newRingTestContext(t)
	p := NewProcessor(logrus.New(), ctx)

	withUpstreamFn(t, func(_ logrus.FieldLogger, _ context.Context, _ uint32) ([]Model, error) {
		calls++
		return []Model{coupleActive}, nil
	})

	if err := p.Populate(characterId); err != nil {
		t.Fatalf("Populate (first load): %v", err)
	}
	if calls != 1 {
		t.Fatalf("upstream calls after first Populate = %d, want 1", calls)
	}

	// Second load in the same presence -- must not re-fetch.
	if err := p.Populate(characterId); err != nil {
		t.Fatalf("Populate (second load): %v", err)
	}
	if calls != 1 {
		t.Fatalf("upstream calls after second Populate = %d, want 1 (no re-fetch while still cached)", calls)
	}

	eq := equipment.NewModel()
	equipCash(t, eq, "ring1", coupleActive.CashId())
	rs := p.GetRingSet(characterId, eq)
	if rs.Couple == nil {
		t.Fatal("Couple = nil after two Populate calls, want the cached half")
	}
}

// TestProcessorInvalidate proves Processor.Invalidate threads the tenant id
// from its own context through to the underlying cache -- carried from Task
// 10's review (non-blocking): only ringCache.invalidate was tested before,
// leaving a mis-threaded tenant id in Processor.Invalidate uncaught even
// though this task's whole invalidation path (RING_PURCHASED) runs through
// it.
func TestProcessorInvalidate(t *testing.T) {
	resetRingCache()
	t.Cleanup(resetRingCache)

	const characterId = uint32(100)
	ctx, tid := newRingTestContext(t)
	p := NewProcessor(logrus.New(), ctx)
	populate(t, p, characterId, []Model{coupleActive})

	// Sanity: the entry actually landed under this test's own tenant.
	if _, ok := getRingCache().lookup(tid, characterId); !ok {
		t.Fatalf("setup: expected character [%d] cached under tenant [%s]", characterId, tid)
	}

	p.Invalidate(characterId)

	if _, ok := getRingCache().lookup(tid, characterId); ok {
		t.Fatalf("lookup(tid, %d) = true after Processor.Invalidate, want false", characterId)
	}
}

// TestPopulateFailsSoftOnCashshopOutage proves PRD FR-5: a cashshop outage
// (here, a closed httptest.Server) degrades to an empty RingSet -- Populate
// never propagates the failure to break character spawn. It also asserts the
// degradation is observed (DOM-28): a silent empty ring cache would give no
// signal that a cashshop outage just happened.
func TestPopulateFailsSoftOnCashshopOutage(t *testing.T) {
	resetRingCache()
	t.Cleanup(resetRingCache)

	const characterId = uint32(100)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	srv.Close() // closed before any request reaches it -- connection refused
	t.Setenv("CASHSHOP_SERVICE_URL", srv.URL+"/api/")

	ctx, _ := newRingTestContext(t)
	logger, hook := testlog.NewNullLogger()
	const component = "channel.ring.populate"
	before := degradedTotalValue(t, component)
	p := NewProcessor(logger, ctx)

	if err := p.Populate(characterId); err != nil {
		t.Fatalf("Populate() = %v, want nil (fail-soft on a cashshop outage)", err)
	}

	after := degradedTotalValue(t, component)
	if after-before != 1 {
		t.Fatalf("atlas_enrichment_degraded_total{component=%q} delta = %v, want 1", component, after-before)
	}
	entry := hook.LastEntry()
	if entry == nil || entry.Level != logrus.WarnLevel {
		t.Fatalf("expected a Warn log entry for the degraded populate, got %+v", entry)
	}

	eq := equipment.NewModel()
	equipCash(t, eq, "ring1", 1111)
	rs := p.GetRingSet(characterId, eq)
	if rs.Couple != nil || rs.Friendship != nil || rs.Marriage != nil {
		t.Fatalf("GetRingSet() = %+v after a cashshop outage, want empty RingSet", rs)
	}
}

// degradedTotalValue reads the current value of
// atlas_enrichment_degraded_total{component=component} off the default
// registry. degrade.Observe registers its counter via promauto against the
// default registerer and does not export it, so this reads the same way an
// external scrape would rather than reaching into the degrade package.
func degradedTotalValue(t *testing.T, component string) float64 {
	t.Helper()
	families, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatalf("gather metrics: %v", err)
	}
	for _, mf := range families {
		if mf.GetName() != "atlas_enrichment_degraded_total" {
			continue
		}
		for _, m := range mf.GetMetric() {
			for _, lp := range m.GetLabel() {
				if lp.GetName() == "component" && lp.GetValue() == component {
					return m.GetCounter().GetValue()
				}
			}
		}
	}
	return 0
}
