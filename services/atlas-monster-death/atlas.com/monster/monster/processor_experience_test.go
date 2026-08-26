package monster

import (
	"atlas-monster-death/character"
	charactermock "atlas-monster-death/character/mock"
	mapmock "atlas-monster-death/map/mock"
	"atlas-monster-death/monster/information"
	informationmock "atlas-monster-death/monster/information/mock"
	"atlas-monster-death/party"
	partymock "atlas-monster-death/party/mock"
	"atlas-monster-death/rates"
	ratesmock "atlas-monster-death/rates/mock"
	"atlas-monster-death/system_message"
	systemmessagemock "atlas-monster-death/system_message/mock"
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// awardCall records one invocation of character.Processor.AwardExperience.
// The mock's "party" parameter is the party-bonus exp amount, not a partyId
// -- see character/producer.go's ExperienceDistributionTypeParty.
type awardCall struct {
	characterId uint32
	white       bool
	amount      uint32
	party       uint32
}

// hintCall records one invocation of system_message.Processor.ShowHint.
type hintCall struct {
	characterId uint32
	hint        string
	width       uint16
	height      uint16
}

func newExperienceTestSetup() (*logrus.Logger, context.Context, field.Model) {
	l := logrus.New()
	l.SetOutput(io.Discard)
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), ten)
	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000000)).SetInstance(uuid.Nil).Build()
	return l, ctx, f
}

func newTestThrottle(clock *time.Time) *system_message.Throttle {
	return system_message.NewThrottle(time.Minute, 4096, func() time.Time { return *clock })
}

func TestDistributeExperience_OnePartyLookupPerParty(t *testing.T) {
	l, ctx, f := newExperienceTestSetup()
	clock := time.Now()

	getByMemberIdCalls := 0
	pp := &partymock.ProcessorMock{
		GetByMemberIdFunc: func(memberId uint32) (party.Model, error) {
			getByMemberIdCalls++
			b := party.NewBuilder(9)
			for _, id := range []uint32{1, 2, 3, 4} {
				b.AddMember(party.NewMemberBuilder(id).SetLevel(50).Build())
			}
			return b.Build(), nil
		},
	}
	fp := &mapmock.ProcessorMock{
		CharacterIdsInFieldFunc: func(field.Model) ([]uint32, error) {
			return []uint32{1, 2, 3, 4}, nil
		},
	}
	ip := &informationmock.ProcessorMock{
		GetByIdFunc: func(uint32) (information.Model, error) {
			return information.NewBuilder().SetLevel(50).SetExperience(1000).SetName("Slime").Build()
		},
	}
	cp := &charactermock.ProcessorMock{}
	smp := &systemmessagemock.ProcessorMock{}

	p := NewProcessor(l, ctx).(*ProcessorImpl).With(
		WithPartyProcessor(pp),
		WithFieldProcessor(fp),
		WithInformationProcessor(ip),
		WithCharacterProcessor(cp),
		WithSystemMessageProcessor(smp),
		WithHintThrottle(newTestThrottle(&clock)),
	)

	des := []DamageEntryModel{
		NewDamageEntryModel(1, 100),
		NewDamageEntryModel(2, 100),
		NewDamageEntryModel(3, 100),
		NewDamageEntryModel(4, 100),
	}
	if err := p.DistributeExperience(f, 1, des); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if getByMemberIdCalls != 1 {
		t.Fatalf("expected exactly 1 party lookup, got %d", getByMemberIdCalls)
	}
}

func TestDistributeExperience_OneRateLookupPerRecipient(t *testing.T) {
	l, ctx, f := newExperienceTestSetup()
	clock := time.Now()

	getForCharacterCalls := 0
	var recordedIds []uint32
	rp := &ratesmock.ProcessorMock{
		GetForCharacterFunc: func(ch channel.Model, characterId uint32) rates.Model {
			getForCharacterCalls++
			recordedIds = append(recordedIds, characterId)
			return rates.Default()
		},
	}
	pp := &partymock.ProcessorMock{
		GetByMemberIdFunc: func(memberId uint32) (party.Model, error) {
			b := party.NewBuilder(9)
			b.AddMember(party.NewMemberBuilder(1).SetLevel(50).Build())
			b.AddMember(party.NewMemberBuilder(2).SetLevel(50).Build())
			return b.Build(), nil
		},
	}
	fp := &mapmock.ProcessorMock{
		CharacterIdsInFieldFunc: func(field.Model) ([]uint32, error) {
			return []uint32{1, 2}, nil
		},
	}
	ip := &informationmock.ProcessorMock{
		GetByIdFunc: func(uint32) (information.Model, error) {
			return information.NewBuilder().SetLevel(50).SetExperience(1000).SetName("Slime").Build()
		},
	}
	cp := &charactermock.ProcessorMock{}
	smp := &systemmessagemock.ProcessorMock{}

	p := NewProcessor(l, ctx).(*ProcessorImpl).With(
		WithRatesProcessor(rp),
		WithPartyProcessor(pp),
		WithFieldProcessor(fp),
		WithInformationProcessor(ip),
		WithCharacterProcessor(cp),
		WithSystemMessageProcessor(smp),
		WithHintThrottle(newTestThrottle(&clock)),
	)

	des := []DamageEntryModel{NewDamageEntryModel(1, 500)}
	if err := p.DistributeExperience(f, 1, des); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if getForCharacterCalls != 2 {
		t.Fatalf("expected exactly 2 rate lookups, got %d", getForCharacterCalls)
	}
	if len(recordedIds) != 2 || recordedIds[0] != 1 || recordedIds[1] != 2 {
		t.Fatalf("expected recorded characterIds [1, 2], got %v", recordedIds)
	}
}

func TestDistributeExperience_PartyLookupErrorFallsBackToSolo(t *testing.T) {
	l, ctx, f := newExperienceTestSetup()
	clock := time.Now()

	var awardCalls []awardCall
	cp := &charactermock.ProcessorMock{
		GetByIdFunc: func(characterId uint32) (character.Model, error) {
			return character.NewBuilder().SetId(characterId).SetLevel(50).Build()
		},
		AwardExperienceFunc: func(ch channel.Model, characterId uint32, white bool, amount uint32, party uint32) error {
			awardCalls = append(awardCalls, awardCall{characterId, white, amount, party})
			return nil
		},
	}
	pp := &partymock.ProcessorMock{
		GetByMemberIdFunc: func(memberId uint32) (party.Model, error) {
			return party.Model{}, errors.New("party service unavailable")
		},
	}
	fp := &mapmock.ProcessorMock{
		CharacterIdsInFieldFunc: func(field.Model) ([]uint32, error) {
			return []uint32{1}, nil
		},
	}
	ip := &informationmock.ProcessorMock{
		GetByIdFunc: func(uint32) (information.Model, error) {
			return information.NewBuilder().SetLevel(50).SetExperience(1000).SetName("Slime").Build()
		},
	}

	p := NewProcessor(l, ctx).(*ProcessorImpl).With(
		WithCharacterProcessor(cp),
		WithPartyProcessor(pp),
		WithFieldProcessor(fp),
		WithInformationProcessor(ip),
		WithHintThrottle(newTestThrottle(&clock)),
	)

	des := []DamageEntryModel{NewDamageEntryModel(1, 500)}
	if err := p.DistributeExperience(f, 1, des); err != nil {
		t.Fatalf("expected no error (panic-free degrade to solo), got %v", err)
	}

	if len(awardCalls) != 1 {
		t.Fatalf("expected exactly 1 award call, got %d", len(awardCalls))
	}
	if awardCalls[0].party != 0 {
		t.Fatalf("expected party (bonus) to be 0 for a solo recipient, got %d", awardCalls[0].party)
	}
}

func TestDistributeExperience_PartyRecipientsCarryNonZeroParty(t *testing.T) {
	l, ctx, f := newExperienceTestSetup()
	clock := time.Now()

	var awardCalls []awardCall
	cp := &charactermock.ProcessorMock{
		AwardExperienceFunc: func(ch channel.Model, characterId uint32, white bool, amount uint32, party uint32) error {
			awardCalls = append(awardCalls, awardCall{characterId, white, amount, party})
			return nil
		},
	}
	pp := &partymock.ProcessorMock{
		GetByMemberIdFunc: func(memberId uint32) (party.Model, error) {
			b := party.NewBuilder(9)
			b.AddMember(party.NewMemberBuilder(1).SetLevel(50).Build())
			b.AddMember(party.NewMemberBuilder(2).SetLevel(50).Build())
			return b.Build(), nil
		},
	}
	fp := &mapmock.ProcessorMock{
		CharacterIdsInFieldFunc: func(field.Model) ([]uint32, error) {
			return []uint32{1, 2}, nil
		},
	}
	ip := &informationmock.ProcessorMock{
		GetByIdFunc: func(uint32) (information.Model, error) {
			return information.NewBuilder().SetLevel(50).SetExperience(1000).SetName("Slime").Build()
		},
	}

	p := NewProcessor(l, ctx).(*ProcessorImpl).With(
		WithCharacterProcessor(cp),
		WithPartyProcessor(pp),
		WithFieldProcessor(fp),
		WithInformationProcessor(ip),
		WithHintThrottle(newTestThrottle(&clock)),
	)

	des := []DamageEntryModel{NewDamageEntryModel(1, 1000)}
	if err := p.DistributeExperience(f, 1, des); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(awardCalls) != 2 {
		t.Fatalf("expected exactly 2 award calls, got %d", len(awardCalls))
	}
	for _, c := range awardCalls {
		if c.party == 0 {
			t.Fatalf("expected party (bonus) > 0 for characterId %d, got 0", c.characterId)
		}
		if c.party != uint32(0.10*float64(c.amount)) {
			t.Fatalf("expected party == 0.10*amount for characterId %d; party=%d amount=%d", c.characterId, c.party, c.amount)
		}
	}
}

func TestDistributeExperience_SoloRecipientCarriesZeroParty(t *testing.T) {
	l, ctx, f := newExperienceTestSetup()
	clock := time.Now()

	var awardCalls []awardCall
	cp := &charactermock.ProcessorMock{
		GetByIdFunc: func(characterId uint32) (character.Model, error) {
			return character.NewBuilder().SetId(characterId).SetLevel(50).Build()
		},
		AwardExperienceFunc: func(ch channel.Model, characterId uint32, white bool, amount uint32, party uint32) error {
			awardCalls = append(awardCalls, awardCall{characterId, white, amount, party})
			return nil
		},
	}
	pp := &partymock.ProcessorMock{
		GetByMemberIdFunc: func(memberId uint32) (party.Model, error) {
			return party.Model{}, nil
		},
	}
	fp := &mapmock.ProcessorMock{
		CharacterIdsInFieldFunc: func(field.Model) ([]uint32, error) {
			return []uint32{1}, nil
		},
	}
	ip := &informationmock.ProcessorMock{
		GetByIdFunc: func(uint32) (information.Model, error) {
			return information.NewBuilder().SetLevel(50).SetExperience(1000).SetName("Slime").Build()
		},
	}

	p := NewProcessor(l, ctx).(*ProcessorImpl).With(
		WithCharacterProcessor(cp),
		WithPartyProcessor(pp),
		WithFieldProcessor(fp),
		WithInformationProcessor(ip),
		WithHintThrottle(newTestThrottle(&clock)),
	)

	des := []DamageEntryModel{NewDamageEntryModel(1, 1000)}
	if err := p.DistributeExperience(f, 1, des); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(awardCalls) != 1 {
		t.Fatalf("expected exactly 1 award call, got %d", len(awardCalls))
	}
	if awardCalls[0].party != 0 {
		t.Fatalf("expected party (bonus) to be 0 for solo, got %d", awardCalls[0].party)
	}
	if awardCalls[0].amount != 1000 {
		t.Fatalf("expected amount == monsterExp * expRate == 1000, got %d", awardCalls[0].amount)
	}
}

func TestDistributeExperience_ZeroDamageAwardsNothing(t *testing.T) {
	l, ctx, f := newExperienceTestSetup()
	clock := time.Now()

	awardCalls := 0
	showHintCalls := 0
	cp := &charactermock.ProcessorMock{
		AwardExperienceFunc: func(ch channel.Model, characterId uint32, white bool, amount uint32, party uint32) error {
			awardCalls++
			return nil
		},
	}
	smp := &systemmessagemock.ProcessorMock{
		ShowHintFunc: func(transactionId uuid.UUID, ch channel.Model, characterId uint32, hint string, width uint16, height uint16) error {
			showHintCalls++
			return nil
		},
	}

	p := NewProcessor(l, ctx).(*ProcessorImpl).With(
		WithCharacterProcessor(cp),
		WithSystemMessageProcessor(smp),
		WithHintThrottle(newTestThrottle(&clock)),
	)

	des := []DamageEntryModel{NewDamageEntryModel(1, 0)}
	if err := p.DistributeExperience(f, 1, des); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if awardCalls != 0 {
		t.Fatalf("expected 0 award calls, got %d", awardCalls)
	}
	if showHintCalls != 0 {
		t.Fatalf("expected 0 hint calls, got %d", showHintCalls)
	}
}

func TestDistributeExperience_OutOfFieldDamagerReceivesNothing(t *testing.T) {
	l, ctx, f := newExperienceTestSetup()
	clock := time.Now()

	getByMemberIdCalls := 0
	var awardCalls []awardCall
	cp := &charactermock.ProcessorMock{
		GetByIdFunc: func(characterId uint32) (character.Model, error) {
			return character.NewBuilder().SetId(characterId).SetLevel(50).Build()
		},
		AwardExperienceFunc: func(ch channel.Model, characterId uint32, white bool, amount uint32, party uint32) error {
			awardCalls = append(awardCalls, awardCall{characterId, white, amount, party})
			return nil
		},
	}
	pp := &partymock.ProcessorMock{
		GetByMemberIdFunc: func(memberId uint32) (party.Model, error) {
			getByMemberIdCalls++
			return party.Model{}, nil
		},
	}
	fp := &mapmock.ProcessorMock{
		CharacterIdsInFieldFunc: func(field.Model) ([]uint32, error) {
			return []uint32{1}, nil
		},
	}
	ip := &informationmock.ProcessorMock{
		GetByIdFunc: func(uint32) (information.Model, error) {
			return information.NewBuilder().SetLevel(50).SetExperience(1000).SetName("Slime").Build()
		},
	}

	p := NewProcessor(l, ctx).(*ProcessorImpl).With(
		WithCharacterProcessor(cp),
		WithPartyProcessor(pp),
		WithFieldProcessor(fp),
		WithInformationProcessor(ip),
		WithHintThrottle(newTestThrottle(&clock)),
	)

	des := []DamageEntryModel{
		NewDamageEntryModel(1, 500),
		NewDamageEntryModel(7, 300),
	}
	if err := p.DistributeExperience(f, 1, des); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(awardCalls) != 1 {
		t.Fatalf("expected exactly 1 award call, got %d", len(awardCalls))
	}
	if awardCalls[0].characterId != 1 {
		t.Fatalf("expected the only award to go to characterId 1, got %d", awardCalls[0].characterId)
	}
	if getByMemberIdCalls != 1 {
		t.Fatalf("expected exactly 1 party lookup (character 7 never resolved), got %d", getByMemberIdCalls)
	}
}

func TestDistributeExperience_ExcludedMemberGetsExactlyOneHint(t *testing.T) {
	l, ctx, f := newExperienceTestSetup()
	clock := time.Now()

	var hintCalls []hintCall
	pp := &partymock.ProcessorMock{
		GetByMemberIdFunc: func(memberId uint32) (party.Model, error) {
			b := party.NewBuilder(9)
			b.AddMember(party.NewMemberBuilder(1).SetLevel(120).Build())
			b.AddMember(party.NewMemberBuilder(2).SetLevel(70).Build())
			return b.Build(), nil
		},
	}
	fp := &mapmock.ProcessorMock{
		CharacterIdsInFieldFunc: func(field.Model) ([]uint32, error) {
			return []uint32{1, 2}, nil
		},
	}
	ip := &informationmock.ProcessorMock{
		GetByIdFunc: func(uint32) (information.Model, error) {
			return information.NewBuilder().SetLevel(125).SetExperience(1000).SetName("Zakum").Build()
		},
	}
	smp := &systemmessagemock.ProcessorMock{
		ShowHintFunc: func(transactionId uuid.UUID, ch channel.Model, characterId uint32, hint string, width uint16, height uint16) error {
			hintCalls = append(hintCalls, hintCall{characterId, hint, width, height})
			return nil
		},
	}
	cp := &charactermock.ProcessorMock{}

	p := NewProcessor(l, ctx).(*ProcessorImpl).With(
		WithPartyProcessor(pp),
		WithFieldProcessor(fp),
		WithInformationProcessor(ip),
		WithCharacterProcessor(cp),
		WithSystemMessageProcessor(smp),
		WithHintThrottle(newTestThrottle(&clock)),
	)

	des := []DamageEntryModel{NewDamageEntryModel(1, 500)}
	if err := p.DistributeExperience(f, 1, des); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(hintCalls) != 1 {
		t.Fatalf("expected exactly 1 hint call, got %d", len(hintCalls))
	}
	c := hintCalls[0]
	if c.characterId != 2 {
		t.Fatalf("expected hint to go to characterId 2, got %d", c.characterId)
	}
	if c.width != 0 || c.height != 0 {
		t.Fatalf("expected width/height 0/0, got %d/%d", c.width, c.height)
	}
	if c.hint != levelGateHintText("Zakum", 125) {
		t.Fatalf("unexpected hint text: %q", c.hint)
	}
}

func TestDistributeExperience_HintFailureDoesNotAbortAwards(t *testing.T) {
	l, ctx, f := newExperienceTestSetup()
	clock := time.Now()

	showHintCalls := 0
	var awardCalls []awardCall
	pp := &partymock.ProcessorMock{
		GetByMemberIdFunc: func(memberId uint32) (party.Model, error) {
			b := party.NewBuilder(9)
			b.AddMember(party.NewMemberBuilder(1).SetLevel(120).Build())
			b.AddMember(party.NewMemberBuilder(2).SetLevel(70).Build())
			b.AddMember(party.NewMemberBuilder(3).SetLevel(10).Build())
			b.AddMember(party.NewMemberBuilder(4).SetLevel(250).Build())
			return b.Build(), nil
		},
	}
	fp := &mapmock.ProcessorMock{
		CharacterIdsInFieldFunc: func(field.Model) ([]uint32, error) {
			return []uint32{1, 2, 3, 4}, nil
		},
	}
	ip := &informationmock.ProcessorMock{
		GetByIdFunc: func(uint32) (information.Model, error) {
			return information.NewBuilder().SetLevel(125).SetExperience(1000).SetName("Zakum").Build()
		},
	}
	cp := &charactermock.ProcessorMock{
		AwardExperienceFunc: func(ch channel.Model, characterId uint32, white bool, amount uint32, party uint32) error {
			awardCalls = append(awardCalls, awardCall{characterId, white, amount, party})
			return nil
		},
	}
	smp := &systemmessagemock.ProcessorMock{
		ShowHintFunc: func(transactionId uuid.UUID, ch channel.Model, characterId uint32, hint string, width uint16, height uint16) error {
			showHintCalls++
			if showHintCalls == 1 {
				return errors.New("publish failed")
			}
			return nil
		},
	}

	p := NewProcessor(l, ctx).(*ProcessorImpl).With(
		WithPartyProcessor(pp),
		WithFieldProcessor(fp),
		WithInformationProcessor(ip),
		WithCharacterProcessor(cp),
		WithSystemMessageProcessor(smp),
		WithHintThrottle(newTestThrottle(&clock)),
	)

	des := []DamageEntryModel{NewDamageEntryModel(1, 500)}
	if err := p.DistributeExperience(f, 1, des); err != nil {
		t.Fatalf("expected no error even though a hint publish failed, got %v", err)
	}

	if showHintCalls != 3 {
		t.Fatalf("expected exactly 3 hint attempts, got %d", showHintCalls)
	}
	if len(awardCalls) != 1 {
		t.Fatalf("expected the in-range contributor to still be awarded, got %d awards", len(awardCalls))
	}
	if awardCalls[0].characterId != 1 {
		t.Fatalf("expected award to go to characterId 1, got %d", awardCalls[0].characterId)
	}
}

func TestDistributeExperience_AwardFailureDoesNotAbortOthers(t *testing.T) {
	l, ctx, f := newExperienceTestSetup()
	clock := time.Now()

	var awardCalls []awardCall
	pp := &partymock.ProcessorMock{
		GetByMemberIdFunc: func(memberId uint32) (party.Model, error) {
			b := party.NewBuilder(9)
			b.AddMember(party.NewMemberBuilder(1).SetLevel(50).Build())
			b.AddMember(party.NewMemberBuilder(2).SetLevel(50).Build())
			b.AddMember(party.NewMemberBuilder(3).SetLevel(50).Build())
			return b.Build(), nil
		},
	}
	fp := &mapmock.ProcessorMock{
		CharacterIdsInFieldFunc: func(field.Model) ([]uint32, error) {
			return []uint32{1, 2, 3}, nil
		},
	}
	ip := &informationmock.ProcessorMock{
		GetByIdFunc: func(uint32) (information.Model, error) {
			return information.NewBuilder().SetLevel(50).SetExperience(1000).SetName("Slime").Build()
		},
	}
	cp := &charactermock.ProcessorMock{
		AwardExperienceFunc: func(ch channel.Model, characterId uint32, white bool, amount uint32, party uint32) error {
			awardCalls = append(awardCalls, awardCall{characterId, white, amount, party})
			if characterId == 1 {
				return errors.New("award publish failed")
			}
			return nil
		},
	}

	p := NewProcessor(l, ctx).(*ProcessorImpl).With(
		WithPartyProcessor(pp),
		WithFieldProcessor(fp),
		WithInformationProcessor(ip),
		WithCharacterProcessor(cp),
		WithHintThrottle(newTestThrottle(&clock)),
	)

	des := []DamageEntryModel{NewDamageEntryModel(1, 1000)}
	if err := p.DistributeExperience(f, 1, des); err != nil {
		t.Fatalf("expected no error even though one award failed, got %v", err)
	}

	if len(awardCalls) != 3 {
		t.Fatalf("expected exactly 3 award attempts, got %d", len(awardCalls))
	}
}

func TestDistributeExperience_HintIsThrottledAcrossKills(t *testing.T) {
	l, ctx, f := newExperienceTestSetup()
	clock := time.Now()

	showHintCalls := 0
	pp := &partymock.ProcessorMock{
		GetByMemberIdFunc: func(memberId uint32) (party.Model, error) {
			b := party.NewBuilder(9)
			b.AddMember(party.NewMemberBuilder(1).SetLevel(120).Build())
			b.AddMember(party.NewMemberBuilder(2).SetLevel(70).Build())
			return b.Build(), nil
		},
	}
	fp := &mapmock.ProcessorMock{
		CharacterIdsInFieldFunc: func(field.Model) ([]uint32, error) {
			return []uint32{1, 2}, nil
		},
	}
	ip := &informationmock.ProcessorMock{
		GetByIdFunc: func(uint32) (information.Model, error) {
			return information.NewBuilder().SetLevel(125).SetExperience(1000).SetName("Zakum").Build()
		},
	}
	smp := &systemmessagemock.ProcessorMock{
		ShowHintFunc: func(transactionId uuid.UUID, ch channel.Model, characterId uint32, hint string, width uint16, height uint16) error {
			showHintCalls++
			return nil
		},
	}
	cp := &charactermock.ProcessorMock{}

	p := NewProcessor(l, ctx).(*ProcessorImpl).With(
		WithPartyProcessor(pp),
		WithFieldProcessor(fp),
		WithInformationProcessor(ip),
		WithCharacterProcessor(cp),
		WithSystemMessageProcessor(smp),
		WithHintThrottle(newTestThrottle(&clock)),
	)

	des := []DamageEntryModel{NewDamageEntryModel(1, 500)}

	if err := p.DistributeExperience(f, 1, des); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	clock = clock.Add(30 * time.Second)
	if err := p.DistributeExperience(f, 1, des); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if showHintCalls != 1 {
		t.Fatalf("expected exactly 1 hint call within the throttle window, got %d", showHintCalls)
	}

	clock = clock.Add(61 * time.Second)
	if err := p.DistributeExperience(f, 1, des); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if showHintCalls != 2 {
		t.Fatalf("expected a second hint call once the throttle window elapsed, got %d", showHintCalls)
	}
}

func TestDistributeExperience_GateDisabledEmitsNoHint(t *testing.T) {
	l, ctx, f := newExperienceTestSetup()
	clock := time.Now()

	showHintCalls := 0
	var awardCalls []awardCall
	pp := &partymock.ProcessorMock{
		GetByMemberIdFunc: func(memberId uint32) (party.Model, error) {
			b := party.NewBuilder(9)
			b.AddMember(party.NewMemberBuilder(1).SetLevel(120).Build())
			b.AddMember(party.NewMemberBuilder(2).SetLevel(70).Build())
			return b.Build(), nil
		},
	}
	fp := &mapmock.ProcessorMock{
		CharacterIdsInFieldFunc: func(field.Model) ([]uint32, error) {
			return []uint32{1, 2}, nil
		},
	}
	ip := &informationmock.ProcessorMock{
		GetByIdFunc: func(uint32) (information.Model, error) {
			return information.NewBuilder().SetLevel(125).SetExperience(1000).SetName("Zakum").Build()
		},
	}
	cp := &charactermock.ProcessorMock{
		AwardExperienceFunc: func(ch channel.Model, characterId uint32, white bool, amount uint32, party uint32) error {
			awardCalls = append(awardCalls, awardCall{characterId, white, amount, party})
			return nil
		},
	}
	smp := &systemmessagemock.ProcessorMock{
		ShowHintFunc: func(transactionId uuid.UUID, ch channel.Model, characterId uint32, hint string, width uint16, height uint16) error {
			showHintCalls++
			return nil
		},
	}

	cfg := DefaultExperienceConfig()
	cfg.EnforceMobLevelRange = false

	p := NewProcessor(l, ctx).(*ProcessorImpl).With(
		WithPartyProcessor(pp),
		WithFieldProcessor(fp),
		WithInformationProcessor(ip),
		WithCharacterProcessor(cp),
		WithSystemMessageProcessor(smp),
		WithExperienceConfig(cfg),
		WithHintThrottle(newTestThrottle(&clock)),
	)

	des := []DamageEntryModel{NewDamageEntryModel(1, 500)}
	if err := p.DistributeExperience(f, 1, des); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if showHintCalls != 0 {
		t.Fatalf("expected 0 hint calls with the gate disabled, got %d", showHintCalls)
	}
	if len(awardCalls) != 2 {
		t.Fatalf("expected both members awarded with the gate disabled, got %d", len(awardCalls))
	}
}

func TestDistributeExperience_InformationErrorReturnsError(t *testing.T) {
	l, ctx, f := newExperienceTestSetup()
	clock := time.Now()

	wantErr := errors.New("monster information unavailable")
	awardCalls := 0
	cp := &charactermock.ProcessorMock{
		AwardExperienceFunc: func(ch channel.Model, characterId uint32, white bool, amount uint32, party uint32) error {
			awardCalls++
			return nil
		},
	}
	fp := &mapmock.ProcessorMock{
		CharacterIdsInFieldFunc: func(field.Model) ([]uint32, error) {
			return []uint32{1}, nil
		},
	}
	ip := &informationmock.ProcessorMock{
		GetByIdFunc: func(uint32) (information.Model, error) {
			return information.Model{}, wantErr
		},
	}

	p := NewProcessor(l, ctx).(*ProcessorImpl).With(
		WithCharacterProcessor(cp),
		WithFieldProcessor(fp),
		WithInformationProcessor(ip),
		WithHintThrottle(newTestThrottle(&clock)),
	)

	des := []DamageEntryModel{NewDamageEntryModel(1, 500)}
	err := p.DistributeExperience(f, 1, des)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected the information error to be returned, got %v", err)
	}
	if awardCalls != 0 {
		t.Fatalf("expected 0 award calls, got %d", awardCalls)
	}
}

func TestDistributeExperience_FieldErrorReturnsError(t *testing.T) {
	l, ctx, f := newExperienceTestSetup()
	clock := time.Now()

	wantErr := errors.New("field lookup unavailable")
	awardCalls := 0
	cp := &charactermock.ProcessorMock{
		AwardExperienceFunc: func(ch channel.Model, characterId uint32, white bool, amount uint32, party uint32) error {
			awardCalls++
			return nil
		},
	}
	fp := &mapmock.ProcessorMock{
		CharacterIdsInFieldFunc: func(field.Model) ([]uint32, error) {
			return nil, wantErr
		},
	}
	ip := &informationmock.ProcessorMock{
		GetByIdFunc: func(uint32) (information.Model, error) {
			return information.NewBuilder().SetLevel(50).SetExperience(1000).SetName("Slime").Build()
		},
	}

	p := NewProcessor(l, ctx).(*ProcessorImpl).With(
		WithCharacterProcessor(cp),
		WithFieldProcessor(fp),
		WithInformationProcessor(ip),
		WithHintThrottle(newTestThrottle(&clock)),
	)

	des := []DamageEntryModel{NewDamageEntryModel(1, 500)}
	err := p.DistributeExperience(f, 1, des)
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected the field error to be returned, got %v", err)
	}
	if awardCalls != 0 {
		t.Fatalf("expected 0 award calls, got %d", awardCalls)
	}
}

func TestDistributeExperience_AwardOrderIsAscendingCharacterId(t *testing.T) {
	l, ctx, f := newExperienceTestSetup()
	clock := time.Now()

	var order []uint32
	pp := &partymock.ProcessorMock{
		GetByMemberIdFunc: func(memberId uint32) (party.Model, error) {
			b := party.NewBuilder(9)
			for _, id := range []uint32{3, 1, 2} {
				b.AddMember(party.NewMemberBuilder(id).SetLevel(50).Build())
			}
			return b.Build(), nil
		},
	}
	fp := &mapmock.ProcessorMock{
		CharacterIdsInFieldFunc: func(field.Model) ([]uint32, error) {
			return []uint32{1, 2, 3}, nil
		},
	}
	ip := &informationmock.ProcessorMock{
		GetByIdFunc: func(uint32) (information.Model, error) {
			return information.NewBuilder().SetLevel(50).SetExperience(1000).SetName("Slime").Build()
		},
	}
	cp := &charactermock.ProcessorMock{
		AwardExperienceFunc: func(ch channel.Model, characterId uint32, white bool, amount uint32, party uint32) error {
			order = append(order, characterId)
			return nil
		},
	}

	p := NewProcessor(l, ctx).(*ProcessorImpl).With(
		WithPartyProcessor(pp),
		WithFieldProcessor(fp),
		WithInformationProcessor(ip),
		WithCharacterProcessor(cp),
		WithHintThrottle(newTestThrottle(&clock)),
	)

	des := []DamageEntryModel{NewDamageEntryModel(1, 1000)}
	if err := p.DistributeExperience(f, 1, des); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	want := []uint32{1, 2, 3}
	if len(order) != len(want) {
		t.Fatalf("expected %d awards, got %d (%v)", len(want), len(order), order)
	}
	for i, id := range want {
		if order[i] != id {
			t.Fatalf("expected award order %v, got %v", want, order)
		}
	}
}
