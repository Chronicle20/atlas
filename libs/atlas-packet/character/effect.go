package character

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const CharacterEffectWriter = "CharacterEffect"

// EffectSimple - mode only (LevelUp, PlayPortalSound, JobChanged, QuestComplete, MonsterBookCardGet, ItemLevelUp, SoulStoneUse)
type EffectSimple struct {
	mode byte
}

func NewEffectSimple(mode byte) EffectSimple {
	return EffectSimple{mode: mode}
}

func (m EffectSimple) Mode() byte       { return m.mode }
func (m EffectSimple) Operation() string { return CharacterEffectWriter }
func (m EffectSimple) String() string    { return fmt.Sprintf("effect mode [%d]", m.mode) }

func (m EffectSimple) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		return w.Bytes()
	}
}

func (m *EffectSimple) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
	}
}

// EffectSimpleForeign - characterId + mode (LevelUp, PlayPortalSound, etc.)
type EffectSimpleForeign struct {
	characterId uint32
	mode        byte
}

func NewEffectSimpleForeign(characterId uint32, mode byte) EffectSimpleForeign {
	return EffectSimpleForeign{characterId: characterId, mode: mode}
}

func (m EffectSimpleForeign) CharacterId() uint32 { return m.characterId }
func (m EffectSimpleForeign) Mode() byte          { return m.mode }
func (m EffectSimpleForeign) Operation() string    { return CharacterEffectWriter }
func (m EffectSimpleForeign) String() string {
	return fmt.Sprintf("foreign effect characterId [%d] mode [%d]", m.characterId, m.mode)
}

func (m EffectSimpleForeign) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.characterId)
		w.WriteByte(m.mode)
		return w.Bytes()
	}
}

func (m *EffectSimpleForeign) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.characterId = r.ReadUint32()
		m.mode = r.ReadByte()
	}
}

// EffectSkillAffected - mode, skillId, skillLevel
type EffectSkillAffected struct {
	mode       byte
	skillId    uint32
	skillLevel byte
}

func NewEffectSkillAffected(mode byte, skillId uint32, skillLevel byte) EffectSkillAffected {
	return EffectSkillAffected{mode: mode, skillId: skillId, skillLevel: skillLevel}
}

func (m EffectSkillAffected) Mode() byte        { return m.mode }
func (m EffectSkillAffected) SkillId() uint32    { return m.skillId }
func (m EffectSkillAffected) SkillLevel() byte   { return m.skillLevel }
func (m EffectSkillAffected) Operation() string  { return CharacterEffectWriter }
func (m EffectSkillAffected) String() string {
	return fmt.Sprintf("skill affected skillId [%d] level [%d]", m.skillId, m.skillLevel)
}

func (m EffectSkillAffected) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteInt(m.skillId)
		w.WriteByte(m.skillLevel)
		return w.Bytes()
	}
}

func (m *EffectSkillAffected) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.skillId = r.ReadUint32()
		m.skillLevel = r.ReadByte()
	}
}

// EffectPet - mode, effectType, petIndex
type EffectPet struct {
	mode       byte
	effectType byte
	petIndex   byte
}

func NewEffectPet(mode byte, effectType byte, petIndex byte) EffectPet {
	return EffectPet{mode: mode, effectType: effectType, petIndex: petIndex}
}

func (m EffectPet) Mode() byte        { return m.mode }
func (m EffectPet) EffectType() byte  { return m.effectType }
func (m EffectPet) PetIndex() byte    { return m.petIndex }
func (m EffectPet) Operation() string { return CharacterEffectWriter }
func (m EffectPet) String() string {
	return fmt.Sprintf("pet effect type [%d] index [%d]", m.effectType, m.petIndex)
}

func (m EffectPet) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(m.effectType)
		w.WriteByte(m.petIndex)
		return w.Bytes()
	}
}

func (m *EffectPet) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.effectType = r.ReadByte()
		m.petIndex = r.ReadByte()
	}
}

// EffectWithId - mode, id (used for SkillSpecial, BuffItem, ConsumeEffect)
type EffectWithId struct {
	mode byte
	id   uint32
}

func NewEffectWithId(mode byte, id uint32) EffectWithId {
	return EffectWithId{mode: mode, id: id}
}

func (m EffectWithId) Mode() byte       { return m.mode }
func (m EffectWithId) Id() uint32       { return m.id }
func (m EffectWithId) Operation() string { return CharacterEffectWriter }
func (m EffectWithId) String() string {
	return fmt.Sprintf("effect mode [%d] id [%d]", m.mode, m.id)
}

func (m EffectWithId) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteInt(m.id)
		return w.Bytes()
	}
}

func (m *EffectWithId) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.id = r.ReadUint32()
	}
}

// EffectWithMessage - mode, message (ShowIntro, Reserved, Battlefield)
type EffectWithMessage struct {
	mode    byte
	message string
}

func NewEffectWithMessage(mode byte, message string) EffectWithMessage {
	return EffectWithMessage{mode: mode, message: message}
}

func (m EffectWithMessage) Mode() byte       { return m.mode }
func (m EffectWithMessage) Message() string   { return m.message }
func (m EffectWithMessage) Operation() string { return CharacterEffectWriter }
func (m EffectWithMessage) String() string {
	return fmt.Sprintf("effect mode [%d] message [%s]", m.mode, m.message)
}

func (m EffectWithMessage) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteAsciiString(m.message)
		return w.Bytes()
	}
}

func (m *EffectWithMessage) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.message = r.ReadAsciiString()
	}
}

// EffectProtectOnDie - mode, safetyCharm, usesRemaining, days, itemId (only if !safetyCharm)
type EffectProtectOnDie struct {
	mode          byte
	safetyCharm   bool
	usesRemaining byte
	days          byte
	itemId        uint32
}

func NewEffectProtectOnDie(mode byte, safetyCharm bool, usesRemaining byte, days byte, itemId uint32) EffectProtectOnDie {
	return EffectProtectOnDie{mode: mode, safetyCharm: safetyCharm, usesRemaining: usesRemaining, days: days, itemId: itemId}
}

func (m EffectProtectOnDie) Mode() byte          { return m.mode }
func (m EffectProtectOnDie) SafetyCharm() bool    { return m.safetyCharm }
func (m EffectProtectOnDie) UsesRemaining() byte  { return m.usesRemaining }
func (m EffectProtectOnDie) Days() byte           { return m.days }
func (m EffectProtectOnDie) ItemId() uint32       { return m.itemId }
func (m EffectProtectOnDie) Operation() string    { return CharacterEffectWriter }
func (m EffectProtectOnDie) String() string {
	return fmt.Sprintf("protect on die safetyCharm [%v]", m.safetyCharm)
}

func (m EffectProtectOnDie) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteBool(m.safetyCharm)
		w.WriteByte(m.usesRemaining)
		w.WriteByte(m.days)
		if !m.safetyCharm {
			w.WriteInt(m.itemId)
		}
		return w.Bytes()
	}
}

func (m *EffectProtectOnDie) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.safetyCharm = r.ReadBool()
		m.usesRemaining = r.ReadByte()
		m.days = r.ReadByte()
		if !m.safetyCharm {
			m.itemId = r.ReadUint32()
		}
	}
}

// EffectIncDecHP - mode, delta int8
type EffectIncDecHP struct {
	mode  byte
	delta int8
}

func NewEffectIncDecHP(mode byte, delta int8) EffectIncDecHP {
	return EffectIncDecHP{mode: mode, delta: delta}
}

func (m EffectIncDecHP) Mode() byte       { return m.mode }
func (m EffectIncDecHP) Delta() int8      { return m.delta }
func (m EffectIncDecHP) Operation() string { return CharacterEffectWriter }
func (m EffectIncDecHP) String() string {
	return fmt.Sprintf("inc dec hp delta [%d]", m.delta)
}

func (m EffectIncDecHP) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteInt8(m.delta)
		return w.Bytes()
	}
}

func (m *EffectIncDecHP) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.delta = r.ReadInt8()
	}
}

// EffectShowInfo - mode, path, unk(1)
type EffectShowInfo struct {
	mode byte
	path string
}

func NewEffectShowInfo(mode byte, path string) EffectShowInfo {
	return EffectShowInfo{mode: mode, path: path}
}

func (m EffectShowInfo) Mode() byte       { return m.mode }
func (m EffectShowInfo) Path() string     { return m.path }
func (m EffectShowInfo) Operation() string { return CharacterEffectWriter }
func (m EffectShowInfo) String() string {
	return fmt.Sprintf("show info path [%s]", m.path)
}

func (m EffectShowInfo) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteAsciiString(m.path)
		w.WriteInt(1)
		return w.Bytes()
	}
}

func (m *EffectShowInfo) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.path = r.ReadAsciiString()
		_ = r.ReadUint32()
	}
}

// EffectLotteryUse - mode, itemId, success, message (if success)
type EffectLotteryUse struct {
	mode    byte
	itemId  uint32
	success bool
	message string
}

func NewEffectLotteryUse(mode byte, itemId uint32, success bool, message string) EffectLotteryUse {
	return EffectLotteryUse{mode: mode, itemId: itemId, success: success, message: message}
}

func (m EffectLotteryUse) Mode() byte       { return m.mode }
func (m EffectLotteryUse) ItemId() uint32   { return m.itemId }
func (m EffectLotteryUse) Success() bool    { return m.success }
func (m EffectLotteryUse) Message() string  { return m.message }
func (m EffectLotteryUse) Operation() string { return CharacterEffectWriter }
func (m EffectLotteryUse) String() string {
	return fmt.Sprintf("lottery use itemId [%d] success [%v]", m.itemId, m.success)
}

func (m EffectLotteryUse) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteInt(m.itemId)
		w.WriteBool(m.success)
		if m.success {
			w.WriteAsciiString(m.message)
		}
		return w.Bytes()
	}
}

func (m *EffectLotteryUse) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.itemId = r.ReadUint32()
		m.success = r.ReadBool()
		if m.success {
			m.message = r.ReadAsciiString()
		}
	}
}

// EffectItemMaker - mode, state
type EffectItemMaker struct {
	mode  byte
	state uint32
}

func NewEffectItemMaker(mode byte, state uint32) EffectItemMaker {
	return EffectItemMaker{mode: mode, state: state}
}

func (m EffectItemMaker) Mode() byte       { return m.mode }
func (m EffectItemMaker) State() uint32    { return m.state }
func (m EffectItemMaker) Operation() string { return CharacterEffectWriter }
func (m EffectItemMaker) String() string {
	return fmt.Sprintf("item maker state [%d]", m.state)
}

func (m EffectItemMaker) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteInt(m.state)
		return w.Bytes()
	}
}

func (m *EffectItemMaker) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.state = r.ReadUint32()
	}
}

// EffectUpgradeTomb - mode, usesRemaining
type EffectUpgradeTomb struct {
	mode          byte
	usesRemaining byte
}

func NewEffectUpgradeTomb(mode byte, usesRemaining byte) EffectUpgradeTomb {
	return EffectUpgradeTomb{mode: mode, usesRemaining: usesRemaining}
}

func (m EffectUpgradeTomb) Mode() byte          { return m.mode }
func (m EffectUpgradeTomb) UsesRemaining() byte { return m.usesRemaining }
func (m EffectUpgradeTomb) Operation() string    { return CharacterEffectWriter }
func (m EffectUpgradeTomb) String() string {
	return fmt.Sprintf("upgrade tomb uses [%d]", m.usesRemaining)
}

func (m EffectUpgradeTomb) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(m.usesRemaining)
		return w.Bytes()
	}
}

func (m *EffectUpgradeTomb) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.usesRemaining = r.ReadByte()
	}
}

// EffectIncubatorUse - mode, itemId, message
type EffectIncubatorUse struct {
	mode    byte
	itemId  uint32
	message string
}

func NewEffectIncubatorUse(mode byte, itemId uint32, message string) EffectIncubatorUse {
	return EffectIncubatorUse{mode: mode, itemId: itemId, message: message}
}

func (m EffectIncubatorUse) Mode() byte       { return m.mode }
func (m EffectIncubatorUse) ItemId() uint32   { return m.itemId }
func (m EffectIncubatorUse) Message() string  { return m.message }
func (m EffectIncubatorUse) Operation() string { return CharacterEffectWriter }
func (m EffectIncubatorUse) String() string {
	return fmt.Sprintf("incubator use itemId [%d]", m.itemId)
}

func (m EffectIncubatorUse) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteInt(m.itemId)
		w.WriteAsciiString(m.message)
		return w.Bytes()
	}
}

func (m *EffectIncubatorUse) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.itemId = r.ReadUint32()
		m.message = r.ReadAsciiString()
	}
}
