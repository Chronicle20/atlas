package buff

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/google/uuid"

	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func testCtx(t *testing.T, region string, major, minor uint16) context.Context {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), region, major, minor)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	return tenant.WithContext(context.Background(), tm)
}

func TestIsGmHidden(t *testing.T) {
	ctx := testCtx(t, "GMS", 83, 1)
	future := time.Now().Add(time.Hour)
	past := time.Now().Add(-time.Hour)

	hide := NewBuff(int32(skill2.SuperGmHideId), 1, math.MaxInt32, nil, time.Now(), future, false)
	if !IsGmHidden(ctx, []Model{hide}) {
		t.Errorf("IsGmHidden = false for an active SuperGmHide buff, want true")
	}

	// Rogue Dark Sight is a different source and must NOT read as GM-hidden,
	// even though it also produces a DARK_SIGHT stat.
	darkSight := NewBuff(int32(skill2.RogueDarkSightId), 1, 1000, nil, time.Now(), future, false)
	if IsGmHidden(ctx, []Model{darkSight}) {
		t.Errorf("IsGmHidden = true for a Rogue Dark Sight buff, want false")
	}

	// An expired SuperGmHide buff does not count as hidden.
	expired := NewBuff(int32(skill2.SuperGmHideId), 1, math.MaxInt32, nil, past.Add(-time.Hour), past, false)
	if IsGmHidden(ctx, []Model{expired}) {
		t.Errorf("IsGmHidden = true for an expired SuperGmHide buff, want false")
	}

	if IsGmHidden(ctx, nil) {
		t.Errorf("IsGmHidden = true for nil buff slice, want false")
	}
}

// TestIsGmHidden_v48Wire pins the task-187 v0.48 correctness property for
// GM-hide detection: at v0.48, SuperGmHide's wire id is 5101004, not the
// canonical 9101004 (skill2.SuperGmHideId). A buff sourced from wire 5101004
// must read as GM-hidden under a v0.48 tenant.
func TestIsGmHidden_v48Wire(t *testing.T) {
	ctx48 := testCtx(t, "GMS", 48, 1)
	future := time.Now().Add(time.Hour)

	hideV48 := NewBuff(5101004, 1, math.MaxInt32, nil, time.Now(), future, false)
	if !IsGmHidden(ctx48, []Model{hideV48}) {
		t.Errorf("IsGmHidden = false for a v48 SuperGmHide buff (source wire 5101004), want true")
	}

	// The v83-canonical wire value is a DIFFERENT (Brawler-band) skill at
	// v48 and must not be misread as hide.
	canonicalUnderV48 := NewBuff(int32(skill2.SuperGmHideId), 1, math.MaxInt32, nil, time.Now(), future, false)
	if IsGmHidden(ctx48, []Model{canonicalUnderV48}) {
		t.Errorf("IsGmHidden = true for a buff sourced from the v83-canonical wire id (%d) under a v48 tenant, want false", skill2.SuperGmHideId)
	}
}

// TestIsGmHidden_v72CorkscrewNotHidden guards the inverse of the PRD-motivating
// bug: at v0.72, wire 5101004 is BrawlerCorkscrewBlow (an attack skill), not
// SuperGmHide, so a buff sourced from that wire under a v72 tenant must NOT
// read as GM-hidden.
func TestIsGmHidden_v72CorkscrewNotHidden(t *testing.T) {
	ctx72 := testCtx(t, "GMS", 72, 1)
	future := time.Now().Add(time.Hour)

	corkscrewV72 := NewBuff(5101004, 1, 1000, nil, time.Now(), future, false)
	if IsGmHidden(ctx72, []Model{corkscrewV72}) {
		t.Errorf("IsGmHidden = true for a v72 wire-5101004 (BrawlerCorkscrewBlow) buff, want false")
	}
}
