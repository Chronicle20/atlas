package buff

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
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

// TestHasActiveGmHide_v83CanonicalWire pins the baseline: the v0.83
// canonical SuperGmHide wire id (9101004) must read as GM-hidden under a
// v83 tenant, and an unrelated skill (Rogue Dark Sight) must not.
func TestHasActiveGmHide_v83CanonicalWire(t *testing.T) {
	ctx := testCtx(t, "GMS", 83, 1)
	future := time.Now().Add(time.Hour)
	past := time.Now().Add(-time.Hour)

	hide := NewModel(int32(skill.SuperGmHideId), future)
	if !HasActiveGmHide(ctx, []Model{hide}) {
		t.Errorf("HasActiveGmHide = false for an active v83 SuperGmHide buff, want true")
	}

	// Rogue Dark Sight is a different source and must NOT read as GM-hidden.
	darkSight := NewModel(int32(skill.RogueDarkSightId), future)
	if HasActiveGmHide(ctx, []Model{darkSight}) {
		t.Errorf("HasActiveGmHide = true for a Rogue Dark Sight buff, want false")
	}

	// An expired SuperGmHide buff does not count as hidden.
	expired := NewModel(int32(skill.SuperGmHideId), past)
	if HasActiveGmHide(ctx, []Model{expired}) {
		t.Errorf("HasActiveGmHide = true for an expired SuperGmHide buff, want false")
	}

	if HasActiveGmHide(ctx, nil) {
		t.Errorf("HasActiveGmHide = true for nil buff slice, want false")
	}
}

// TestHasActiveGmHide_v48Wire pins the task-187 v0.48 correctness property
// for GM-hide detection: at v0.48, SuperGmHide's wire id is 5101004, not
// the canonical 9101004 (skill.SuperGmHideId). A buff sourced from wire
// 5101004 must read as GM-hidden under a v0.48 tenant, and the v83-canonical
// wire value must NOT (it is a different, non-hide skill at v48).
func TestHasActiveGmHide_v48Wire(t *testing.T) {
	ctx48 := testCtx(t, "GMS", 48, 1)
	future := time.Now().Add(time.Hour)

	hideV48 := NewModel(5101004, future)
	if !HasActiveGmHide(ctx48, []Model{hideV48}) {
		t.Errorf("HasActiveGmHide = false for a v48 SuperGmHide buff (source wire 5101004), want true")
	}

	// The v83-canonical wire value is a DIFFERENT skill at v48 and must not
	// be misread as hide.
	canonicalUnderV48 := NewModel(int32(skill.SuperGmHideId), future)
	if HasActiveGmHide(ctx48, []Model{canonicalUnderV48}) {
		t.Errorf("HasActiveGmHide = true for a buff sourced from the v83-canonical wire id (%d) under a v48 tenant, want false", skill.SuperGmHideId)
	}
}
