package buff

import (
	"io"
	"testing"
	"time"

	"atlas-rates/character"
	"atlas-rates/kafka/message/buff"
	"atlas-rates/rate"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/sirupsen/logrus"
)

// fakeProcessor captures calls to the rate-relevant Processor methods. All
// other interface methods are stubs that satisfy the interface but are unused
// by the consumer.
type fakeProcessor struct {
	character.Processor // embed so unused methods compile (will panic if called)
	addBuffFactorCalls     []addBuffFactorCall
	removeAllBuffFactorIds []removeAllBuffFactorCall
}

type addBuffFactorCall struct {
	channel      channel.Model
	characterId  uint32
	buffSourceId int32
	rateType     rate.Type
	multiplier   float64
}

type removeAllBuffFactorCall struct {
	characterId  uint32
	buffSourceId int32
}

func (f *fakeProcessor) AddBuffFactor(ch channel.Model, characterId uint32, buffSourceId int32, rateType rate.Type, multiplier float64) error {
	f.addBuffFactorCalls = append(f.addBuffFactorCalls, addBuffFactorCall{
		channel:      ch,
		characterId:  characterId,
		buffSourceId: buffSourceId,
		rateType:     rateType,
		multiplier:   multiplier,
	})
	return nil
}

func (f *fakeProcessor) RemoveAllBuffFactors(characterId uint32, buffSourceId int32) error {
	f.removeAllBuffFactorIds = append(f.removeAllBuffFactorIds, removeAllBuffFactorCall{
		characterId:  characterId,
		buffSourceId: buffSourceId,
	})
	return nil
}

func discardLogger() logrus.FieldLogger {
	l := logrus.New()
	l.Out = io.Discard
	return l
}

func appliedEvent(changes []buff.StatChange) buff.StatusEvent[buff.AppliedStatusEventBody] {
	return buff.StatusEvent[buff.AppliedStatusEventBody]{
		WorldId:     world.Id(0),
		ChannelId:   channel.Id(0),
		CharacterId: 1234,
		Type:        buff.EventStatusTypeBuffApplied,
		Body: buff.AppliedStatusEventBody{
			FromId:    99,
			SourceId:  4120002, // Stirge curse skill id (illustrative)
			Duration:  30000,
			Changes:   changes,
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(30 * time.Second),
		},
	}
}

func expiredEvent(sourceId int32) buff.StatusEvent[buff.ExpiredStatusEventBody] {
	return buff.StatusEvent[buff.ExpiredStatusEventBody]{
		WorldId:     world.Id(0),
		ChannelId:   channel.Id(0),
		CharacterId: 1234,
		Type:        buff.EventStatusTypeBuffExpired,
		Body: buff.ExpiredStatusEventBody{
			SourceId:  sourceId,
			Duration:  30000,
			Changes:   nil,
			CreatedAt: time.Now(),
			ExpiresAt: time.Now(),
		},
	}
}

func TestHandleBuffApplied_CurseRegistersHalfExpFactor(t *testing.T) {
	p := &fakeProcessor{}
	e := appliedEvent([]buff.StatChange{{Type: buff.StatTypeCurse, Amount: 0}})

	handleBuffAppliedFor(p, discardLogger(), e)

	if len(p.addBuffFactorCalls) != 1 {
		t.Fatalf("AddBuffFactor calls = %d, want 1", len(p.addBuffFactorCalls))
	}
	c := p.addBuffFactorCalls[0]
	if c.characterId != 1234 {
		t.Errorf("characterId = %d, want 1234", c.characterId)
	}
	if c.buffSourceId != e.Body.SourceId {
		t.Errorf("buffSourceId = %d, want %d", c.buffSourceId, e.Body.SourceId)
	}
	if c.rateType != rate.Type("exp") {
		t.Errorf("rateType = %q, want \"exp\"", c.rateType)
	}
	if c.multiplier != 0.5 {
		t.Errorf("multiplier = %v, want 0.5", c.multiplier)
	}
}

func TestHandleBuffApplied_HolySymbolStillProducesAdditive(t *testing.T) {
	p := &fakeProcessor{}
	e := appliedEvent([]buff.StatChange{{Type: buff.StatTypeHolySymbol, Amount: 50}})

	handleBuffAppliedFor(p, discardLogger(), e)

	if len(p.addBuffFactorCalls) != 1 {
		t.Fatalf("AddBuffFactor calls = %d, want 1", len(p.addBuffFactorCalls))
	}
	c := p.addBuffFactorCalls[0]
	if c.rateType != rate.Type("exp") {
		t.Errorf("rateType = %q, want \"exp\"", c.rateType)
	}
	if c.multiplier != 1.5 {
		t.Errorf("multiplier = %v, want 1.5", c.multiplier)
	}
}

func TestHandleBuffApplied_CurseAndHolySymbolBothRegister(t *testing.T) {
	p := &fakeProcessor{}
	e := appliedEvent([]buff.StatChange{
		{Type: buff.StatTypeCurse, Amount: 0},
		{Type: buff.StatTypeHolySymbol, Amount: 50},
	})

	handleBuffAppliedFor(p, discardLogger(), e)

	if len(p.addBuffFactorCalls) != 2 {
		t.Fatalf("AddBuffFactor calls = %d, want 2", len(p.addBuffFactorCalls))
	}
	multipliers := []float64{p.addBuffFactorCalls[0].multiplier, p.addBuffFactorCalls[1].multiplier}
	hasCurse := false
	hasHoly := false
	for _, m := range multipliers {
		if m == 0.5 {
			hasCurse = true
		}
		if m == 1.5 {
			hasHoly = true
		}
	}
	if !hasCurse || !hasHoly {
		t.Errorf("missing factor: multipliers = %v, want both 0.5 and 1.5", multipliers)
	}
	for _, c := range p.addBuffFactorCalls {
		if c.rateType != rate.Type("exp") {
			t.Errorf("unexpected rateType %q on factor with multiplier %v", c.rateType, c.multiplier)
		}
	}
}

func TestHandleBuffApplied_NonRateStatIsNoOp(t *testing.T) {
	p := &fakeProcessor{}
	e := appliedEvent([]buff.StatChange{{Type: "WEAPON_ATTACK", Amount: 30}})

	handleBuffAppliedFor(p, discardLogger(), e)

	if len(p.addBuffFactorCalls) != 0 {
		t.Errorf("AddBuffFactor calls = %d, want 0 (non-rate stat)", len(p.addBuffFactorCalls))
	}
}

func TestHandleBuffApplied_WrongTypeIsNoOp(t *testing.T) {
	p := &fakeProcessor{}
	e := appliedEvent([]buff.StatChange{{Type: buff.StatTypeCurse, Amount: 0}})
	e.Type = "EXPIRED" // wrong type — guard should skip

	handleBuffAppliedFor(p, discardLogger(), e)

	if len(p.addBuffFactorCalls) != 0 {
		t.Errorf("AddBuffFactor calls = %d, want 0 (type guard)", len(p.addBuffFactorCalls))
	}
}

func TestHandleBuffApplied_CurseNeverTouchesMesoOrItemOrQuest(t *testing.T) {
	// Acceptance criterion #4: CURSE only registers against exp. Send CURSE
	// alongside an unrelated non-rate stat and verify CURSE still produces
	// exactly one factor, against "exp", with no incidental meso/item/quest
	// emissions.
	p := &fakeProcessor{}
	e := appliedEvent([]buff.StatChange{
		{Type: buff.StatTypeCurse, Amount: 0},
		{Type: "WEAPON_ATTACK", Amount: 30},
	})

	handleBuffAppliedFor(p, discardLogger(), e)

	if len(p.addBuffFactorCalls) != 1 {
		t.Fatalf("AddBuffFactor calls = %d, want 1 (only CURSE; WEAPON_ATTACK is non-rate)", len(p.addBuffFactorCalls))
	}
	c := p.addBuffFactorCalls[0]
	if c.rateType != rate.Type("exp") {
		t.Errorf("rateType = %q, want \"exp\"", c.rateType)
	}
	if c.multiplier != 0.5 {
		t.Errorf("multiplier = %v, want 0.5", c.multiplier)
	}
}

func TestHandleBuffExpired_CallsRemoveAllBuffFactors(t *testing.T) {
	p := &fakeProcessor{}
	e := expiredEvent(4120002)

	handleBuffExpiredFor(p, discardLogger(), e)

	if len(p.removeAllBuffFactorIds) != 1 {
		t.Fatalf("RemoveAllBuffFactors calls = %d, want 1", len(p.removeAllBuffFactorIds))
	}
	c := p.removeAllBuffFactorIds[0]
	if c.characterId != 1234 {
		t.Errorf("characterId = %d, want 1234", c.characterId)
	}
	if c.buffSourceId != 4120002 {
		t.Errorf("buffSourceId = %d, want 4120002", c.buffSourceId)
	}
}

func TestHandleBuffExpired_WrongTypeIsNoOp(t *testing.T) {
	p := &fakeProcessor{}
	e := expiredEvent(4120002)
	e.Type = "APPLIED" // wrong type

	handleBuffExpiredFor(p, discardLogger(), e)

	if len(p.removeAllBuffFactorIds) != 0 {
		t.Errorf("RemoveAllBuffFactors calls = %d, want 0 (type guard)", len(p.removeAllBuffFactorIds))
	}
}
