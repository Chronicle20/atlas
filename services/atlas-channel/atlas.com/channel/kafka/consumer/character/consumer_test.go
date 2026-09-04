package character

import (
	"atlas-channel/character/snapshot"
	"atlas-channel/server"
	model2 "atlas-channel/socket/model"
	"context"
	"testing"

	character3 "atlas-channel/character"

	character2 "atlas-channel/kafka/message/character"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/stat"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func newTestTenant(t *testing.T) tenant.Model {
	t.Helper()
	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant: %v", err)
	}
	return tm
}

func newTestServer(t *testing.T, tm tenant.Model) server.Model {
	t.Helper()
	ch := channel.NewModel(0, 1)
	return server.NewProcessor(logrus.New(), context.Background()).Register(tm, ch, "127.0.0.1", 8484)
}

// seedSnapshotCore creates a snapshot entry and validates its core so
// in-place event updates have a base to apply to.
func seedSnapshotCore(t *testing.T, tm tenant.Model, characterId uint32) {
	t.Helper()
	v := snapshot.GetRegistry().View(tm, characterId)
	core := character3.NewBuilder().
		SetId(characterId).SetLevel(30).SetMp(500).SetMaxMp(800).
		MustBuild()
	if !snapshot.GetRegistry().BackfillCore(tm, characterId, core, v.CoreGen) {
		t.Fatalf("seed backfill rejected")
	}
}

// TestSnapshotHandlers is table-driven over the character snapshot
// consumer's distinct scenarios (DOM-20). Each case's body is preserved
// verbatim from the original single-purpose test function, including its
// exact assertions, so no scenario's checking strength changed in the
// conversion.
func TestSnapshotHandlers(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T)
	}{
		{
			name: "StatChanged_RichValuesApplyInPlace",
			run: func(t *testing.T) {
				tm := newTestTenant(t)
				ctx := tenant.WithContext(context.Background(), tm)
				sc := newTestServer(t, tm)
				seedSnapshotCore(t, tm, 41)

				e := character2.StatusEvent[character2.StatusEventStatChangedBody]{
					WorldId: 0, CharacterId: 41, Type: character2.StatusEventTypeStatChanged,
					Body: character2.StatusEventStatChangedBody{
						ChannelId: 1,
						Updates:   []stat.Type{stat.TypeMp},
						Values:    map[string]interface{}{"mp": float64(463)},
					},
				}
				handleSnapshotStatChanged(sc, nil)(logrus.New(), ctx, e)

				v := snapshot.GetRegistry().View(tm, 41)
				if !v.CoreValid || v.Core.Mp() != 463 {
					t.Fatalf("rich STAT_CHANGED must apply in place: valid=%v mp=%d", v.CoreValid, v.Core.Mp())
				}
			},
		},
		{
			name: "StatChanged_NilValuesInvalidates",
			run: func(t *testing.T) {
				tm := newTestTenant(t)
				ctx := tenant.WithContext(context.Background(), tm)
				sc := newTestServer(t, tm)
				seedSnapshotCore(t, tm, 42)

				e := character2.StatusEvent[character2.StatusEventStatChangedBody]{
					WorldId: 0, CharacterId: 42, Type: character2.StatusEventTypeStatChanged,
					Body: character2.StatusEventStatChangedBody{ChannelId: 1, Updates: []stat.Type{stat.TypeMp}},
				}
				handleSnapshotStatChanged(sc, nil)(logrus.New(), ctx, e)

				if v := snapshot.GetRegistry().View(tm, 42); v.CoreValid {
					t.Fatalf("nil-Values STAT_CHANGED must invalidate (rollout safety)")
				}
			},
		},
		{
			name: "LevelAndExperience_ApplyAbsolute",
			run: func(t *testing.T) {
				tm := newTestTenant(t)
				ctx := tenant.WithContext(context.Background(), tm)
				sc := newTestServer(t, tm)
				seedSnapshotCore(t, tm, 43)

				le := character2.StatusEvent[character2.LevelChangedStatusEventBody]{
					WorldId: 0, CharacterId: 43, Type: character2.StatusEventTypeLevelChanged,
					Body: character2.LevelChangedStatusEventBody{ChannelId: 1, Amount: 1, Current: 31},
				}
				handleSnapshotLevelChanged(sc, nil)(logrus.New(), ctx, le)

				ee := character2.StatusEvent[character2.ExperienceChangedStatusEventBody]{
					WorldId: 0, CharacterId: 43, Type: character2.StatusEventTypeExperienceChanged,
					Body: character2.ExperienceChangedStatusEventBody{ChannelId: 1, Current: 999},
				}
				handleSnapshotExperienceChanged(sc, nil)(logrus.New(), ctx, ee)

				v := snapshot.GetRegistry().View(tm, 43)
				if v.Core.Level() != 31 || v.Core.Experience() != 999 {
					t.Fatalf("level/exp not applied: %d/%d", v.Core.Level(), v.Core.Experience())
				}
			},
		},
		{
			name: "MapChanged_TargetPositionSetsOverlay",
			run: func(t *testing.T) {
				tm := newTestTenant(t)
				ctx := tenant.WithContext(context.Background(), tm)
				sc := newTestServer(t, tm)
				seedSnapshotCore(t, tm, 44)

				e := character2.StatusEvent[character2.StatusEventMapChangedBody]{
					WorldId: 0, CharacterId: 44, Type: character2.StatusEventTypeMapChanged,
					Body: character2.StatusEventMapChangedBody{
						ChannelId: 1, TargetMapId: 100000000,
						UseTargetPosition: true, TargetX: 77, TargetY: -88,
					},
				}
				handleSnapshotMapChanged(sc, nil)(logrus.New(), ctx, e)

				v := snapshot.GetRegistry().View(tm, 44)
				if !v.PosValid || v.PosX != 77 || v.PosY != -88 {
					t.Fatalf("UseTargetPosition must set the overlay: %+v", v)
				}
				if !v.CoreValid {
					t.Fatalf("UseTargetPosition path must not invalidate core (overlay covers X/Y)")
				}
			},
		},
		{
			name: "MapChanged_PortalWarpInvalidatesPositionAndCore",
			run: func(t *testing.T) {
				tm := newTestTenant(t)
				ctx := tenant.WithContext(context.Background(), tm)
				sc := newTestServer(t, tm)
				seedSnapshotCore(t, tm, 45)
				snapshot.GetRegistry().SetPosition(tm, 45, 1, 2)

				e := character2.StatusEvent[character2.StatusEventMapChangedBody]{
					WorldId: 0, CharacterId: 45, Type: character2.StatusEventTypeMapChanged,
					Body: character2.StatusEventMapChangedBody{ChannelId: 1, TargetMapId: 100000000},
				}
				handleSnapshotMapChanged(sc, nil)(logrus.New(), ctx, e)

				v := snapshot.GetRegistry().View(tm, 45)
				if v.PosValid {
					t.Fatalf("portal warp must invalidate the position overlay")
				}
				if v.CoreValid {
					t.Fatalf("portal warp must invalidate core so the next read refetches fresh REST X/Y (design §10.4)")
				}
			},
		},
		{
			name: "IgnoreOtherWorlds",
			run: func(t *testing.T) {
				tm := newTestTenant(t)
				ctx := tenant.WithContext(context.Background(), tm)
				sc := newTestServer(t, tm) // world 0
				seedSnapshotCore(t, tm, 46)

				e := character2.StatusEvent[character2.StatusEventStatChangedBody]{
					WorldId: 3, CharacterId: 46, Type: character2.StatusEventTypeStatChanged,
					Body: character2.StatusEventStatChangedBody{ChannelId: 1, Updates: []stat.Type{stat.TypeMp}},
				}
				handleSnapshotStatChanged(sc, nil)(logrus.New(), ctx, e)
				if v := snapshot.GetRegistry().View(tm, 46); !v.CoreValid {
					t.Fatalf("other-world events must be ignored")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.run)
	}
}

// distributionMappingCase drives both TestBuildIncreaseExperienceConfig and
// TestExperienceDistributionTypeExhaustiveness. `types` names the distribution
// types the case is the coverage owner of; multi-distribution and unknown-type
// cases declare nil so they contribute nothing to the coverage set.
type distributionMappingCase struct {
	name  string
	types []string
	given []character2.ExperienceDistributions
	want  model2.IncreaseExperienceConfig
}

var distributionMappingCases = []distributionMappingCase{
	{
		name:  "White_PrimaryWhiteText",
		types: []string{character2.ExperienceDistributionTypeWhite},
		given: []character2.ExperienceDistributions{
			{ExperienceType: character2.ExperienceDistributionTypeWhite, Amount: 1000},
		},
		want: model2.IncreaseExperienceConfig{White: true, Amount: 1000},
	},
	{
		name:  "Yellow_PrimaryYellowText",
		types: []string{character2.ExperienceDistributionTypeYellow},
		given: []character2.ExperienceDistributions{
			{ExperienceType: character2.ExperienceDistributionTypeYellow, Amount: 2000},
		},
		want: model2.IncreaseExperienceConfig{Amount: 2000},
	},
	{
		name:  "Chat_PrimaryInChat",
		types: []string{character2.ExperienceDistributionTypeChat},
		given: []character2.ExperienceDistributions{
			{ExperienceType: character2.ExperienceDistributionTypeChat, Amount: 3000},
		},
		want: model2.IncreaseExperienceConfig{InChat: true, Amount: 3000},
	},
	{
		name:  "MonsterBook_BonusEventExp",
		types: []string{character2.ExperienceDistributionTypeMonsterBook},
		given: []character2.ExperienceDistributions{
			{ExperienceType: character2.ExperienceDistributionTypeMonsterBook, Amount: 4000},
		},
		want: model2.IncreaseExperienceConfig{MonsterBookBonus: 4000},
	},
	{
		name:  "MonsterEvent_MobEventPercentage",
		types: []string{character2.ExperienceDistributionTypeMonsterEvent},
		given: []character2.ExperienceDistributions{
			{ExperienceType: character2.ExperienceDistributionTypeMonsterEvent, Amount: 11},
		},
		want: model2.IncreaseExperienceConfig{MobEventBonusPercentage: 11},
	},
	{
		name:  "PlayTime_MobEventPercentageAndHours",
		types: []string{character2.ExperienceDistributionTypePlayTime},
		given: []character2.ExperienceDistributions{
			{ExperienceType: character2.ExperienceDistributionTypePlayTime, Amount: 22, Attr1: 33},
		},
		want: model2.IncreaseExperienceConfig{MobEventBonusPercentage: 22, PlayTimeHour: 33},
	},
	{
		name:  "Wedding_BonusWeddingExp",
		types: []string{character2.ExperienceDistributionTypeWedding},
		given: []character2.ExperienceDistributions{
			{ExperienceType: character2.ExperienceDistributionTypeWedding, Amount: 5000},
		},
		want: model2.IncreaseExperienceConfig{WeddingBonusEXP: 5000},
	},
	{
		name:  "SpiritWeek_QuestBonusRate",
		types: []string{character2.ExperienceDistributionTypeSpiritWeek},
		given: []character2.ExperienceDistributions{
			{ExperienceType: character2.ExperienceDistributionTypeSpiritWeek, Amount: 55},
		},
		want: model2.IncreaseExperienceConfig{QuestBonusRate: 55},
	},
	{
		name:  "Party_BonusExpAndEventRate",
		types: []string{character2.ExperienceDistributionTypeParty},
		given: []character2.ExperienceDistributions{
			{ExperienceType: character2.ExperienceDistributionTypeParty, Amount: 6000, Attr1: 44},
		},
		want: model2.IncreaseExperienceConfig{PartyBonusExp: 6000, PartyBonusEventRate: 44},
	},
	{
		// task-277 trap: an ITEM-only award leaves Amount at zero, so the
		// client renders "You have gained experience (+0)".
		name:  "Item_EquipItemBonusExpNotPrimary",
		types: []string{character2.ExperienceDistributionTypeItem},
		given: []character2.ExperienceDistributions{
			{ExperienceType: character2.ExperienceDistributionTypeItem, Amount: 7000},
		},
		want: model2.IncreaseExperienceConfig{ItemBonusEXP: 7000},
	},
	{
		name:  "InternetCafe_PremiumIpExp",
		types: []string{character2.ExperienceDistributionTypeInternetCafe},
		given: []character2.ExperienceDistributions{
			{ExperienceType: character2.ExperienceDistributionTypeInternetCafe, Amount: 8000},
		},
		want: model2.IncreaseExperienceConfig{PremiumIPExp: 8000},
	},
	{
		name:  "RainbowWeek_BonusEventExp",
		types: []string{character2.ExperienceDistributionTypeRainbowWeek},
		given: []character2.ExperienceDistributions{
			{ExperienceType: character2.ExperienceDistributionTypeRainbowWeek, Amount: 9000},
		},
		want: model2.IncreaseExperienceConfig{RainbowWeekEventEXP: 9000},
	},
	{
		name:  "PartyRing_ExpRingExp_v95Plus",
		types: []string{character2.ExperienceDistributionTypePartyRing},
		given: []character2.ExperienceDistributions{
			{ExperienceType: character2.ExperienceDistributionTypePartyRing, Amount: 10000},
		},
		want: model2.IncreaseExperienceConfig{PartyEXPRingEXP: 10000},
	},
	{
		name:  "CakePie_EventBonus_v95Plus",
		types: []string{character2.ExperienceDistributionTypeCakePie},
		given: []character2.ExperienceDistributions{
			{ExperienceType: character2.ExperienceDistributionTypeCakePie, Amount: 11000},
		},
		want: model2.IncreaseExperienceConfig{CakePieEventBonus: 11000},
	},
	{
		// Producer parity: character/processor.go appends WHITE and CHAT
		// with the same amount, so the primary-award shape combines both.
		name:  "WhiteAndChat_PrimaryAwardShape",
		types: nil,
		given: []character2.ExperienceDistributions{
			{ExperienceType: character2.ExperienceDistributionTypeWhite, Amount: 1500},
			{ExperienceType: character2.ExperienceDistributionTypeChat, Amount: 1500},
		},
		want: model2.IncreaseExperienceConfig{White: true, InChat: true, Amount: 1500},
	},
	{
		name:  "PrimaryPlusBonuses_Accumulate",
		types: nil,
		given: []character2.ExperienceDistributions{
			{ExperienceType: character2.ExperienceDistributionTypeWhite, Amount: 2500},
			{ExperienceType: character2.ExperienceDistributionTypeParty, Amount: 600, Attr1: 66},
			{ExperienceType: character2.ExperienceDistributionTypeItem, Amount: 770},
		},
		want: model2.IncreaseExperienceConfig{
			White: true, Amount: 2500,
			PartyBonusExp: 600, PartyBonusEventRate: 66,
			ItemBonusEXP: 770,
		},
	},
	{
		// FR-8/FR-9: the last distribution to touch White/Amount wins; WHITE
		// followed by YELLOW clears White even though a WHITE entry was seen.
		name:  "WhiteThenYellow_LastWins",
		types: nil,
		given: []character2.ExperienceDistributions{
			{ExperienceType: character2.ExperienceDistributionTypeWhite, Amount: 1200},
			{ExperienceType: character2.ExperienceDistributionTypeYellow, Amount: 3400},
		},
		want: model2.IncreaseExperienceConfig{Amount: 3400},
	},
	{
		name:  "EmptySlice_ZeroConfig",
		types: nil,
		given: nil,
		want:  model2.IncreaseExperienceConfig{},
	},
	{
		// FR-6: "DEATH" is a real distribution type the producer emits
		// (atlas-character's ExperienceDistributionTypeDeath), but
		// atlas-channel has no arm for it, so it is silently dropped today.
		// This case pins that observed behavior.
		name:  "UnknownType_DeathIgnored",
		types: nil,
		given: []character2.ExperienceDistributions{
			{ExperienceType: "DEATH", Amount: 9999},
			{ExperienceType: character2.ExperienceDistributionTypeWhite, Amount: 1300},
		},
		want: model2.IncreaseExperienceConfig{White: true, Amount: 1300},
	},
}

func TestBuildIncreaseExperienceConfig(t *testing.T) {
	for _, tc := range distributionMappingCases {
		t.Run(tc.name, func(t *testing.T) {
			got := buildIncreaseExperienceConfig(tc.given)
			if got != tc.want {
				t.Errorf("config mismatch\n got: %+v\nwant: %+v", got, tc.want)
			}
		})
	}
}

func TestExperienceDistributionTypeExhaustiveness(t *testing.T) {
	covered := map[string]string{} // distribution type -> owning case name
	for _, tc := range distributionMappingCases {
		for _, dt := range tc.types {
			if prev, ok := covered[dt]; ok {
				t.Fatalf("distribution type %q claimed by two cases: %q and %q", dt, prev, tc.name)
			}
			covered[dt] = tc.name
		}
	}

	registered := map[string]bool{}
	for _, dt := range character2.AllExperienceDistributionTypes {
		registered[dt] = true
		if _, ok := covered[dt]; !ok {
			t.Errorf("distribution type %q is in AllExperienceDistributionTypes but has no case in distributionMappingCases", dt)
		}
	}

	for dt, name := range covered {
		if !registered[dt] {
			t.Errorf("case %q covers distribution type %q, which is not in AllExperienceDistributionTypes", name, dt)
		}
	}
}
