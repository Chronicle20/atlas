package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// effectSkillUseIncludesCharacterLevel reports whether the SHOW_SKILL_USE_EFFECT
// arm (CUser::OnEffect case 1) wire carries the caster-level byte before
// skillLevel. IDA-verified: v79 CUser::OnEffect @0x89112c case 1 reads only
// skillId (Decode4 @0x891225) + one byte (Decode1 @0x89122f) — the single byte
// feeds SKILLENTRY::IsActionAppointed @0x89125a, i.e. it is the skill level
// (v83's nSLV). v83 CUser::OnEffect @0x9377d9 case 1 reads an EXTRA characterLevel
// byte FIRST (Decode1 @0x9378d4, stored to this+0x2AE0) before skillLevel
// (Decode1 @0x9378f1 -> IsActionAppointed @0x93791c). So the caster-level byte was
// introduced at v83; legacy GMS (< 83) omits it. Leave v83/84/87/95 and JMS
// unchanged.
func effectSkillUseIncludesCharacterLevel(t tenant.Model) bool {
	return !(t.Region() == "GMS" && t.MajorVersion() < 83)
}

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

// NewEffectSkillUseForDecode creates an EffectSkillUse with the constructor flags
// needed to drive non-self-describing Decode branches.
func NewEffectSkillUseForDecode(isBerserk bool, isDragonFury bool, isMonsterMagnet bool) EffectSkillUse {
	return EffectSkillUse{
		isBerserk:       isBerserk,
		isDragonFury:    isDragonFury,
		isMonsterMagnet: isMonsterMagnet,
	}
}

func (m EffectSkillUse) Mode() byte              { return m.mode }
func (m EffectSkillUse) SkillId() uint32         { return m.skillId }
func (m EffectSkillUse) CharacterLevel() byte    { return m.characterLevel }
func (m EffectSkillUse) SkillLevel() byte        { return m.skillLevel }
func (m EffectSkillUse) BerserkDarkForce() bool  { return m.berserkDarkForce }
func (m EffectSkillUse) DragonFuryCreate() bool  { return m.dragonFuryCreate }
func (m EffectSkillUse) MonsterMagnetLeft() bool { return m.monsterMagnetLeft }
func (m EffectSkillUse) IsBerserk() bool         { return m.isBerserk }
func (m EffectSkillUse) IsDragonFury() bool      { return m.isDragonFury }
func (m EffectSkillUse) IsMonsterMagnet() bool   { return m.isMonsterMagnet }
func (m EffectSkillUse) Operation() string       { return CharacterEffectWriter }

func (m EffectSkillUse) String() string {
	return fmt.Sprintf("skill use skillId [%d] level [%d]", m.skillId, m.skillLevel)
}

func (m EffectSkillUse) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteInt(m.skillId)
		if effectSkillUseIncludesCharacterLevel(t) {
			w.WriteByte(m.characterLevel)
		}
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

func (m *EffectSkillUse) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.skillId = r.ReadUint32()
		if effectSkillUseIncludesCharacterLevel(t) {
			m.characterLevel = r.ReadByte()
		}
		m.skillLevel = r.ReadByte()
		if m.isBerserk {
			m.berserkDarkForce = r.ReadBool()
		}
		if m.isDragonFury {
			m.dragonFuryCreate = r.ReadBool()
		}
		if m.isMonsterMagnet {
			m.monsterMagnetLeft = r.ReadBool()
		}
	}
}

// EffectSkillUseForeign - characterId + mode, skillId, characterLevel, skillLevel + conditional berserk/dragonFury/monsterMagnet
type EffectSkillUseForeign struct {
	characterId       uint32
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

func NewEffectSkillUseForeign(characterId uint32, mode byte, skillId uint32, characterLevel byte, skillLevel byte, isBerserk bool, berserkDarkForce bool, isDragonFury bool, dragonFuryCreate bool, isMonsterMagnet bool, monsterMagnetLeft bool) EffectSkillUseForeign {
	return EffectSkillUseForeign{
		characterId:       characterId,
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

// NewEffectSkillUseForeignForDecode creates an EffectSkillUseForeign with the
// constructor flags needed to drive non-self-describing Decode branches.
func NewEffectSkillUseForeignForDecode(isBerserk bool, isDragonFury bool, isMonsterMagnet bool) EffectSkillUseForeign {
	return EffectSkillUseForeign{
		isBerserk:       isBerserk,
		isDragonFury:    isDragonFury,
		isMonsterMagnet: isMonsterMagnet,
	}
}

func (m EffectSkillUseForeign) CharacterId() uint32     { return m.characterId }
func (m EffectSkillUseForeign) Mode() byte              { return m.mode }
func (m EffectSkillUseForeign) SkillId() uint32         { return m.skillId }
func (m EffectSkillUseForeign) CharacterLevel() byte    { return m.characterLevel }
func (m EffectSkillUseForeign) SkillLevel() byte        { return m.skillLevel }
func (m EffectSkillUseForeign) BerserkDarkForce() bool  { return m.berserkDarkForce }
func (m EffectSkillUseForeign) DragonFuryCreate() bool  { return m.dragonFuryCreate }
func (m EffectSkillUseForeign) MonsterMagnetLeft() bool { return m.monsterMagnetLeft }
func (m EffectSkillUseForeign) IsBerserk() bool         { return m.isBerserk }
func (m EffectSkillUseForeign) IsDragonFury() bool      { return m.isDragonFury }
func (m EffectSkillUseForeign) IsMonsterMagnet() bool   { return m.isMonsterMagnet }
func (m EffectSkillUseForeign) Operation() string       { return CharacterEffectWriter }

func (m EffectSkillUseForeign) String() string {
	return fmt.Sprintf("foreign skill use characterId [%d] skillId [%d] level [%d]", m.characterId, m.skillId, m.skillLevel)
}

func (m EffectSkillUseForeign) Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.characterId)
		w.WriteByte(m.mode)
		w.WriteInt(m.skillId)
		if effectSkillUseIncludesCharacterLevel(t) {
			w.WriteByte(m.characterLevel)
		}
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

func (m *EffectSkillUseForeign) Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{}) {
	t := tenant.MustFromContext(ctx)
	return func(r *request.Reader, options map[string]interface{}) {
		m.characterId = r.ReadUint32()
		m.mode = r.ReadByte()
		m.skillId = r.ReadUint32()
		if effectSkillUseIncludesCharacterLevel(t) {
			m.characterLevel = r.ReadByte()
		}
		m.skillLevel = r.ReadByte()
		if m.isBerserk {
			m.berserkDarkForce = r.ReadBool()
		}
		if m.isDragonFury {
			m.dragonFuryCreate = r.ReadBool()
		}
		if m.isMonsterMagnet {
			m.monsterMagnetLeft = r.ReadBool()
		}
	}
}
