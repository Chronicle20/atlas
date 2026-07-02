package character

import (
	"atlas-character/kafka/message"
	character2 "atlas-character/kafka/message/character"
	"encoding/json"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	"github.com/Chronicle20/atlas/libs/atlas-constants/stat"
	databasetest "github.com/Chronicle20/atlas/libs/atlas-database/databasetest"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// -- shared test harness (mirrors provider_test.go's newCharsDB / processor_test.go's
// testDatabase-testTenant-testLogger, adapted for the internal `character` package
// so unexported entity fields can be set directly without a public setter path for
// Hp/MaxHp/HpMpUsed/JobId/Level). --

func transferApLogger() logrus.FieldLogger {
	l, _ := test.NewNullLogger()
	return l
}

// newTransferApFixture returns a fresh in-memory tenant-scoped DB, a tenant-bound
// Processor, and creates one character row from the supplied entity template
// (TenantId is injected). Returns the created character id.
func newTransferApFixture(t *testing.T, e entity) (*gorm.DB, Processor, uint32) {
	t.Helper()
	db := databasetest.NewInMemoryTenantDB(t, Migration)
	tid := uuid.New()
	ctx := databasetest.TenantContext(tid)
	e.TenantId = tid
	require.NoError(t, db.Create(&e).Error)
	p := NewProcessor(transferApLogger(), ctx, db)
	return db, p, e.ID
}

func decodeStatusEvent[T any](t *testing.T, mb *message.Buffer) character2.StatusEvent[T] {
	t.Helper()
	msgs := mb.GetAll()[character2.EnvEventTopicCharacterStatus]
	if len(msgs) != 1 {
		t.Fatalf("expected exactly 1 character status message, got %d", len(msgs))
	}
	var evt character2.StatusEvent[T]
	if err := json.Unmarshal(msgs[0].Value, &evt); err != nil {
		t.Fatalf("failed to unmarshal status event: %v", err)
	}
	return evt
}

func containsStat(list []stat.Type, want stat.Type) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}

const warriorJobId = job.Id(100)

// Case 1: STR->DEX success: STR 10 -> 9, DEX 4 -> 5; STAT_CHANGED buffered
// with stat.TypeStrength and stat.TypeDexterity.
func TestTransferAP_Case1_STRtoDEX_Success(t *testing.T) {
	_, p, id := newTransferApFixture(t, entity{
		AccountId: 1000, Name: "Case1", Strength: 10, Dexterity: 4,
	})

	mb := message.NewBuffer()
	txId := uuid.New()
	err := p.TransferAP(mb)(txId, id, channel.NewModel(0, 1), CommandDistributeApAbilityStrength, CommandDistributeApAbilityDexterity)
	if err != nil {
		t.Fatalf("TransferAP returned error: %v", err)
	}

	c, err := p.GetById()(id)
	if err != nil {
		t.Fatalf("GetById failed: %v", err)
	}
	if c.Strength() != 9 {
		t.Errorf("expected Strength 9, got %d", c.Strength())
	}
	if c.Dexterity() != 5 {
		t.Errorf("expected Dexterity 5, got %d", c.Dexterity())
	}

	evt := decodeStatusEvent[character2.StatusEventStatChangedBody](t, mb)
	if evt.Type != character2.StatusEventTypeStatChanged {
		t.Fatalf("expected STAT_CHANGED event, got %q", evt.Type)
	}
	if !containsStat(evt.Body.Updates, stat.TypeStrength) {
		t.Errorf("expected Updates to contain TypeStrength, got %+v", evt.Body.Updates)
	}
	if !containsStat(evt.Body.Updates, stat.TypeDexterity) {
		t.Errorf("expected Updates to contain TypeDexterity, got %+v", evt.Body.Updates)
	}
}

// Case 2: STR at 4 -> rejected STAT_AT_MINIMUM detail STRENGTH; nothing mutated.
func TestTransferAP_Case2_SourceAtMinimum_Rejected(t *testing.T) {
	_, p, id := newTransferApFixture(t, entity{
		AccountId: 1000, Name: "Case2", Strength: 4, Dexterity: 4,
	})

	mb := message.NewBuffer()
	err := p.TransferAP(mb)(uuid.New(), id, channel.NewModel(0, 1), CommandDistributeApAbilityStrength, CommandDistributeApAbilityDexterity)
	if err != nil {
		t.Fatalf("TransferAP returned error: %v", err)
	}

	c, err := p.GetById()(id)
	if err != nil {
		t.Fatalf("GetById failed: %v", err)
	}
	if c.Strength() != 4 {
		t.Errorf("expected Strength unchanged at 4, got %d", c.Strength())
	}
	if c.Dexterity() != 4 {
		t.Errorf("expected Dexterity unchanged at 4, got %d", c.Dexterity())
	}

	evt := decodeStatusEvent[character2.StatusEventApTransferErrorBody](t, mb)
	if evt.Type != character2.StatusEventTypeError {
		t.Fatalf("expected ERROR event, got %q", evt.Type)
	}
	if evt.Body.Error != character2.StatusEventErrorTypeStatAtMinimum {
		t.Errorf("expected error STAT_AT_MINIMUM, got %q", evt.Body.Error)
	}
	if evt.Body.Detail != CommandDistributeApAbilityStrength {
		t.Errorf("expected detail STRENGTH, got %q", evt.Body.Detail)
	}
}

// Case 3: Target DEX at 32767 -> rejected STAT_AT_MAXIMUM detail DEXTERITY; nothing mutated.
func TestTransferAP_Case3_TargetAtMaximum_Rejected(t *testing.T) {
	_, p, id := newTransferApFixture(t, entity{
		AccountId: 1000, Name: "Case3", Strength: 10, Dexterity: 32767,
	})

	mb := message.NewBuffer()
	err := p.TransferAP(mb)(uuid.New(), id, channel.NewModel(0, 1), CommandDistributeApAbilityStrength, CommandDistributeApAbilityDexterity)
	if err != nil {
		t.Fatalf("TransferAP returned error: %v", err)
	}

	c, err := p.GetById()(id)
	if err != nil {
		t.Fatalf("GetById failed: %v", err)
	}
	if c.Strength() != 10 {
		t.Errorf("expected Strength unchanged at 10, got %d", c.Strength())
	}
	if c.Dexterity() != 32767 {
		t.Errorf("expected Dexterity unchanged at 32767, got %d", c.Dexterity())
	}

	evt := decodeStatusEvent[character2.StatusEventApTransferErrorBody](t, mb)
	if evt.Body.Error != character2.StatusEventErrorTypeStatAtMaximum {
		t.Errorf("expected error STAT_AT_MAXIMUM, got %q", evt.Body.Error)
	}
	if evt.Body.Detail != CommandDistributeApAbilityDexterity {
		t.Errorf("expected detail DEXTERITY, got %q", evt.Body.Detail)
	}
}

// Case 4: HP->STR with hpMpUsed 0 -> rejected INSUFFICIENT_HPMP_AP_USED; nothing mutated.
func TestTransferAP_Case4_InsufficientHpMpUsed_Rejected(t *testing.T) {
	_, p, id := newTransferApFixture(t, entity{
		AccountId: 1000, Name: "Case4", Strength: 10,
		MaxHp: 2000, Hp: 2000, HpMpUsed: 0,
	})

	mb := message.NewBuffer()
	err := p.TransferAP(mb)(uuid.New(), id, channel.NewModel(0, 1), CommandDistributeApAbilityHp, CommandDistributeApAbilityStrength)
	if err != nil {
		t.Fatalf("TransferAP returned error: %v", err)
	}

	c, err := p.GetById()(id)
	if err != nil {
		t.Fatalf("GetById failed: %v", err)
	}
	if c.MaxHp() != 2000 || c.Hp() != 2000 {
		t.Errorf("expected HP/MaxHp unchanged at 2000/2000, got %d/%d", c.Hp(), c.MaxHp())
	}
	if c.Strength() != 10 {
		t.Errorf("expected Strength unchanged at 10, got %d", c.Strength())
	}
	if c.HpMpUsed() != 0 {
		t.Errorf("expected HpMpUsed unchanged at 0, got %d", c.HpMpUsed())
	}

	evt := decodeStatusEvent[character2.StatusEventApTransferErrorBody](t, mb)
	if evt.Body.Error != character2.StatusEventErrorTypeInsufficientHpMpApUsed {
		t.Errorf("expected error INSUFFICIENT_HPMP_AP_USED, got %q", evt.Body.Error)
	}
	if evt.Body.Detail != CommandDistributeApAbilityHp {
		t.Errorf("expected detail HP, got %q", evt.Body.Detail)
	}
}

// Case 5: HP->STR success (warrior job 100, level 30, MaxHp 2000, Hp 2000,
// hpMpUsed 2): MaxHp 2000 -> 1946 (-54), Hp 2000 -> 1946, hpMpUsed 2 -> 1, STR +1.
func TestTransferAP_Case5_HPtoSTR_Success(t *testing.T) {
	_, p, id := newTransferApFixture(t, entity{
		AccountId: 1000, Name: "Case5", JobId: warriorJobId, Level: 30,
		Strength: 10, MaxHp: 2000, Hp: 2000, HpMpUsed: 2,
	})

	mb := message.NewBuffer()
	err := p.TransferAP(mb)(uuid.New(), id, channel.NewModel(0, 1), CommandDistributeApAbilityHp, CommandDistributeApAbilityStrength)
	if err != nil {
		t.Fatalf("TransferAP returned error: %v", err)
	}

	c, err := p.GetById()(id)
	if err != nil {
		t.Fatalf("GetById failed: %v", err)
	}
	if c.MaxHp() != 1946 {
		t.Errorf("expected MaxHp 1946, got %d", c.MaxHp())
	}
	if c.Hp() != 1946 {
		t.Errorf("expected Hp 1946, got %d", c.Hp())
	}
	if c.HpMpUsed() != 1 {
		t.Errorf("expected HpMpUsed 1, got %d", c.HpMpUsed())
	}
	if c.Strength() != 11 {
		t.Errorf("expected Strength 11, got %d", c.Strength())
	}

	evt := decodeStatusEvent[character2.StatusEventStatChangedBody](t, mb)
	if evt.Type != character2.StatusEventTypeStatChanged {
		t.Fatalf("expected STAT_CHANGED event, got %q", evt.Type)
	}
}

// Case 6: HP->STR where MaxHp-54 < pointResetMinHp(100, level) -> rejected
// POOL_BELOW_JOB_MINIMUM detail HP; nothing mutated.
func TestTransferAP_Case6_PoolBelowJobMinimum_Rejected(t *testing.T) {
	level := byte(30)
	minHp := pointResetMinHp(warriorJobId, level) // 24*30+118 = 838
	maxHp := uint16(minHp) + 54 - 1               // one below the floor after -54

	_, p, id := newTransferApFixture(t, entity{
		AccountId: 1000, Name: "Case6", JobId: warriorJobId, Level: level,
		Strength: 10, MaxHp: maxHp, Hp: maxHp, HpMpUsed: 1,
	})

	mb := message.NewBuffer()
	err := p.TransferAP(mb)(uuid.New(), id, channel.NewModel(0, 1), CommandDistributeApAbilityHp, CommandDistributeApAbilityStrength)
	if err != nil {
		t.Fatalf("TransferAP returned error: %v", err)
	}

	c, err := p.GetById()(id)
	if err != nil {
		t.Fatalf("GetById failed: %v", err)
	}
	if c.MaxHp() != maxHp || c.Hp() != maxHp {
		t.Errorf("expected HP/MaxHp unchanged at %d, got %d/%d", maxHp, c.Hp(), c.MaxHp())
	}
	if c.Strength() != 10 {
		t.Errorf("expected Strength unchanged at 10, got %d", c.Strength())
	}
	if c.HpMpUsed() != 1 {
		t.Errorf("expected HpMpUsed unchanged at 1, got %d", c.HpMpUsed())
	}

	evt := decodeStatusEvent[character2.StatusEventApTransferErrorBody](t, mb)
	if evt.Body.Error != character2.StatusEventErrorTypePoolBelowJobMinimum {
		t.Errorf("expected error POOL_BELOW_JOB_MINIMUM, got %q", evt.Body.Error)
	}
	if evt.Body.Detail != CommandDistributeApAbilityHp {
		t.Errorf("expected detail HP, got %q", evt.Body.Detail)
	}
}

// Case 7: STR->HP success (warrior): MaxHp +20 (gain table), hpMpUsed +1,
// STR -1; MaxHp at 30000 -> rejected STAT_AT_MAXIMUM detail HP; MaxHp
// approaching the pool cap from below is clamped to the cap rather than
// overshooting it.
func TestTransferAP_Case7_STRtoHP(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		_, p, id := newTransferApFixture(t, entity{
			AccountId: 1000, Name: "Case7Success", JobId: warriorJobId, Level: 30,
			Strength: 10, MaxHp: 1000, Hp: 1000, HpMpUsed: 0,
		})

		mb := message.NewBuffer()
		err := p.TransferAP(mb)(uuid.New(), id, channel.NewModel(0, 1), CommandDistributeApAbilityStrength, CommandDistributeApAbilityHp)
		if err != nil {
			t.Fatalf("TransferAP returned error: %v", err)
		}

		c, err := p.GetById()(id)
		if err != nil {
			t.Fatalf("GetById failed: %v", err)
		}
		if c.MaxHp() != 1020 {
			t.Errorf("expected MaxHp 1020, got %d", c.MaxHp())
		}
		if c.HpMpUsed() != 1 {
			t.Errorf("expected HpMpUsed 1, got %d", c.HpMpUsed())
		}
		if c.Strength() != 9 {
			t.Errorf("expected Strength 9, got %d", c.Strength())
		}

		evt := decodeStatusEvent[character2.StatusEventStatChangedBody](t, mb)
		if evt.Type != character2.StatusEventTypeStatChanged {
			t.Fatalf("expected STAT_CHANGED event, got %q", evt.Type)
		}
	})

	t.Run("at_cap_rejected", func(t *testing.T) {
		_, p, id := newTransferApFixture(t, entity{
			AccountId: 1000, Name: "Case7Cap", JobId: warriorJobId, Level: 30,
			Strength: 10, MaxHp: 30000, Hp: 30000, HpMpUsed: 0,
		})

		mb := message.NewBuffer()
		err := p.TransferAP(mb)(uuid.New(), id, channel.NewModel(0, 1), CommandDistributeApAbilityStrength, CommandDistributeApAbilityHp)
		if err != nil {
			t.Fatalf("TransferAP returned error: %v", err)
		}

		c, err := p.GetById()(id)
		if err != nil {
			t.Fatalf("GetById failed: %v", err)
		}
		if c.MaxHp() != 30000 {
			t.Errorf("expected MaxHp unchanged at 30000, got %d", c.MaxHp())
		}
		if c.Strength() != 10 {
			t.Errorf("expected Strength unchanged at 10, got %d", c.Strength())
		}

		evt := decodeStatusEvent[character2.StatusEventApTransferErrorBody](t, mb)
		if evt.Body.Error != character2.StatusEventErrorTypeStatAtMaximum {
			t.Errorf("expected error STAT_AT_MAXIMUM, got %q", evt.Body.Error)
		}
		if evt.Body.Detail != CommandDistributeApAbilityHp {
			t.Errorf("expected detail HP, got %q", evt.Body.Detail)
		}
	})

	t.Run("clamp_post_add", func(t *testing.T) {
		_, p, id := newTransferApFixture(t, entity{
			AccountId: 1000, Name: "Case7Clamp", JobId: warriorJobId, Level: 30,
			Strength: 10, MaxHp: 29985, Hp: 29985, HpMpUsed: 0,
		})

		mb := message.NewBuffer()
		err := p.TransferAP(mb)(uuid.New(), id, channel.NewModel(0, 1), CommandDistributeApAbilityStrength, CommandDistributeApAbilityHp)
		if err != nil {
			t.Fatalf("TransferAP returned error: %v", err)
		}

		c, err := p.GetById()(id)
		if err != nil {
			t.Fatalf("GetById failed: %v", err)
		}
		// 29985 + 20 (warrior gainHp) = 30005, clamped down to the 30000 pool cap.
		if c.MaxHp() != 30000 {
			t.Errorf("expected MaxHp clamped to 30000, got %d", c.MaxHp())
		}
	})
}

// Case 8: HP->MP success: MaxHp -54, Hp -54 (floored at 1), MaxMp +2,
// hpMpUsed net 0 (-1 +1).
func TestTransferAP_Case8_HPtoMP_Success(t *testing.T) {
	_, p, id := newTransferApFixture(t, entity{
		AccountId: 1000, Name: "Case8", JobId: warriorJobId, Level: 30,
		MaxHp: 2000, Hp: 30, MaxMp: 200, Mp: 200, HpMpUsed: 1,
	})

	mb := message.NewBuffer()
	err := p.TransferAP(mb)(uuid.New(), id, channel.NewModel(0, 1), CommandDistributeApAbilityHp, CommandDistributeApAbilityMp)
	if err != nil {
		t.Fatalf("TransferAP returned error: %v", err)
	}

	c, err := p.GetById()(id)
	if err != nil {
		t.Fatalf("GetById failed: %v", err)
	}
	if c.MaxHp() != 1946 {
		t.Errorf("expected MaxHp 1946, got %d", c.MaxHp())
	}
	if c.Hp() != 1 {
		t.Errorf("expected Hp floored at 1, got %d", c.Hp())
	}
	if c.MaxMp() != 202 {
		t.Errorf("expected MaxMp 202, got %d", c.MaxMp())
	}
	if c.HpMpUsed() != 1 {
		t.Errorf("expected HpMpUsed net unchanged at 1, got %d", c.HpMpUsed())
	}

	evt := decodeStatusEvent[character2.StatusEventStatChangedBody](t, mb)
	if evt.Type != character2.StatusEventTypeStatChanged {
		t.Fatalf("expected STAT_CHANGED event, got %q", evt.Type)
	}
}

// Case 9: From==To STR->STR with STR 10: processed, net value unchanged
// (validations still ran); STAT_CHANGED still emitted (zero-mod success).
func TestTransferAP_Case9_FromEqualsTo_STR_ZeroModStillEmits(t *testing.T) {
	_, p, id := newTransferApFixture(t, entity{
		AccountId: 1000, Name: "Case9", Strength: 10,
	})

	mb := message.NewBuffer()
	err := p.TransferAP(mb)(uuid.New(), id, channel.NewModel(0, 1), CommandDistributeApAbilityStrength, CommandDistributeApAbilityStrength)
	if err != nil {
		t.Fatalf("TransferAP returned error: %v", err)
	}

	c, err := p.GetById()(id)
	if err != nil {
		t.Fatalf("GetById failed: %v", err)
	}
	if c.Strength() != 10 {
		t.Errorf("expected Strength unchanged at 10, got %d", c.Strength())
	}

	evt := decodeStatusEvent[character2.StatusEventStatChangedBody](t, mb)
	if evt.Type != character2.StatusEventTypeStatChanged {
		t.Fatalf("expected STAT_CHANGED event even for a zero-mod From==To transfer, got %q", evt.Type)
	}
	if len(evt.Body.Updates) != 0 {
		t.Errorf("expected empty Updates for a zero-mod From==To transfer, got %+v", evt.Body.Updates)
	}
}

// Case 10: From==To HP->HP (warrior, hpMpUsed >= 1, pool comfortably above
// minimum): MaxHp net -34 (-54 +20), hpMpUsed net 0.
func TestTransferAP_Case10_FromEqualsTo_HP_NetChange(t *testing.T) {
	_, p, id := newTransferApFixture(t, entity{
		AccountId: 1000, Name: "Case10", JobId: warriorJobId, Level: 30,
		MaxHp: 2000, Hp: 2000, HpMpUsed: 1,
	})

	mb := message.NewBuffer()
	err := p.TransferAP(mb)(uuid.New(), id, channel.NewModel(0, 1), CommandDistributeApAbilityHp, CommandDistributeApAbilityHp)
	if err != nil {
		t.Fatalf("TransferAP returned error: %v", err)
	}

	c, err := p.GetById()(id)
	if err != nil {
		t.Fatalf("GetById failed: %v", err)
	}
	if c.MaxHp() != 1966 {
		t.Errorf("expected MaxHp net -34 -> 1966, got %d", c.MaxHp())
	}
	if c.HpMpUsed() != 1 {
		t.Errorf("expected HpMpUsed net unchanged at 1, got %d", c.HpMpUsed())
	}
	if c.Hp() != 1946 {
		t.Errorf("expected Hp reduced by takeHp to 1946, got %d", c.Hp())
	}

	evt := decodeStatusEvent[character2.StatusEventStatChangedBody](t, mb)
	if evt.Type != character2.StatusEventTypeStatChanged {
		t.Fatalf("expected STAT_CHANGED event, got %q", evt.Type)
	}
}

// Case 11: Invalid ability string (e.g. "FAME") -> rejected INVALID_TARGET;
// nothing mutated.
func TestTransferAP_Case11_InvalidAbility_Rejected(t *testing.T) {
	_, p, id := newTransferApFixture(t, entity{
		AccountId: 1000, Name: "Case11", Strength: 10,
	})

	mb := message.NewBuffer()
	err := p.TransferAP(mb)(uuid.New(), id, channel.NewModel(0, 1), "FAME", CommandDistributeApAbilityStrength)
	if err != nil {
		t.Fatalf("TransferAP returned error: %v", err)
	}

	c, err := p.GetById()(id)
	if err != nil {
		t.Fatalf("GetById failed: %v", err)
	}
	if c.Strength() != 10 {
		t.Errorf("expected Strength unchanged at 10, got %d", c.Strength())
	}

	evt := decodeStatusEvent[character2.StatusEventApTransferErrorBody](t, mb)
	if evt.Body.Error != character2.StatusEventErrorTypeApTransferInvalidTarget {
		t.Errorf("expected error INVALID_TARGET, got %q", evt.Body.Error)
	}
	if evt.Body.Detail != "FAME" {
		t.Errorf("expected detail FAME, got %q", evt.Body.Detail)
	}
}

// Case 12: Current HP floor: Hp 30, take 54 -> Hp floored at 1 (not underflowed).
func TestTransferAP_Case12_CurrentHpFloor(t *testing.T) {
	_, p, id := newTransferApFixture(t, entity{
		AccountId: 1000, Name: "Case12", JobId: warriorJobId, Level: 1,
		Strength: 10, MaxHp: 1000, Hp: 30, HpMpUsed: 1,
	})

	mb := message.NewBuffer()
	err := p.TransferAP(mb)(uuid.New(), id, channel.NewModel(0, 1), CommandDistributeApAbilityHp, CommandDistributeApAbilityStrength)
	if err != nil {
		t.Fatalf("TransferAP returned error: %v", err)
	}

	c, err := p.GetById()(id)
	if err != nil {
		t.Fatalf("GetById failed: %v", err)
	}
	if c.Hp() != 1 {
		t.Errorf("expected Hp floored at 1, got %d", c.Hp())
	}
	if c.MaxHp() != 946 {
		t.Errorf("expected MaxHp 946, got %d", c.MaxHp())
	}
	if c.Strength() != 11 {
		t.Errorf("expected Strength 11, got %d", c.Strength())
	}

	evt := decodeStatusEvent[character2.StatusEventStatChangedBody](t, mb)
	if evt.Type != character2.StatusEventTypeStatChanged {
		t.Fatalf("expected STAT_CHANGED event, got %q", evt.Type)
	}
}
