package character

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

// EffectSkillUse - mode, skillId, characterLevel, skillLevel + conditional berserk/dragonFury/monsterMagnet
type EffectSkillUse struct {
	mode              byte
	skillId           uint32
	characterLevel    byte
	skillLevel        byte
	berserkDarkForce  bool
	dragonFuryCreate  bool
	monsterMagnetLeft bool
	isBerserk         bool
	isDragonFury      bool
	isMonsterMagnet   bool
}

func NewEffectSkillUse(mode byte, skillId uint32, characterLevel byte, skillLevel byte, isBerserk bool, berserkDarkForce bool, isDragonFury bool, dragonFuryCreate bool, isMonsterMagnet bool, monsterMagnetLeft bool) EffectSkillUse {
	return EffectSkillUse{
		mode:              mode,
		skillId:           skillId,
		characterLevel:    characterLevel,
		skillLevel:        skillLevel,
		berserkDarkForce:  berserkDarkForce,
		dragonFuryCreate:  dragonFuryCreate,
		monsterMagnetLeft: monsterMagnetLeft,
		isBerserk:         isBerserk,
		isDragonFury:      isDragonFury,
		isMonsterMagnet:   isMonsterMagnet,
	}
}

func (m EffectSkillUse) Mode() byte              { return m.mode }
func (m EffectSkillUse) SkillId() uint32          { return m.skillId }
func (m EffectSkillUse) CharacterLevel() byte     { return m.characterLevel }
func (m EffectSkillUse) SkillLevel() byte         { return m.skillLevel }
func (m EffectSkillUse) BerserkDarkForce() bool   { return m.berserkDarkForce }
func (m EffectSkillUse) DragonFuryCreate() bool   { return m.dragonFuryCreate }
func (m EffectSkillUse) MonsterMagnetLeft() bool  { return m.monsterMagnetLeft }
func (m EffectSkillUse) IsBerserk() bool          { return m.isBerserk }
func (m EffectSkillUse) IsDragonFury() bool       { return m.isDragonFury }
func (m EffectSkillUse) IsMonsterMagnet() bool    { return m.isMonsterMagnet }
func (m EffectSkillUse) Operation() string        { return CharacterEffectWriter }

func (m EffectSkillUse) String() string {
	return fmt.Sprintf("skill use skillId [%d] level [%d]", m.skillId, m.skillLevel)
}

func (m EffectSkillUse) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteInt(m.skillId)
		w.WriteByte(m.characterLevel)
		w.WriteByte(m.skillLevel)
		if m.isBerserk {
			w.WriteBool(m.berserkDarkForce)
		}
		if m.isDragonFury {
			w.WriteBool(m.dragonFuryCreate)
		}
		if m.isMonsterMagnet {
			w.WriteBool(m.monsterMagnetLeft)
		}
		return w.Bytes()
	}
}

func (m *EffectSkillUse) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		// No-op: server-send-only
	}
}
