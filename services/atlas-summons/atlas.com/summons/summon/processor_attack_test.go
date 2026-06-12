package summon

import (
	"context"
	"encoding/json"
	"testing"

	monstermsg "atlas-summons/monster"

	"atlas-summons/data/skill/effect"
	"atlas-summons/effectivestats"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	objectid "github.com/Chronicle20/atlas/libs/atlas-object-id"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

// stubStatsSource is a fake effective-stats source satisfying the statsSource
// seam on ProcessorImpl. It returns a fixed model so the damage ceiling is
// deterministic without a live atlas-effective-stats.
type stubStatsSource struct {
	model effectivestats.Model
	err   error
}

func (s stubStatsSource) GetByCharacter(_ world.Id, _ channel.Id, _ uint32) (effectivestats.Model, error) {
	return s.model, s.err
}

// stubWeaponSource is a fake equipped-weapon-type source satisfying the
// weaponSource seam. It returns a fixed weapon type so the physical ceiling is
// deterministic without a live atlas-inventory.
type stubWeaponSource struct {
	weaponType item.WeaponType
	err        error
}

func (s stubWeaponSource) GetEquippedWeaponType(_ uint32) (item.WeaponType, error) {
	return s.weaponType, s.err
}

// capturedMessage is a topic + a decoded payload map captured by the fake emitter.
type capturedMessage struct {
	topic   string
	payload map[string]any
}

// newAttackProcessor wires a ProcessorImpl backed by miniredis, a stub effect
// source, a stub effective-stats source, and a capturing emitter. rollProc is
// forced to always-proc so APPLY_STATUS emission is deterministic.
func newAttackProcessor(t *testing.T, eff effect.Model, watk uint32, statsErr error) (*ProcessorImpl, *[]capturedMessage) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(mr.Close)
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})

	registry = newRegistry(rc)
	idAllocator = &IdAllocator{inner: objectid.NewRedisAllocator(rc)}

	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx := tenant.WithContext(context.Background(), ten)

	// Non-zero str/dex so the faithful weapon-type ceiling computes a positive
	// bound (maxBaseDmg = 0 with zero stats would degrade to "no ceiling").
	stats, _ := effectivestats.Extract(effectivestats.RestModel{
		WeaponAttack: watk, MagicAttack: watk, Strength: 200, Dexterity: 100, Luck: 50,
	})

	captured := &[]capturedMessage{}
	p := &ProcessorImpl{
		l:       logrus.New(),
		ctx:     ctx,
		t:       ten,
		effects: stubEffectSource{eff: eff},
		stats:   stubStatsSource{model: stats, err: statsErr},
		equip:   stubWeaponSource{weaponType: item.WeaponTypeOneHandedSword},
		// force a proc so the test can assert APPLY_STATUS deterministically.
		proc: func(_ float64) bool { return true },
		emit: func(topic string, provider model.Provider[[]kafka.Message]) error {
			msgs, perr := provider()
			if perr != nil {
				return perr
			}
			for _, msg := range msgs {
				var payload map[string]any
				_ = json.Unmarshal(msg.Value, &payload)
				*captured = append(*captured, capturedMessage{topic: topic, payload: payload})
			}
			return nil
		},
	}
	return p, captured
}

// effectAttacker builds a physical-attacker effect (weaponAttack > 0) with a
// proc chance, used to drive the damage ceiling and status proc.
func effectAttacker(watk int16, prop float64) effect.Model {
	e, _ := effect.Extract(effect.RestModel{
		WeaponAttack: watk,
		Duration:     5000,
		Prop:         prop,
	})
	return e
}

func spawnAttacker(t *testing.T, p *ProcessorImpl, skillId uint32, owner uint32) Model {
	t.Helper()
	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(100000000)).SetInstance(uuid.Nil).Build()
	m, err := p.Spawn(f, owner, skillId, 10, 100, -50)
	if err != nil {
		t.Fatalf("Spawn returned error: %v", err)
	}
	if m.Id() == 0 {
		t.Fatalf("expected non-zero summon id from spawn")
	}
	return m
}

func TestAttackCreditsOwnerAndClamps(t *testing.T) {
	// Silver Hawk 3111005, physical attacker (effect weaponAttack > 0), owner 42.
	p, captured := newAttackProcessor(t, effectAttacker(50, 1.0), 200, nil)
	m := spawnAttacker(t, p, 3111005, 42)

	const reported = uint32(4000000) // absurd, must be clamped
	err := p.Attack(m.Id(), 42, 0, []AttackTarget{{MonsterId: 9999, Damage: reported}})
	if err != nil {
		t.Fatalf("Attack returned error: %v", err)
	}

	var dmg *capturedMessage
	var status *capturedMessage
	var attacked *capturedMessage
	for i := range *captured {
		c := &(*captured)[i]
		if c.topic == monstermsg.EnvCommandTopic && c.payload["type"] == monstermsg.CommandTypeDamage {
			dmg = c
		}
		if c.topic == monstermsg.EnvCommandTopic && c.payload["type"] == monstermsg.CommandTypeApplyStatus {
			status = c
		}
		if c.topic == EnvEventTopicSummonStatus && c.payload["type"] == EventSummonStatusAttacked {
			attacked = c
		}
	}
	if dmg == nil {
		t.Fatalf("expected a COMMAND_TOPIC_MONSTER DAMAGE message; got %+v", *captured)
	}
	body, _ := dmg.payload["body"].(map[string]any)
	if body == nil {
		t.Fatalf("DAMAGE message missing body: %+v", dmg.payload)
	}
	if cid := uint32(body["characterId"].(float64)); cid != 42 {
		t.Fatalf("expected DAMAGE characterId == owner 42, got %d", cid)
	}
	damages, _ := body["damages"].([]any)
	if len(damages) != 1 {
		t.Fatalf("expected exactly one damage value, got %v", damages)
	}
	emitted := uint32(damages[0].(float64))
	if emitted >= reported {
		t.Fatalf("expected emitted damage clamped below reported (%d), got %d", reported, emitted)
	}
	if emitted == 0 {
		t.Fatalf("expected emitted damage > 0 (legit clamp), got 0")
	}
	// Silver Hawk has Stun: true and the proc is forced => APPLY_STATUS expected.
	if status == nil {
		t.Fatalf("expected a COMMAND_TOPIC_MONSTER APPLY_STATUS message (forced proc); got %+v", *captured)
	}
	if attacked == nil {
		t.Fatalf("expected an ATTACKED status event; got %+v", *captured)
	}
	// ATTACKED must carry the CLAMPED damage, not the raw reported value.
	abody, _ := attacked.payload["body"].(map[string]any)
	tgts, _ := abody["targets"].([]any)
	if len(tgts) != 1 {
		t.Fatalf("expected one ATTACKED target, got %v", tgts)
	}
	at0, _ := tgts[0].(map[string]any)
	if ad := uint32(at0["damage"].(float64)); uint32(ad) != emitted {
		t.Fatalf("expected ATTACKED target damage to equal clamped value %d, got %d", emitted, ad)
	}
}

func TestGaviotaSelfCancels(t *testing.T) {
	p, _ := newAttackProcessor(t, effectAttacker(50, 1.0), 200, nil)
	m := spawnAttacker(t, p, 5211002, 42) // Gaviota, OneShot

	if err := p.Attack(m.Id(), 42, 0, []AttackTarget{{MonsterId: 9999, Damage: 1000}}); err != nil {
		t.Fatalf("Attack returned error: %v", err)
	}
	if _, err := p.GetById(m.Id()); err == nil {
		t.Fatalf("expected Gaviota to be despawned after one attack, but GetById found it")
	}
}

func TestAttackByNonOwnerDropped(t *testing.T) {
	p, captured := newAttackProcessor(t, effectAttacker(50, 1.0), 200, nil)
	m := spawnAttacker(t, p, 3111005, 42)

	if err := p.Attack(m.Id(), 99, 0, []AttackTarget{{MonsterId: 9999, Damage: 1000}}); err != nil {
		t.Fatalf("Attack returned error: %v", err)
	}
	for _, c := range *captured {
		if c.topic == monstermsg.EnvCommandTopic && c.payload["type"] == monstermsg.CommandTypeDamage {
			t.Fatalf("expected no DAMAGE emitted for non-owner attack; got %+v", c.payload)
		}
		if c.topic == EnvEventTopicSummonStatus && c.payload["type"] == EventSummonStatusAttacked {
			t.Fatalf("expected no ATTACKED event for non-owner attack; got %+v", c.payload)
		}
	}
}
