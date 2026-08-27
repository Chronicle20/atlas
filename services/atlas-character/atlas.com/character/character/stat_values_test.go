package character

import (
	"atlas-character/kafka/message"
	character2 "atlas-character/kafka/message/character"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	"github.com/Chronicle20/atlas/libs/atlas-constants/stat"
	databasetest "github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
	outbox "github.com/Chronicle20/atlas/libs/atlas-outbox"
)

// -- shared harness --
//
// Reuses newTransferApFixture's shape (transfer_ap_test.go) — a fresh
// in-memory tenant-scoped DB seeded from an `entity` template so unexported
// columns (AP, JobId, MaxHp, ...) can be set directly — but additionally
// migrates the outbox table, since RequestChangeMeso and RequestDistributeAp
// route their STAT_CHANGED emission through outbox.EmitProvider rather than
// taking a *message.Buffer directly.

func newStatValuesFixture(t *testing.T, e entity) (*gorm.DB, Processor, uint32) {
	t.Helper()
	db := databasetest.NewInMemoryTenantDB(t, Migration, outbox.Migration)
	tid := uuid.New()
	ctx := databasetest.TenantContext(tid)
	e.TenantId = tid
	require.NoError(t, db.Create(&e).Error)
	p := NewProcessor(transferApLogger(), ctx, db)
	return db, p, e.ID
}

// decodeStatChangedFromOutbox reads the most recently enqueued STAT_CHANGED
// outbox row for the character status topic and decodes its payload. Used
// for processor methods (RequestChangeMeso, RequestDistributeAp) that emit
// via outbox.EmitProvider inside their own transaction rather than taking a
// caller-supplied *message.Buffer.
func decodeStatChangedFromOutbox(t *testing.T, db *gorm.DB) character2.StatusEvent[character2.StatusEventStatChangedBody] {
	t.Helper()
	var rows []outbox.Entity
	require.NoError(t, db.Where("topic = ?", character2.EnvEventTopicCharacterStatus).Order("id desc").Find(&rows).Error)
	for _, r := range rows {
		var evt character2.StatusEvent[character2.StatusEventStatChangedBody]
		if err := json.Unmarshal(r.MessageValue, &evt); err != nil {
			continue
		}
		if evt.Type == character2.StatusEventTypeStatChanged {
			return evt
		}
	}
	t.Fatal("no STAT_CHANGED event found in outbox")
	return character2.StatusEvent[character2.StatusEventStatChangedBody]{}
}

// decodeStatChangedFromBuffer decodes the STAT_CHANGED event out of a
// *message.Buffer that may carry other character-status events alongside it
// (e.g. AwardExperience also buffers EXPERIENCE_CHANGED; AwardLevel also
// buffers LEVEL_CHANGED) on the same topic.
func decodeStatChangedFromBuffer(t *testing.T, mb *message.Buffer) character2.StatusEvent[character2.StatusEventStatChangedBody] {
	t.Helper()
	msgs := mb.GetAll()[character2.EnvEventTopicCharacterStatus]
	for _, m := range msgs {
		var evt character2.StatusEvent[character2.StatusEventStatChangedBody]
		if err := json.Unmarshal(m.Value, &evt); err != nil {
			continue
		}
		if evt.Type == character2.StatusEventTypeStatChanged {
			return evt
		}
	}
	t.Fatal("no STAT_CHANGED event found in buffer")
	return character2.StatusEvent[character2.StatusEventStatChangedBody]{}
}

// statKeyFor mirrors the channel-side snake_case Values key table
// (atlas-channel: character/snapshot/registry.go's statValueKeys) so this
// test pins the producer side of the same contract.
func statKeyFor(t stat.Type) string {
	switch t {
	case stat.TypeSkin:
		return "skin"
	case stat.TypeFace:
		return "face"
	case stat.TypeHair:
		return "hair"
	case stat.TypeLevel:
		return "level"
	case stat.TypeJob:
		return "job"
	case stat.TypeStrength:
		return "strength"
	case stat.TypeDexterity:
		return "dexterity"
	case stat.TypeIntelligence:
		return "intelligence"
	case stat.TypeLuck:
		return "luck"
	case stat.TypeHp:
		return "hp"
	case stat.TypeMaxHp:
		return "max_hp"
	case stat.TypeMp:
		return "mp"
	case stat.TypeMaxMp:
		return "max_mp"
	case stat.TypeAvailableAP:
		return "available_ap"
	case stat.TypeAvailableSP:
		return "available_sp"
	case stat.TypeExperience:
		return "experience"
	case stat.TypeFame:
		return "fame"
	case stat.TypeMeso:
		return "meso"
	case stat.TypeGachaponExperience:
		return "gachapon_experience"
	}
	return ""
}

// modelFieldFor returns the post-mutation column corresponding to a
// statKeyFor key, re-read from the character Model. "available_sp" is
// special-cased to book 0 — every fixture in this file uses a Beginner
// (JobId 0) source/target job, which getSkillBook resolves to book 0.
func modelFieldFor(c Model, key string) interface{} {
	switch key {
	case "skin":
		return c.SkinColor()
	case "face":
		return c.Face()
	case "hair":
		return c.Hair()
	case "level":
		return c.Level()
	case "job":
		return c.JobId()
	case "strength":
		return c.Strength()
	case "dexterity":
		return c.Dexterity()
	case "intelligence":
		return c.Intelligence()
	case "luck":
		return c.Luck()
	case "hp":
		return c.Hp()
	case "max_hp":
		return c.MaxHp()
	case "mp":
		return c.Mp()
	case "max_mp":
		return c.MaxMp()
	case "available_ap":
		return c.AP()
	case "available_sp":
		return c.SP(0)
	case "experience":
		return c.Experience()
	case "fame":
		return c.Fame()
	case "meso":
		return c.Meso()
	}
	return nil
}

func toFloat64(t *testing.T, v interface{}) float64 {
	t.Helper()
	switch x := v.(type) {
	case uint16:
		return float64(x)
	case uint32:
		return float64(x)
	case byte:
		return float64(x)
	case int16:
		return float64(x)
	case int8:
		return float64(x)
	case job.Id:
		return float64(x)
	default:
		t.Fatalf("toFloat64: unsupported type %T (%v)", v, v)
		return 0
	}
}

// assertValuesCompleteForUpdates is the shared assertion: for every stat.Type
// in updates, its snake_case Values key must be present and numerically equal
// to the post-mutation column re-read from the DB. This would fail if a call
// site's values map were removed, extended incompletely, or reverted to nil —
// the exact regression task-122 guards against.
func assertValuesCompleteForUpdates(t *testing.T, updates []stat.Type, values map[string]interface{}, c Model) {
	t.Helper()
	require.NotEmpty(t, updates, "no Updates to verify — fixture produced an empty-update STAT_CHANGED")
	for _, u := range updates {
		key := statKeyFor(u)
		require.NotEmpty(t, key, "statKeyFor has no snake_case mapping for stat.Type %v — update the helper", u)
		want := modelFieldFor(c, key)
		require.NotNil(t, want, "modelFieldFor has no column mapping for key %q", key)
		got, ok := values[key]
		require.True(t, ok, "Values missing key %q for update %v — producer emitted an incomplete Values map", key, u)
		gotF, ok := got.(float64)
		require.True(t, ok, "Values[%q] decoded as non-numeric %T", key, got)
		require.Equal(t, toFloat64(t, want), gotF, "Values[%q] does not match post-mutation column", key)
	}
}

// TestStatChanged_ValuesCompleteOnHotPaths pins task-122's producer
// contract: every STAT_CHANGED carries one snake_case key per updated stat
// with the absolute post-mutation value. Flows driven directly against the
// ProcessorImpl over an in-memory tenant DB; the assertion helper decodes
// each captured STAT_CHANGED and requires, for every u in Updates,
// statKeyFor(u) present in Values and numerically equal to the post-mutation
// column re-read from the DB.

func TestStatChanged_ValuesCompleteOnHotPaths_ChangeMP(t *testing.T) {
	// The attack-path-critical site: every per-swing MP deduction must carry
	// an absolute mp so atlas-channel's snapshot can apply in place instead
	// of invalidating and refetching.
	_, p, id := newStatValuesFixture(t, entity{
		AccountId: 1000, Name: "ChangeMP", Level: 10, MaxMp: 100, Mp: 50,
	})

	mb := message.NewBuffer()
	txId := uuid.New()
	require.NoError(t, p.ChangeMP(mb)(txId, channel.NewModel(0, 1), id, 20))

	evt := decodeStatusEvent[character2.StatusEventStatChangedBody](t, mb)
	require.True(t, containsStat(evt.Body.Updates, stat.TypeMp))

	c, err := p.GetById()(id)
	require.NoError(t, err)
	require.Equal(t, uint16(70), c.Mp())
	assertValuesCompleteForUpdates(t, evt.Body.Updates, evt.Body.Values, c)
}

func TestStatChanged_ValuesCompleteOnHotPaths_ChangeHP(t *testing.T) {
	_, p, id := newStatValuesFixture(t, entity{
		AccountId: 1000, Name: "ChangeHP", Level: 10, MaxHp: 100, Hp: 50,
	})

	mb := message.NewBuffer()
	txId := uuid.New()
	require.NoError(t, p.ChangeHP(mb)(txId, channel.NewModel(0, 1), id, 20))

	evt := decodeStatusEvent[character2.StatusEventStatChangedBody](t, mb)
	require.True(t, containsStat(evt.Body.Updates, stat.TypeHp))

	c, err := p.GetById()(id)
	require.NoError(t, err)
	require.Equal(t, uint16(70), c.Hp())
	assertValuesCompleteForUpdates(t, evt.Body.Updates, evt.Body.Values, c)
}

func TestStatChanged_ValuesCompleteOnHotPaths_AwardExperience(t *testing.T) {
	_, p, id := newStatValuesFixture(t, entity{
		AccountId: 1000, Name: "AwardExp", Level: 1, Experience: 0,
	})

	mb := message.NewBuffer()
	txId := uuid.New()
	require.NoError(t, p.AwardExperience(mb)(txId, id, channel.NewModel(0, 1), []ExperienceModel{NewExperienceModel("WHITE", 1, 0)}, false))

	evt := decodeStatChangedFromBuffer(t, mb)
	require.True(t, containsStat(evt.Body.Updates, stat.TypeExperience))

	c, err := p.GetById()(id)
	require.NoError(t, err)
	require.Equal(t, uint32(1), c.Experience())
	assertValuesCompleteForUpdates(t, evt.Body.Updates, evt.Body.Values, c)
}

func TestStatChanged_ValuesCompleteOnHotPaths_AwardLevel(t *testing.T) {
	_, p, id := newStatValuesFixture(t, entity{
		AccountId: 1000, Name: "AwardLvl", Level: 5,
	})

	mb := message.NewBuffer()
	txId := uuid.New()
	require.NoError(t, p.AwardLevel(mb)(txId, id, channel.NewModel(0, 1), 1))

	evt := decodeStatChangedFromBuffer(t, mb)
	require.True(t, containsStat(evt.Body.Updates, stat.TypeLevel))

	c, err := p.GetById()(id)
	require.NoError(t, err)
	require.Equal(t, byte(6), c.Level())
	assertValuesCompleteForUpdates(t, evt.Body.Updates, evt.Body.Values, c)
}

func TestStatChanged_ValuesCompleteOnHotPaths_RequestChangeMeso(t *testing.T) {
	db, p, id := newStatValuesFixture(t, entity{
		AccountId: 1000, Name: "Meso1", Meso: 100,
	})

	txId := uuid.New()
	require.NoError(t, p.RequestChangeMeso(txId, id, 50, 0, "SYSTEM", false))

	evt := decodeStatChangedFromOutbox(t, db)
	require.True(t, containsStat(evt.Body.Updates, stat.TypeMeso))

	c, err := p.GetById()(id)
	require.NoError(t, err)
	require.Equal(t, uint32(150), c.Meso())
	assertValuesCompleteForUpdates(t, evt.Body.Updates, evt.Body.Values, c)
}

// -- controller ruling R10: the four "extend an existing map" sites --
//
// These four are where a missing key silently degrades to invalidate-and-
// refetch instead of an outright bug, so they need the same coverage as the
// greenfield sites above.

func TestStatChanged_ValuesCompleteOnHotPaths_APDistributeSuccess(t *testing.T) {
	db, p, id := newStatValuesFixture(t, entity{
		AccountId: 1000, Name: "APDist", Strength: 10, AP: 10,
	})

	txId := uuid.New()
	require.NoError(t, p.RequestDistributeAp(txId, id, []Distribution{
		{Ability: CommandDistributeApAbilityStrength, Amount: 5},
	}))

	evt := decodeStatChangedFromOutbox(t, db)
	require.True(t, containsStat(evt.Body.Updates, stat.TypeStrength))
	require.True(t, containsStat(evt.Body.Updates, stat.TypeAvailableAP))

	c, err := p.GetById()(id)
	require.NoError(t, err)
	require.Equal(t, uint16(15), c.Strength())
	require.Equal(t, uint16(5), c.AP())
	assertValuesCompleteForUpdates(t, evt.Body.Updates, evt.Body.Values, c)
}

func TestStatChanged_ValuesCompleteOnHotPaths_LevelUpGrowth(t *testing.T) {
	// JobId 0 (Beginner) + Level 15 keeps the level-up past the beginner
	// auto-STR/DEX threshold (<11) so AvailableAP is the branch exercised,
	// and avoids the improving-HP/MP skill lookups that only fire for the
	// combat job branches.
	_, p, id := newStatValuesFixture(t, entity{
		AccountId: 1000, Name: "LevelUp", Level: 15, JobId: job.Id(0),
		MaxHp: 100, Hp: 100, MaxMp: 50, Mp: 50, AP: 0,
	})

	mb := message.NewBuffer()
	txId := uuid.New()
	require.NoError(t, p.ProcessLevelChange(mb)(txId, channel.NewModel(0, 1), id, 1))

	evt := decodeStatusEvent[character2.StatusEventStatChangedBody](t, mb)
	require.True(t, containsStat(evt.Body.Updates, stat.TypeAvailableAP))
	require.True(t, containsStat(evt.Body.Updates, stat.TypeAvailableSP))
	require.True(t, containsStat(evt.Body.Updates, stat.TypeHp))
	require.True(t, containsStat(evt.Body.Updates, stat.TypeMaxHp))
	require.True(t, containsStat(evt.Body.Updates, stat.TypeMp))
	require.True(t, containsStat(evt.Body.Updates, stat.TypeMaxMp))

	c, err := p.GetById()(id)
	require.NoError(t, err)
	assertValuesCompleteForUpdates(t, evt.Body.Updates, evt.Body.Values, c)
}

func TestStatChanged_ValuesCompleteOnHotPaths_JobChangeGrowth(t *testing.T) {
	_, p, id := newStatValuesFixture(t, entity{
		AccountId: 1000, Name: "JobChange", Level: 10, JobId: job.Id(0),
		MaxHp: 100, Hp: 100, MaxMp: 50, Mp: 50, AP: 0,
	})

	mb := message.NewBuffer()
	txId := uuid.New()
	require.NoError(t, p.ProcessJobChange(mb)(txId, channel.NewModel(0, 1), id, job.Id(100)))

	evt := decodeStatusEvent[character2.StatusEventStatChangedBody](t, mb)
	require.True(t, containsStat(evt.Body.Updates, stat.TypeAvailableAP))
	require.True(t, containsStat(evt.Body.Updates, stat.TypeAvailableSP))
	require.True(t, containsStat(evt.Body.Updates, stat.TypeHp))
	require.True(t, containsStat(evt.Body.Updates, stat.TypeMaxHp))
	require.True(t, containsStat(evt.Body.Updates, stat.TypeMp))
	require.True(t, containsStat(evt.Body.Updates, stat.TypeMaxMp))

	c, err := p.GetById()(id)
	require.NoError(t, err)
	assertValuesCompleteForUpdates(t, evt.Body.Updates, evt.Body.Values, c)
}

func TestStatChanged_ValuesCompleteOnHotPaths_ResetStats(t *testing.T) {
	_, p, id := newStatValuesFixture(t, entity{
		AccountId: 1000, Name: "ResetStats", Strength: 10, Dexterity: 8, Intelligence: 6, Luck: 5, AP: 0,
	})

	mb := message.NewBuffer()
	txId := uuid.New()
	require.NoError(t, p.ResetStats(mb)(txId, id, channel.NewModel(0, 1)))

	evt := decodeStatusEvent[character2.StatusEventStatChangedBody](t, mb)
	require.True(t, containsStat(evt.Body.Updates, stat.TypeAvailableAP))

	c, err := p.GetById()(id)
	require.NoError(t, err)
	require.Equal(t, uint16(4), c.Strength())
	assertValuesCompleteForUpdates(t, evt.Body.Updates, evt.Body.Values, c)
}

func TestStatChanged_ValuesCompleteOnHotPaths_RebalanceAP(t *testing.T) {
	_, p, id := newStatValuesFixture(t, entity{
		AccountId: 1000, Name: "Rebalance", Strength: 10, Dexterity: 5, Intelligence: 4, Luck: 4, AP: 0,
	})

	mb := message.NewBuffer()
	txId := uuid.New()
	require.NoError(t, p.RebalanceAP(mb)(txId, id, channel.NewModel(0, 1), []RebalanceTarget{
		{Stat: "strength", Floor: 10},
	}))

	evt := decodeStatusEvent[character2.StatusEventStatChangedBody](t, mb)
	require.True(t, containsStat(evt.Body.Updates, stat.TypeAvailableAP))

	c, err := p.GetById()(id)
	require.NoError(t, err)
	assertValuesCompleteForUpdates(t, evt.Body.Updates, evt.Body.Values, c)
}
