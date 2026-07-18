package buff

import (
	"math"
	"testing"
	"time"

	skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"
)

func TestIsGmHidden(t *testing.T) {
	future := time.Now().Add(time.Hour)
	past := time.Now().Add(-time.Hour)

	hide := NewBuff(int32(skill2.SuperGmHideId), 1, math.MaxInt32, nil, time.Now(), future)
	if !IsGmHidden([]Model{hide}) {
		t.Errorf("IsGmHidden = false for an active SuperGmHide buff, want true")
	}

	// Rogue Dark Sight is a different source and must NOT read as GM-hidden,
	// even though it also produces a DARK_SIGHT stat.
	darkSight := NewBuff(int32(skill2.RogueDarkSightId), 1, 1000, nil, time.Now(), future)
	if IsGmHidden([]Model{darkSight}) {
		t.Errorf("IsGmHidden = true for a Rogue Dark Sight buff, want false")
	}

	// An expired SuperGmHide buff does not count as hidden.
	expired := NewBuff(int32(skill2.SuperGmHideId), 1, math.MaxInt32, nil, past.Add(-time.Hour), past)
	if IsGmHidden([]Model{expired}) {
		t.Errorf("IsGmHidden = true for an expired SuperGmHide buff, want false")
	}

	if IsGmHidden(nil) {
		t.Errorf("IsGmHidden = true for nil buff slice, want false")
	}
}
