package consumable

import (
	"testing"
	"time"

	"github.com/google/uuid"

	item2 "github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	sharedsaga "github.com/Chronicle20/atlas/libs/atlas-saga"
)

// Job-prefix rule (Cosmic ItemInformationProvider.java:1481-1483):
// first skills[] entry with skillId/10000 == jobId; no match ⇒ unusable.
func TestSelectSkillBookTargetSkill(t *testing.T) {
	// v83 WZ 2290000 (Monster Magnet books): [1121001 Hero, 1221001 Paladin, 1321001 DK]
	skills := []uint32{1121001, 1221001, 1321001}
	tests := []struct {
		name   string
		jobId  job.Id
		wantId uint32
		wantOk bool
	}{
		{"hero matches first entry", job.Id(112), 1121001, true},
		{"paladin matches second entry", job.Id(122), 1221001, true},
		{"dark knight matches third entry", job.Id(132), 1321001, true},
		{"bishop matches nothing", job.Id(232), 0, false},
		{"beginner matches nothing", job.Id(0), 0, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := SelectSkillBookTargetSkill(skills, tc.jobId)
			if ok != tc.wantOk || got != tc.wantId {
				t.Errorf("got (%d,%t), want (%d,%t)", got, ok, tc.wantId, tc.wantOk)
			}
		})
	}
}

// First-match wins when multiple entries share the job prefix.
func TestSelectSkillBookTargetSkillFirstMatch(t *testing.T) {
	got, ok := SelectSkillBookTargetSkill([]uint32{1121001, 1121002}, job.Id(112))
	if !ok || got != 1121001 {
		t.Errorf("got (%d,%t), want (1121001,true)", got, ok)
	}
}

// Strictly-less-than semantics (design §2 D-3): success=0 NEVER passes,
// success=100 ALWAYS passes. This is deliberately NOT the scroll path's <=.
func TestSkillBookRollPassesBoundaries(t *testing.T) {
	tests := []struct {
		roll int32
		rate uint32
		want bool
	}{
		{0, 0, false},   // success=0 never passes even on the best roll
		{99, 0, false},  //
		{0, 100, true},  // success=100 always passes
		{99, 100, true}, //
		{69, 70, true},  // roll < rate passes
		{70, 70, false}, // roll == rate fails (the <= off-by-one guard)
		{71, 70, false},
	}
	for _, tc := range tests {
		if got := SkillBookRollPasses(tc.roll, tc.rate); got != tc.want {
			t.Errorf("SkillBookRollPasses(%d, %d) = %t, want %t", tc.roll, tc.rate, got, tc.want)
		}
	}
}

// FR-2.5–2.8 skill-state gates. 228 (skill book) may target an unlearned
// skill only when reqSkillLevel==0 (D-1); 229 (mastery book) requires a
// learned record (Level >= 1).
func TestValidateSkillBookSkillState(t *testing.T) {
	tests := []struct {
		name               string
		isMasteryBook      bool
		hasRecord          bool
		currentLevel       byte
		currentMasterLevel byte
		reqSkillLevel      uint32
		bookMasterLevel    uint32
		wantErr            error
	}{
		{"mastery: happy path", true, true, 5, 10, 5, 20, nil},
		{"mastery: no record", true, false, 0, 0, 5, 20, ErrSkillBookSkillNotLearned},
		{"mastery: record at level 0", true, true, 0, 10, 0, 20, ErrSkillBookSkillNotLearned},
		{"mastery: below reqSkillLevel", true, true, 4, 10, 5, 20, ErrSkillBookReqSkillLevel},
		{"mastery: at reqSkillLevel passes", true, true, 5, 10, 5, 20, nil},
		{"mastery: master level at ceiling", true, true, 5, 20, 5, 20, ErrSkillBookMasterLevelCeiling},
		{"mastery: master level above ceiling", true, true, 5, 30, 5, 20, ErrSkillBookMasterLevelCeiling},
		{"skill book: no record, req 0 (teach)", false, false, 0, 0, 0, 10, nil},
		{"skill book: no record, req > 0", false, false, 0, 0, 1, 10, ErrSkillBookReqSkillLevel},
		{"skill book: existing record ok", false, true, 3, 5, 0, 10, nil},
		{"skill book: existing at ceiling", false, true, 3, 10, 0, 10, ErrSkillBookMasterLevelCeiling},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ValidateSkillBookSkillState(tc.isMasteryBook, tc.hasRecord, tc.currentLevel, tc.currentMasterLevel, tc.reqSkillLevel, tc.bookMasterLevel)
			if got != tc.wantErr {
				t.Errorf("got %v, want %v", got, tc.wantErr)
			}
		})
	}
}

// Saga shapes (design §5.3): destroy step always (consume-on-fail, with
// TemplateId for compensation); skill step only on a passing roll —
// update_skill carries CURRENT level/expiration (atlas-skills Update clobbers
// unconditionally), create_skill teaches at level 0 with zero expiration.
func TestBuildSkillBookSaga(t *testing.T) {
	txId := uuid.New()
	exp := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)

	t.Run("failed roll: destroy only", func(t *testing.T) {
		s := BuildSkillBookSaga(txId, 100, 5, item2.Id(2290000), false, true, 9, exp, 1121001, 20)
		if s.TransactionId != txId {
			t.Errorf("transactionId: got %v", s.TransactionId)
		}
		if s.SagaType != sharedsaga.SkillBookUse {
			t.Errorf("sagaType: got %v", s.SagaType)
		}
		if len(s.Steps) != 1 {
			t.Fatalf("steps: got %d, want 1", len(s.Steps))
		}
		if s.Steps[0].Action != sharedsaga.DestroyAssetFromSlot {
			t.Errorf("step 0 action: got %v", s.Steps[0].Action)
		}
		p, ok := s.Steps[0].Payload.(sharedsaga.DestroyAssetFromSlotPayload)
		if !ok {
			t.Fatalf("step 0 payload type: %T", s.Steps[0].Payload)
		}
		if p.CharacterId != 100 || p.InventoryType != 2 || p.Slot != 5 || p.Quantity != 1 || p.TemplateId != 2290000 {
			t.Errorf("destroy payload: %+v", p)
		}
	})

	t.Run("passed roll with record: destroy + update_skill carrying current level/expiration", func(t *testing.T) {
		s := BuildSkillBookSaga(txId, 100, 5, item2.Id(2290000), true, true, 9, exp, 1121001, 20)
		if len(s.Steps) != 2 {
			t.Fatalf("steps: got %d, want 2", len(s.Steps))
		}
		if s.Steps[1].Action != sharedsaga.UpdateSkill {
			t.Errorf("step 1 action: got %v", s.Steps[1].Action)
		}
		p, ok := s.Steps[1].Payload.(sharedsaga.UpdateSkillPayload)
		if !ok {
			t.Fatalf("step 1 payload type: %T", s.Steps[1].Payload)
		}
		if p.CharacterId != 100 || p.SkillId != 1121001 || p.Level != 9 || p.MasterLevel != 20 || !p.Expiration.Equal(exp) {
			t.Errorf("update payload: %+v", p)
		}
	})

	t.Run("passed roll without record: destroy + create_skill at level 0, permanent", func(t *testing.T) {
		s := BuildSkillBookSaga(txId, 100, 5, item2.Id(2280000), true, false, 0, time.Time{}, 2121003, 10)
		if len(s.Steps) != 2 {
			t.Fatalf("steps: got %d, want 2", len(s.Steps))
		}
		if s.Steps[1].Action != sharedsaga.CreateSkill {
			t.Errorf("step 1 action: got %v", s.Steps[1].Action)
		}
		p, ok := s.Steps[1].Payload.(sharedsaga.CreateSkillPayload)
		if !ok {
			t.Fatalf("step 1 payload type: %T", s.Steps[1].Payload)
		}
		if p.CharacterId != 100 || p.SkillId != 2121003 || p.Level != 0 || p.MasterLevel != 10 || !p.Expiration.IsZero() {
			t.Errorf("create payload: %+v", p)
		}
	})
}
