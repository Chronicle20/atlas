package handler

import (
	"atlas-channel/pet"
	"atlas-channel/saga"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	sagaMsg "atlas-channel/kafka/message/saga"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
)

// FR-1.2/FR-1.3. The pre-fix predicate was `10000*itemId/10000 != itemId`.
// item.Id is uint32, so 10000 * 5170000 = 51,700,000,000 wraps to 160,392,448;
// /10000 = 16039 != 5170000, and the branch returned 0 — the Pet Name Tag never
// reached a handler at all. The client's actual rule is `itemId % 10000 != 0`
// (get_cashslot_item_type @0x48645b, case 517). THIS TEST FAILS AGAINST THE
// PRE-FIX PREDICATE — that is the point of it.
func TestGetCashSlotItemTypePetNameTag(t *testing.T) {
	tm := mustTenant(t, "GMS", 83, 1)

	if got := GetCashSlotItemType(tm)(5170000); got != CashSlotItemTypePetNameTag {
		t.Fatalf("GetCashSlotItemType(5170000) = %d, want 17", got)
	}
	if got := GetCashSlotItemType(tm)(5170001); got != CashSlotItemType(0) {
		t.Fatalf("GetCashSlotItemType(5170001) = %d, want 0", got)
	}
}

// The happy path builds exactly one saga, rename-first, with PreviousName
// captured (PRD FR-7.2/FR-7.4).
func TestHandlePetNameTagUseBuildsRenameFirstSaga(t *testing.T) {
	s := buildPetNameTagUseSaga(uuid.New(), time.Now(), 42, 5170000, 7, "Renamed", "Original")

	if s.SagaType != saga.PetNameTagUse {
		t.Fatalf("SagaType = %s", s.SagaType)
	}
	if len(s.Steps) != 2 {
		t.Fatalf("steps = %d, want 2", len(s.Steps))
	}
	if s.Steps[0].StepId != "rename_pet" || s.Steps[0].Action != saga.RenamePet {
		t.Fatalf("step 1 = %s/%s, want rename_pet/rename_pet", s.Steps[0].StepId, s.Steps[0].Action)
	}
	if s.Steps[1].StepId != "consume_pet_name_tag" || s.Steps[1].Action != saga.DestroyAsset {
		t.Fatalf("step 2 = %s/%s, want consume_pet_name_tag/destroy_asset", s.Steps[1].StepId, s.Steps[1].Action)
	}
	p, ok := s.Steps[0].Payload.(saga.RenamePetPayload)
	if !ok {
		t.Fatalf("step 1 payload type = %T", s.Steps[0].Payload)
	}
	if p.PetId != 7 || p.CharacterId != 42 || p.Name != "Renamed" || p.PreviousName != "Original" {
		t.Fatalf("payload = %+v", p)
	}
}

// Every rejection consumes nothing and starts no saga, and unlocks the client
// (PRD FR-7.3). Two announces per rejection: the pink text, then enable-actions.
func TestHandlePetNameTagUseRejectsAndUnlocks(t *testing.T) {
	const characterId = uint32(4242)

	cases := []struct {
		name    string
		pets    []pet.Model
		err     error
		newName string
	}{
		{"pet lookup failure", nil, errors.New("503"), "Renamed"},
		{"no active pet", nil, nil, "Renamed"},
		{"no lead pet", petsAtSlots(characterId, 1, 2), nil, "Renamed"},
		{"name too short", leadPetNamed(characterId, "Original"), nil, "abc"},
		{"name too long", leadPetNamed(characterId, "Original"), nil, "abcdefghijklm"},
		{"whitespace only", leadPetNamed(characterId, "Original"), nil, "     "},
		{"unchanged name", leadPetNamed(characterId, "Original"), nil, "Original"},
		{"unchanged after trim", leadPetNamed(characterId, "Original"), nil, "  Original  "},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			restorePets := installPetsForOwnerSeam(t, tc.pets, tc.err)
			defer restorePets()
			captured, restoreProducer := installCapturingProducer()
			defer restoreProducer()

			s, ctx, cleanup := newCashItemUseTestSession(t, characterId)
			defer cleanup()

			rec := &gaugeProducerRecorder{}
			handlePetNameTagUse(logrus.New(), ctx, rec.producer())(s, item.Id(5170000), tc.newName)

			if got := len((*captured)[sagaMsg.EnvCommandTopic]); got != 0 {
				t.Fatalf("emitted %d saga commands, want 0 (nothing may be consumed)", got)
			}
			if rec.calls != 2 {
				t.Fatalf("announced %d packets, want 2 (pink text + enable-actions unlock)", rec.calls)
			}
		})
	}
}

// The happy path emits exactly one saga command and announces nothing: the
// success unlock rides on the consume step's INVENTORY_OPERATION. An extra
// empty StatChanged here would race it.
func TestHandlePetNameTagUseCreatesSagaAndAnnouncesNothing(t *testing.T) {
	const characterId = uint32(4242)

	restorePets := installPetsForOwnerSeam(t, leadPetNamed(characterId, "Original"), nil)
	defer restorePets()
	captured, restoreProducer := installCapturingProducer()
	defer restoreProducer()

	s, ctx, cleanup := newCashItemUseTestSession(t, characterId)
	defer cleanup()

	rec := &gaugeProducerRecorder{}
	handlePetNameTagUse(logrus.New(), ctx, rec.producer())(s, item.Id(5170000), "Renamed")

	if got := len((*captured)[sagaMsg.EnvCommandTopic]); got != 1 {
		t.Fatalf("emitted %d saga commands, want exactly 1", got)
	}
	if rec.calls != 0 {
		t.Fatalf("announced %d packets on the success path, want 0", rec.calls)
	}
}

// installPetsForOwnerSeam swaps the package-var pet lookup, mirroring
// installCashItemDataSeam in character_cash_item_use_meso_sack_test.go.
func installPetsForOwnerSeam(t *testing.T, ps []pet.Model, err error) func() {
	t.Helper()
	original := petsForOwnerFunc
	petsForOwnerFunc = func(logrus.FieldLogger, context.Context, uint32) ([]pet.Model, error) {
		return ps, err
	}
	return func() { petsForOwnerFunc = original }
}

func leadPetNamed(ownerId uint32, name string) []pet.Model {
	return []pet.Model{
		pet.NewModelBuilder(7, 0, 5000000, name).SetOwnerID(ownerId).SetSlot(0).MustBuild(),
	}
}

func petsAtSlots(ownerId uint32, slots ...int8) []pet.Model {
	ps := make([]pet.Model, 0, len(slots))
	for i, sl := range slots {
		ps = append(ps, pet.NewModelBuilder(uint32(100+i), 0, 5000000, "Other").SetOwnerID(ownerId).SetSlot(sl).MustBuild())
	}
	return ps
}
