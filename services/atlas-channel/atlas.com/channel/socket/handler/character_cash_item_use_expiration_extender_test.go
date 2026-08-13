package handler

import (
	"testing"
	"time"

	"github.com/google/uuid"

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
