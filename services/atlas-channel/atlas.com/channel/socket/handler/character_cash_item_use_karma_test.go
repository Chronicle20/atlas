package handler

import (
	"atlas-channel/asset"
	character2 "atlas-channel/character"
	charmock "atlas-channel/character/mock"
	cashData "atlas-channel/data/cash"
	cashmock "atlas-channel/data/cash/mock"
	"atlas-channel/data/tradeability"
	trademock "atlas-channel/data/tradeability/mock"
	sagaMsg "atlas-channel/kafka/message/saga"
	"atlas-channel/saga"
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/inventory"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

// installKarmaCharacterSeam swaps karmaCharacterProcessorFunc for the test,
// returning a character2.Processor backed by character/mock.MockProcessor
// (package-var injection precedent: installCashItemInSlotSeam above).
func installKarmaCharacterSeam(t *testing.T, fn func(characterId uint32, invType inventory.Type, slot int16) (asset.Model, error)) func() {
	t.Helper()
	orig := karmaCharacterProcessorFunc
	karmaCharacterProcessorFunc = func(_ logrus.FieldLogger, _ context.Context) character2.Processor {
		return &charmock.MockProcessor{GetItemInSlotFunc: fn}
	}
	return func() { karmaCharacterProcessorFunc = orig }
}

// installKarmaCashDataSeam swaps karmaCashDataProcessorFunc for the test.
func installKarmaCashDataSeam(t *testing.T, fn func(itemId uint32) (cashData.RestModel, error)) func() {
	t.Helper()
	orig := karmaCashDataProcessorFunc
	karmaCashDataProcessorFunc = func(_ logrus.FieldLogger, _ context.Context) cashData.Processor {
		return &cashmock.ProcessorMock{GetByIdFunc: fn}
	}
	return func() { karmaCashDataProcessorFunc = orig }
}

// installKarmaTradeabilitySeam swaps karmaTradeabilityProcessorFunc for the
// test. Every karma test must call this explicitly — the mock's own doc
// comment (data/tradeability/mock/processor.go) warns that an unset GetFunc
// defaults to a zero-valued Model with a NIL error, i.e. "tradeable, no karma
// type", which is exactly the permissive default the real Processor refuses
// to produce.
func installKarmaTradeabilitySeam(t *testing.T, fn func(invType inventory.Type, templateId item.Id) (tradeability.Model, error)) func() {
	t.Helper()
	orig := karmaTradeabilityProcessorFunc
	karmaTradeabilityProcessorFunc = func(_ logrus.FieldLogger, _ context.Context) tradeability.Processor {
		return &trademock.ProcessorMock{GetFunc: fn}
	}
	return func() { karmaTradeabilityProcessorFunc = orig }
}

// karmaScissorsSlot is the fixed CASH-compartment slot the scissors occupy
// for every test in this file.
const karmaScissorsSlot = int16(4)

// karmaScissorsItemId is classification 552 (item.ClassificationKarmaScissors)
// on every configured version (see TestGetCashSlotItemTypeFor552Unchanged in
// karma_slot_type_test.go); the v83 GMS tenant newCashItemUseTestSession
// builds resolves it to CashSlotItemTypeKarmaScissors (63).
const karmaScissorsItemId = uint32(5520001)

// karmaScissorsRequest builds the real wire payload for a v83 GMS karma
// scissors cash-item use: the common cashsb.ItemUse prefix (source slot,
// itemId), then ItemUseKarmaScissors's sub-body (int32 inventoryType, int32
// slot), then the trailing updateTime (v83's updateTimeFirst is false, see
// newCashItemUseTestSession's doc comment).
func karmaScissorsRequest(itemId uint32, invTypeRaw int32, targetSlot int32) *request.Reader {
	raw := append(cashItemUsePrefix(karmaScissorsSlot, itemId),
		byte(invTypeRaw), byte(invTypeRaw>>8), byte(invTypeRaw>>16), byte(invTypeRaw>>24),
		byte(targetSlot), byte(targetSlot>>8), byte(targetSlot>>16), byte(targetSlot>>24),
		0x2A, 0x00, 0x00, 0x00, // trailing updateTime = 42
	)
	req := request.Request(raw)
	reader := request.NewRequestReader(&req, 0)
	return &reader
}

// karmaBundleTemplateId is an ordinary non-pet, non-equip template id
// (classification 200, a consumable) so af.KarmaFlagFor resolves the BUNDLE
// karma bit (FlagKarmaUse) rather than refusing as a pet.
const karmaBundleTemplateId = uint32(2000000)

// karmaPetTemplateId is classification 500 (item.ClassificationPet): the one
// class af.KarmaFlagFor deliberately refuses to resolve (gate 0c).
const karmaPetTemplateId = uint32(5000000)

func karmaTargetAsset(templateId uint32, locked bool, karmaUsed bool, untradeable bool) asset.Model {
	b := asset.NewBuilder(uuid.New(), templateId).
		SetId(1).
		SetSlot(5).
		SetLocked(locked).
		SetKarmaUsed(karmaUsed).
		SetCanBeTraded(!untradeable)
	return b.MustBuild()
}

func TestKarmaArmRefusals(t *testing.T) {
	unreachableChar := func(_ uint32, _ inventory.Type, _ int16) (asset.Model, error) {
		t.Fatal("karma character lookup reached unexpectedly")
		return asset.Model{}, nil
	}
	unreachableCash := func(_ uint32) (cashData.RestModel, error) {
		t.Fatal("karma cash-data lookup reached unexpectedly")
		return cashData.RestModel{}, nil
	}
	unreachableTrade := func(_ inventory.Type, _ item.Id) (tradeability.Model, error) {
		t.Fatal("karma tradeability lookup reached unexpectedly")
		return tradeability.Model{}, nil
	}

	testCases := []struct {
		name string

		slotMismatch bool // gate 0a: cashItemInSlotFunc reports a DIFFERENT template

		invTypeRaw int32
		targetSlot int32

		charFunc  func(characterId uint32, invType inventory.Type, slot int16) (asset.Model, error)
		cashFunc  func(itemId uint32) (cashData.RestModel, error)
		tradeFunc func(invType inventory.Type, templateId item.Id) (tradeability.Model, error)

		wantUnlock int
	}{
		{
			name:         "0a scissors not in the claimed cash slot",
			slotMismatch: true,
			invTypeRaw:   int32(inventory.TypeValueUse),
			targetSlot:   5,
			charFunc:     unreachableChar,
			cashFunc:     unreachableCash,
			tradeFunc:    unreachableTrade,
			// Gate 0a is cashItemInSlotFunc's pre-existing early return
			// (character_cash_item_use.go:55-59), which predates the arm and
			// does not announce an unlock. Documented divergence: unlike
			// every gate inside the arm, this one leaves the client locked.
			wantUnlock: 0,
		},
		{
			name: "0b unknown target inventory type",
			// 200 is > math.MaxInt8: the signed-int8 wraparound trap the arm's
			// doc comment calls out explicitly.
			invTypeRaw: 200,
			targetSlot: 5,
			charFunc:   unreachableChar,
			cashFunc:   unreachableCash,
			tradeFunc:  unreachableTrade,
			wantUnlock: 1,
		},
		{
			name:       "0c pet-class target",
			invTypeRaw: int32(inventory.TypeValueUse),
			targetSlot: 5,
			charFunc: func(_ uint32, _ inventory.Type, _ int16) (asset.Model, error) {
				return karmaTargetAsset(karmaPetTemplateId, false, false, false), nil
			},
			cashFunc:   unreachableCash,
			tradeFunc:  unreachableTrade,
			wantUnlock: 1,
		},
		{
			name:       "0d negative target slot (equipped)",
			invTypeRaw: int32(inventory.TypeValueEquip),
			targetSlot: -5,
			charFunc:   unreachableChar,
			cashFunc:   unreachableCash,
			tradeFunc:  unreachableTrade,
			wantUnlock: 1,
		},
		{
			name:       "0e empty target slot",
			invTypeRaw: int32(inventory.TypeValueUse),
			targetSlot: 5,
			charFunc: func(_ uint32, _ inventory.Type, _ int16) (asset.Model, error) {
				return asset.Model{}, errors.New("item not found")
			},
			cashFunc:   unreachableCash,
			tradeFunc:  unreachableTrade,
			wantUnlock: 1,
		},
		{
			name:       "1 target is sealing-locked",
			invTypeRaw: int32(inventory.TypeValueUse),
			targetSlot: 5,
			charFunc: func(_ uint32, _ inventory.Type, _ int16) (asset.Model, error) {
				return karmaTargetAsset(karmaBundleTemplateId, true, false, false), nil
			},
			cashFunc:   unreachableCash,
			tradeFunc:  unreachableTrade,
			wantUnlock: 1,
		},
		{
			name:       "2 target tradeAvailable is 0",
			invTypeRaw: int32(inventory.TypeValueUse),
			targetSlot: 5,
			charFunc: func(_ uint32, _ inventory.Type, _ int16) (asset.Model, error) {
				return karmaTargetAsset(karmaBundleTemplateId, false, false, false), nil
			},
			cashFunc: func(_ uint32) (cashData.RestModel, error) {
				return cashData.RestModel{Karma: 0}, nil
			},
			tradeFunc: func(_ inventory.Type, _ item.Id) (tradeability.Model, error) {
				return tradeability.NewModel(false, 0, false), nil
			},
			wantUnlock: 1,
		},
		{
			name:       "2 target tradeAvailable differs from the scissors' karma type",
			invTypeRaw: int32(inventory.TypeValueUse),
			targetSlot: 5,
			charFunc: func(_ uint32, _ inventory.Type, _ int16) (asset.Model, error) {
				return karmaTargetAsset(karmaBundleTemplateId, false, false, false), nil
			},
			cashFunc: func(_ uint32) (cashData.RestModel, error) {
				return cashData.RestModel{Karma: 2}, nil
			},
			tradeFunc: func(_ inventory.Type, _ item.Id) (tradeability.Model, error) {
				return tradeability.NewModel(false, 5, false), nil
			},
			wantUnlock: 1,
		},
		{
			name:       "3 target is already karma-marked",
			invTypeRaw: int32(inventory.TypeValueUse),
			targetSlot: 5,
			charFunc: func(_ uint32, _ inventory.Type, _ int16) (asset.Model, error) {
				// eligible (untyped scissors, target karma type 1) but already marked
				return karmaTargetAsset(karmaBundleTemplateId, false, true, false), nil
			},
			cashFunc: func(_ uint32) (cashData.RestModel, error) {
				return cashData.RestModel{Karma: 0}, nil
			},
			tradeFunc: func(_ inventory.Type, _ item.Id) (tradeability.Model, error) {
				return tradeability.NewModel(false, 1, false), nil
			},
			wantUnlock: 1,
		},
		{
			name:       "4 target is already tradeable",
			invTypeRaw: int32(inventory.TypeValueUse),
			targetSlot: 5,
			charFunc: func(_ uint32, _ inventory.Type, _ int16) (asset.Model, error) {
				// eligible, not yet marked, and carries none of the three
				// untradeable indicators (no Untradeable/MergeUntradeable flag,
				// tradeBlock false below) -> gate 4 fires.
				return karmaTargetAsset(karmaBundleTemplateId, false, false, false), nil
			},
			cashFunc: func(_ uint32) (cashData.RestModel, error) {
				return cashData.RestModel{Karma: 0}, nil
			},
			tradeFunc: func(_ inventory.Type, _ item.Id) (tradeability.Model, error) {
				return tradeability.NewModel(false, 1, false), nil
			},
			wantUnlock: 1,
		},
		{
			name:       "unreadable scissors cash data",
			invTypeRaw: int32(inventory.TypeValueUse),
			targetSlot: 5,
			charFunc: func(_ uint32, _ inventory.Type, _ int16) (asset.Model, error) {
				return karmaTargetAsset(karmaBundleTemplateId, false, false, false), nil
			},
			cashFunc: func(_ uint32) (cashData.RestModel, error) {
				return cashData.RestModel{}, errors.New("404")
			},
			tradeFunc:  unreachableTrade,
			wantUnlock: 1,
		},
		{
			name:       "unreadable target item data",
			invTypeRaw: int32(inventory.TypeValueUse),
			targetSlot: 5,
			charFunc: func(_ uint32, _ inventory.Type, _ int16) (asset.Model, error) {
				return karmaTargetAsset(karmaBundleTemplateId, false, false, false), nil
			},
			cashFunc: func(_ uint32) (cashData.RestModel, error) {
				return cashData.RestModel{Karma: 0}, nil
			},
			tradeFunc: func(_ inventory.Type, _ item.Id) (tradeability.Model, error) {
				return tradeability.Model{}, errors.New("404")
			},
			wantUnlock: 1,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			matchTemplate := karmaScissorsItemId
			if tc.slotMismatch {
				matchTemplate = karmaScissorsItemId + 1
			}
			restoreSlot := installCashItemInSlotSeam(t, karmaScissorsSlot, matchTemplate)
			defer restoreSlot()
			restoreChar := installKarmaCharacterSeam(t, tc.charFunc)
			defer restoreChar()
			restoreCash := installKarmaCashDataSeam(t, tc.cashFunc)
			defer restoreCash()
			restoreTrade := installKarmaTradeabilitySeam(t, tc.tradeFunc)
			defer restoreTrade()

			captured, restoreProducer := installCapturingProducer()
			defer restoreProducer()

			s, ctx, cleanup := newCashItemUseTestSession(t, 555)
			defer cleanup()

			rec := &gaugeProducerRecorder{}
			r := karmaScissorsRequest(karmaScissorsItemId, tc.invTypeRaw, tc.targetSlot)

			CharacterCashItemUseHandleFunc(logrus.New(), ctx, rec.producer())(s, r, map[string]interface{}{})

			if got := len((*captured)[sagaMsg.EnvCommandTopic]); got != 0 {
				t.Errorf("saga commands emitted = %d, want 0 (no state may mutate on a refusal)", got)
			}
			if rec.calls != tc.wantUnlock {
				t.Errorf("unlock announces = %d, want %d", rec.calls, tc.wantUnlock)
			}
		})
	}
}

// TestKarmaArmSuccessCreatesTwoStepSaga: consume first, mark second, so a
// failed mark compensates by restoring the scissors rather than leaving a
// free trade behind.
func TestKarmaArmSuccessCreatesTwoStepSaga(t *testing.T) {
	restoreSlot := installCashItemInSlotSeam(t, karmaScissorsSlot, karmaScissorsItemId)
	defer restoreSlot()
	restoreChar := installKarmaCharacterSeam(t, func(_ uint32, _ inventory.Type, _ int16) (asset.Model, error) {
		// Not locked, not yet karma-marked, and untradeable by data
		// (TradeBlock true) -- exactly the item karma exists to unlock.
		return karmaTargetAsset(karmaBundleTemplateId, false, false, false), nil
	})
	defer restoreChar()
	restoreCash := installKarmaCashDataSeam(t, func(_ uint32) (cashData.RestModel, error) {
		return cashData.RestModel{Karma: 3}, nil
	})
	defer restoreCash()
	restoreTrade := installKarmaTradeabilitySeam(t, func(_ inventory.Type, _ item.Id) (tradeability.Model, error) {
		return tradeability.NewModel(true, 3, false), nil
	})
	defer restoreTrade()

	captured, restoreProducer := installCapturingProducer()
	defer restoreProducer()

	s, ctx, cleanup := newCashItemUseTestSession(t, 555)
	defer cleanup()

	rec := &gaugeProducerRecorder{}
	invTypeRaw := int32(inventory.TypeValueUse)
	targetSlot := int32(5)
	r := karmaScissorsRequest(karmaScissorsItemId, invTypeRaw, targetSlot)

	CharacterCashItemUseHandleFunc(logrus.New(), ctx, rec.producer())(s, r, map[string]interface{}{})

	if rec.calls != 0 {
		t.Fatalf("announced %d packets on the success path, want 0 (success unlocks via INVENTORY_OPERATION, not this arm)", rec.calls)
	}

	msgs := (*captured)[sagaMsg.EnvCommandTopic]
	if len(msgs) != 1 {
		t.Fatalf("saga commands emitted = %d, want exactly 1", len(msgs))
	}

	var got saga.Saga
	if err := json.Unmarshal(msgs[0].Value, &got); err != nil {
		t.Fatalf("unmarshal saga: %v", err)
	}

	if got.SagaType != saga.KarmaScissorsUse {
		t.Errorf("sagaType = %s, want %s", got.SagaType, saga.KarmaScissorsUse)
	}
	if len(got.Steps) != 2 {
		t.Fatalf("steps = %d, want 2", len(got.Steps))
	}

	if got.Steps[0].Action != saga.DestroyAsset {
		t.Errorf("step 1 action = %s, want %s (destroy-first is mandatory)", got.Steps[0].Action, saga.DestroyAsset)
	}
	dp, ok := got.Steps[0].Payload.(saga.DestroyAssetPayload)
	if !ok {
		t.Fatalf("step 1 payload type = %T", got.Steps[0].Payload)
	}
	if dp.CharacterId != 555 || dp.TemplateId != karmaScissorsItemId || dp.Quantity != 1 {
		t.Errorf("destroy payload mismatch: %+v", dp)
	}

	if got.Steps[1].Action != saga.ApplyAssetKarma {
		t.Errorf("step 2 action = %s, want %s", got.Steps[1].Action, saga.ApplyAssetKarma)
	}
	ap, ok := got.Steps[1].Payload.(saga.ApplyAssetKarmaPayload)
	if !ok {
		t.Fatalf("step 2 payload type = %T", got.Steps[1].Payload)
	}
	if ap.CharacterId != 555 {
		t.Errorf("apply-karma characterId = %d, want 555", ap.CharacterId)
	}
	if ap.InventoryType != byte(inventory.TypeValueUse) {
		t.Errorf("apply-karma inventoryType = %d, want %d", ap.InventoryType, byte(inventory.TypeValueUse))
	}
	if ap.Slot != 5 {
		t.Errorf("apply-karma slot = %d, want 5", ap.Slot)
	}
	if ap.ScissorsKarma != 3 {
		t.Errorf("apply-karma scissorsKarma = %d, want 3 (the scissors' own cash data karma)", ap.ScissorsKarma)
	}
}
