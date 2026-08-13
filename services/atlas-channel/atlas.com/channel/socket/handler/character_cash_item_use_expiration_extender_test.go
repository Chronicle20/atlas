package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	statpkt "github.com/Chronicle20/atlas/libs/atlas-packet/stat/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func testExtenderTenant(t *testing.T, region string, major uint16, minor uint16) tenant.Model {
	t.Helper()
	te, err := tenant.Create(uuid.New(), region, major, minor)
	if err != nil {
		t.Fatal(err)
	}
	return te
}

func TestExpirationExtenderCashSlotItemTypeIsVersionScoped(t *testing.T) {
	// 62 at GMS >= 95, 61 below — IDA-verified: gms_v83
	// get_cashslot_item_type @0x48645B case 550 -> 61; gms_v95 @0x488C70
	// case 550 -> 62. It must never be a bare literal: 61 is the
	// otherCategory==7 megaphone arm at GMS >= 95, and 62 is classification
	// 551 below it.
	cases := []struct {
		region string
		major  uint16
		want   CashSlotItemType
	}{
		{"GMS", 72, CashSlotItemTypeExpirationExtender},
		{"GMS", 83, CashSlotItemTypeExpirationExtender},
		{"GMS", 87, CashSlotItemTypeExpirationExtender},
		{"GMS", 95, CashSlotItemTypeExpirationExtenderV95},
		{"JMS", 185, CashSlotItemTypeExpirationExtender},
	}
	for _, c := range cases {
		te := testExtenderTenant(t, c.region, c.major, 1)
		if got := expirationExtenderCashSlotItemType(te); got != c.want {
			t.Errorf("%s v%d: got %d, want %d", c.region, c.major, got, c.want)
		}
	}
}

func TestExpirationExtenderResolverAgreesWithClassifier(t *testing.T) {
	// The arm matches on the resolver, but dispatch computes the type through
	// GetCashSlotItemType. If the two ever disagree the arm is unreachable.
	for _, major := range []uint16{72, 79, 83, 84, 87, 92, 95} {
		te := testExtenderTenant(t, "GMS", major, 1)
		classified := GetCashSlotItemType(te)(5500001)
		resolved := expirationExtenderCashSlotItemType(te)
		if classified != resolved {
			t.Errorf("GMS v%d: classifier gave %d, resolver gave %d", major, classified, resolved)
		}
	}
}

func TestEvaluateExpirationExtension(t *testing.T) {
	now := time.Date(2026, 8, 13, 0, 0, 0, 0, time.UTC)
	day := 24 * time.Hour

	cases := []struct {
		name        string
		expiration  time.Time
		locked      bool
		cashId      int64
		notExtend   bool
		addTime     uint32
		maxDays     uint32
		wantReject  string
		wantNewTime time.Time
	}{
		{
			name:        "under cap accepts",
			expiration:  now.Add(5 * day),
			addTime:     604800, // +7d
			maxDays:     30,
			wantNewTime: now.Add(12 * day),
		},
		{
			name:        "exactly at cap accepts",
			expiration:  now.Add(23 * day),
			addTime:     604800, // +7d -> exactly now+30d
			maxDays:     30,
			wantNewTime: now.Add(30 * day),
		},
		{
			name:       "over cap rejects without consuming",
			expiration: now.Add(25 * day),
			addTime:    604800, // +7d -> now+32d, past the ceiling
			maxDays:    30,
			wantReject: "over cap",
		},
		{
			name:       "already past cap rejects",
			expiration: now.Add(40 * day),
			addTime:    604800,
			maxDays:    30,
			wantReject: "over cap",
		},
		{
			name:       "99-day extender against a 30-day ceiling always rejects",
			expiration: now.Add(1 * day),
			addTime:    8553600, // 99d
			maxDays:    30,
			wantReject: "over cap",
		},
		{
			name:       "zero maxDays rejects",
			expiration: now.Add(5 * day),
			addTime:    604800,
			maxDays:    0,
			wantReject: "no ceiling",
		},
		{
			name:       "permanent target rejects",
			expiration: time.Time{},
			addTime:    604800,
			maxDays:    30,
			wantReject: "permanent",
		},
		{
			name:       "cash equip rejects",
			expiration: now.Add(5 * day),
			cashId:     987654321,
			addTime:    604800,
			maxDays:    30,
			wantReject: "cash",
		},
		{
			name:       "locked target rejects",
			expiration: now.Add(5 * day),
			locked:     true,
			addTime:    604800,
			maxDays:    30,
			wantReject: "lock",
		},
		{
			name:       "notExtend target rejects",
			expiration: now.Add(5 * day),
			notExtend:  true,
			addTime:    604800,
			maxDays:    30,
			wantReject: "notExtend",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := evaluateExpirationExtension(now, extensionTarget{
				Expiration: c.expiration,
				Locked:     c.locked,
				CashId:     c.cashId,
				NotExtend:  c.notExtend,
			}, c.addTime, c.maxDays)

			if c.wantReject != "" {
				if got.Reason == "" {
					t.Fatalf("expected rejection (%s), got acceptance with %v", c.wantReject, got.Expiration)
				}
				return
			}
			if got.Reason != "" {
				t.Fatalf("expected acceptance, got rejection: %s", got.Reason)
			}
			if !got.Expiration.Equal(c.wantNewTime) {
				t.Errorf("Expiration = %v, want %v", got.Expiration, c.wantNewTime)
			}
		})
	}
}

// Double-clicking the extender in the CASH tab is a client dead-end, not a
// use: CDraggableItem::OnDoubleClicked (gms_v83 @0x4efd25) falls through
// get_cashslot_item_type 61 into the get_consume_cash_item_type allow-list
// (@0x4863d5) and reaches the send @0x4f05a6 with a hard-coded nEPOS of 0 --
// it never hit-tests a target. The supported flow is the drag-drop one,
// CDraggableItem::ModifyEquipItem (@0x4f4bb7).
//
// SendConsumeCashItemUseRequest is the sole caller of SetExclRequestSent
// (@0xa0ea6f -> @0xa0ebbc), so the excl lock is already armed when the packet
// arrives. This arm consumes and mutates nothing on rejection, so if it
// returns silently the client is wedged for the rest of the session -- there
// is no client-side timeout. It must announce the hint plus the
// enable-actions unlock, and issue no saga.
func TestExpirationExtenderEmptyTargetUnlocksAndConsumesNothing(t *testing.T) {
	const characterId = uint32(4242)
	const itemId = uint32(5500001)
	const sourceSlot = int16(3)

	// Resolve GetItemInSlot(EQUIP, 0) to a 404 so the arm takes its
	// empty-slot branch, exactly as it does live for the nEPOS-0 request.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	t.Setenv("CHARACTERS_SERVICE_URL", srv.URL+"/api/")

	restoreSlot := installCashItemInSlotSeam(t, sourceSlot, itemId)
	defer restoreSlot()
	captured, restoreProducer := installCapturingProducer()
	defer restoreProducer()

	s, ctx, cleanup := newCashItemUseTestSession(t, characterId)
	defer cleanup()

	raw := append(cashItemUsePrefix(sourceSlot, itemId),
		0x00, 0x00, // nEPOS = 0, the double-click sentinel
		0x00, 0x00, 0x00, 0x00, // trailing updateTime (GMS v83)
	)
	req := request.Request(raw)
	reader := request.NewRequestReader(&req, 0)

	rec := &gaugeProducerRecorder{}
	CharacterCashItemUseHandleFunc(logrus.New(), ctx, rec.producer())(s, &reader, map[string]interface{}{})

	for topic, msgs := range *captured {
		if len(msgs) != 0 {
			t.Errorf("emitted %d commands on %q, want 0 — nothing may be consumed", len(msgs), topic)
		}
	}
	if rec.calls != 2 {
		t.Fatalf("announced %d packets, want 2 (the hint and the enable-actions unlock)", rec.calls)
	}
	if rec.lastName != statpkt.StatChangedWriter {
		t.Errorf("last announce = %q, want %q — the unlock must be the final packet", rec.lastName, statpkt.StatChangedWriter)
	}
}
