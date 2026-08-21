package heal

import (
	"atlas-channel/character"
	"atlas-channel/data/skill/effect"
	"atlas-channel/effective_stats"
	"atlas-channel/socket/writer"
	"context"
	"io"
	"testing"

	character2 "atlas-channel/kafka/message/character"

	channelhandler "atlas-channel/skill/handler"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	packetmodel "github.com/Chronicle20/atlas/libs/atlas-packet/model"
)

func tl() logrus.FieldLogger {
	l := logrus.New()
	l.SetOutput(io.Discard)
	return l
}

func castEffect(t *testing.T) effect.Model {
	t.Helper()
	e, err := effect.Extract(effect.RestModel{Hp: 300})
	if err != nil {
		t.Fatalf("effect.Extract: %v", err)
	}
	return e
}

func castInfo(mobIds []uint32) packetmodel.SkillUsageInfo {
	return packetmodel.NewSkillUsageInfoBuilder().
		SetSkillId(2301002).
		SetSkillLevel(1).
		SetAffectedMobIds(mobIds).
		Build()
}

func castField() field.Model {
	return field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(100000000)).Build()
}

// changeHpCall records one changeHpFunc invocation.
type changeHpCall struct {
	characterId uint32
	amount      int16
}

// installHealSeams saves the originals of every seam Apply uses and
// restores them in t.Cleanup, per dispel_test.go:54-59. It installs the
// shared fixtures described in the task-4 brief and returns the recorders
// callers can inspect after Apply runs.
func installHealSeams(t *testing.T) (
	changeHpCalls *[]changeHpCall,
	awardExperienceCalls *[]struct {
		characterId   uint32
		distributions []character2.ExperienceDistributions
	},
	announceCastCalls *int,
) {
	t.Helper()

	origLoadCaster := loadCasterFunc
	origEffectiveStats := effectiveStatsFunc
	origSelectPartyMembers := selectPartyMembersFunc
	origVariance := varianceFunc
	origChangeHp := changeHpFunc
	origAwardExperience := awardExperienceFunc
	origAnnounceCast := announceCastFunc
	t.Cleanup(func() {
		loadCasterFunc = origLoadCaster
		effectiveStatsFunc = origEffectiveStats
		selectPartyMembersFunc = origSelectPartyMembers
		varianceFunc = origVariance
		changeHpFunc = origChangeHp
		awardExperienceFunc = origAwardExperience
		announceCastFunc = origAnnounceCast
	})

	loadCasterFunc = func(cp character.Processor, characterId uint32) (character.Model, error) {
		return character.NewModelBuilder().
			SetId(1).
			SetLevel(30).
			SetIntelligence(100).
			SetHp(500).
			SetMaxHp(1000).
			SetX(0).
			SetY(0).
			MustBuild(), nil
	}
	effectiveStatsFunc = func(esp effective_stats.Processor, worldId world.Id, channelId channel.Id, characterId uint32) (effective_stats.RestModel, error) {
		return effective_stats.RestModel{Intelligence: 100, MagicAttack: 0, MaxHp: 1000}, nil
	}
	selectPartyMembersFunc = func(l logrus.FieldLogger, ctx context.Context, f field.Model, casterId uint32, casterX, casterY int16, e effect.Model, memberBitmap byte) []channelhandler.PartyRecipient {
		return []channelhandler.PartyRecipient{
			channelhandler.NewPartyRecipientBuilder().SetId(2).SetHp(50).SetMaxHp(1000).Build(),
			channelhandler.NewPartyRecipientBuilder().SetId(3).SetHp(60).SetMaxHp(1000).Build(),
			channelhandler.NewPartyRecipientBuilder().SetId(4).SetHp(0).SetMaxHp(1000).Build(),
		}
	}
	varianceFunc = func() float64 { return 1.0 }

	var hpCalls []changeHpCall
	changeHpFunc = func(cp character.Processor, f field.Model, characterId uint32, amount int16) error {
		hpCalls = append(hpCalls, changeHpCall{characterId: characterId, amount: amount})
		return nil
	}

	var xpCalls []struct {
		characterId   uint32
		distributions []character2.ExperienceDistributions
	}
	awardExperienceFunc = func(cp character.Processor, f field.Model, characterId uint32, distributions []character2.ExperienceDistributions, showEffect bool) error {
		xpCalls = append(xpCalls, struct {
			characterId   uint32
			distributions []character2.ExperienceDistributions
		}{characterId: characterId, distributions: distributions})
		return nil
	}

	var announceCalls int
	announceCastFunc = func(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, f field.Model, characterId uint32, casterLevel byte, skillId uint32, skillLevel byte) {
		announceCalls++
	}

	return &hpCalls, &xpCalls, &announceCalls
}

func TestApply_NotZombified_HealsEveryRecipient(t *testing.T) {
	hpCalls, xpCalls, announceCalls := installHealSeams(t)

	origZombified := casterZombifiedFunc
	t.Cleanup(func() { casterZombifiedFunc = origZombified })
	casterZombifiedFunc = func(l logrus.FieldLogger, ctx context.Context, characterId uint32) bool { return false }

	err := Apply(tl())(context.Background())(nil, castField(), 1, castInfo(nil), castEffect(t))
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}

	wantHp := []changeHpCall{
		{characterId: 1, amount: 60},
		{characterId: 2, amount: 60},
		{characterId: 3, amount: 60},
		{characterId: 4, amount: 60},
	}
	if len(*hpCalls) != len(wantHp) {
		t.Fatalf("changeHpFunc calls = %+v, want %+v", *hpCalls, wantHp)
	}
	for i, want := range wantHp {
		if (*hpCalls)[i] != want {
			t.Errorf("changeHpFunc call %d = %+v, want %+v", i, (*hpCalls)[i], want)
		}
	}

	if len(*xpCalls) != 1 {
		t.Fatalf("awardExperienceFunc calls = %d, want 1", len(*xpCalls))
	}
	xp := (*xpCalls)[0]
	if len(xp.distributions) != 1 {
		t.Fatalf("awardExperienceFunc distributions = %+v, want 1 entry", xp.distributions)
	}
	if xp.distributions[0].ExperienceType != character2.ExperienceDistributionTypeWhite {
		t.Errorf("ExperienceType = %v, want ExperienceDistributionTypeWhite", xp.distributions[0].ExperienceType)
	}
	if xp.distributions[0].Amount != 24 {
		t.Errorf("Amount = %d, want 24", xp.distributions[0].Amount)
	}

	if *announceCalls != 1 {
		t.Errorf("announceCastFunc calls = %d, want 1", *announceCalls)
	}
}

func TestApply_ZombifiedCaster_DamagesEveryRecipient(t *testing.T) {
	hpCalls, xpCalls, announceCalls := installHealSeams(t)

	origZombified := casterZombifiedFunc
	t.Cleanup(func() { casterZombifiedFunc = origZombified })
	casterZombifiedFunc = func(l logrus.FieldLogger, ctx context.Context, characterId uint32) bool { return true }

	err := Apply(tl())(context.Background())(nil, castField(), 1, castInfo(nil), castEffect(t))
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}

	wantHp := []changeHpCall{
		{characterId: 1, amount: -60},
		{characterId: 2, amount: -50},
		{characterId: 3, amount: -60},
	}
	if len(*hpCalls) != len(wantHp) {
		t.Fatalf("changeHpFunc calls = %+v, want %+v", *hpCalls, wantHp)
	}
	for i, want := range wantHp {
		if (*hpCalls)[i] != want {
			t.Errorf("changeHpFunc call %d = %+v, want %+v", i, (*hpCalls)[i], want)
		}
	}

	if len(*xpCalls) != 0 {
		t.Errorf("awardExperienceFunc calls = %d, want 0", len(*xpCalls))
	}

	if *announceCalls != 1 {
		t.Errorf("announceCastFunc calls = %d, want 1", *announceCalls)
	}
}

func TestApply_ZombifyReadIsCasterOnlyAndIssuedOnce(t *testing.T) {
	installHealSeams(t)

	origZombified := casterZombifiedFunc
	t.Cleanup(func() { casterZombifiedFunc = origZombified })
	var recorded []uint32
	casterZombifiedFunc = func(l logrus.FieldLogger, ctx context.Context, characterId uint32) bool {
		recorded = append(recorded, characterId)
		return false
	}

	err := Apply(tl())(context.Background())(nil, castField(), 1, castInfo(nil), castEffect(t))
	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}

	want := []uint32{1}
	if len(recorded) != len(want) {
		t.Fatalf("casterZombifiedFunc calls = %v, want %v", recorded, want)
	}
	for i, w := range want {
		if recorded[i] != w {
			t.Errorf("casterZombifiedFunc call %d = %d, want %d", i, recorded[i], w)
		}
	}
}
