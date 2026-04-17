package clientbound

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

const FieldEffectWriter = "FieldEffect"

type EffectSummon struct {
	mode   byte
	effect byte
	x      int32
	y      int32
}

func NewFieldEffectSummon(mode byte, effect byte, x int32, y int32) EffectSummon {
	return EffectSummon{mode: mode, effect: effect, x: x, y: y}
}

func (m EffectSummon) Operation() string { return FieldEffectWriter }
func (m EffectSummon) String() string {
	return fmt.Sprintf("summon effect [%d]", m.effect)
}

func (m EffectSummon) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(m.effect)
		w.WriteInt32(m.x)
		w.WriteInt32(m.y)
		return w.Bytes()
	}
}

func (m *EffectSummon) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.effect = r.ReadByte()
		m.x = r.ReadInt32()
		m.y = r.ReadInt32()
	}
}

type EffectTremble struct {
	mode                  byte
	bHeavyNShortTremble   bool
	delay                 uint32
}

func NewFieldEffectTremble(mode byte, bHeavyNShortTremble bool, delay uint32) EffectTremble {
	return EffectTremble{mode: mode, bHeavyNShortTremble: bHeavyNShortTremble, delay: delay}
}

func (m EffectTremble) Operation() string { return FieldEffectWriter }
func (m EffectTremble) String() string    { return fmt.Sprintf("tremble delay [%d]", m.delay) }

func (m EffectTremble) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteBool(m.bHeavyNShortTremble)
		w.WriteInt(m.delay)
		return w.Bytes()
	}
}

func (m *EffectTremble) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.bHeavyNShortTremble = r.ReadBool()
		m.delay = r.ReadUint32()
	}
}

type EffectString struct {
	mode byte
	name string
}

func NewFieldEffectObject(mode byte, name string) EffectString {
	return EffectString{mode: mode, name: name}
}

func NewFieldEffectScreen(mode byte, path string) EffectString {
	return EffectString{mode: mode, name: path}
}

func NewFieldEffectSound(mode byte, path string) EffectString {
	return EffectString{mode: mode, name: path}
}

func NewFieldEffectBackgroundMusic(mode byte, name string) EffectString {
	return EffectString{mode: mode, name: name}
}

func (m EffectString) Operation() string { return FieldEffectWriter }
func (m EffectString) String() string    { return fmt.Sprintf("mode [%d], name [%s]", m.mode, m.name) }

func (m EffectString) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteAsciiString(m.name)
		return w.Bytes()
	}
}

func (m *EffectString) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.name = r.ReadAsciiString()
	}
}

type EffectBossHp struct {
	mode               byte
	monsterId          uint32
	currentHp          uint32
	maxHp              uint32
	tagColor           byte
	tagBackgroundColor byte
}

func NewFieldEffectBossHp(mode byte, monsterId uint32, currentHp uint32, maxHp uint32, tagColor byte, tagBackgroundColor byte) EffectBossHp {
	return EffectBossHp{mode: mode, monsterId: monsterId, currentHp: currentHp, maxHp: maxHp, tagColor: tagColor, tagBackgroundColor: tagBackgroundColor}
}

func (m EffectBossHp) Operation() string { return FieldEffectWriter }
func (m EffectBossHp) String() string {
	return fmt.Sprintf("bossHp monsterId [%d], hp [%d/%d]", m.monsterId, m.currentHp, m.maxHp)
}

func (m EffectBossHp) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteInt(m.monsterId)
		w.WriteInt(m.currentHp)
		w.WriteInt(m.maxHp)
		w.WriteByte(m.tagColor)
		w.WriteByte(m.tagBackgroundColor)
		return w.Bytes()
	}
}

func (m *EffectBossHp) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.monsterId = r.ReadUint32()
		m.currentHp = r.ReadUint32()
		m.maxHp = r.ReadUint32()
		m.tagColor = r.ReadByte()
		m.tagBackgroundColor = r.ReadByte()
	}
}

type EffectRewardRullet struct {
	mode            byte
	nRewardJobIdx   uint32
	nRewardPartIdx  uint32
	nRewardLevIdx   uint32
}

func NewFieldEffectRewardRullet(mode byte, nRewardJobIdx uint32, nRewardPartIdx uint32, nRewardLevIdx uint32) EffectRewardRullet {
	return EffectRewardRullet{mode: mode, nRewardJobIdx: nRewardJobIdx, nRewardPartIdx: nRewardPartIdx, nRewardLevIdx: nRewardLevIdx}
}

func (m EffectRewardRullet) Operation() string { return FieldEffectWriter }
func (m EffectRewardRullet) String() string {
	return fmt.Sprintf("rewardRullet job [%d]", m.nRewardJobIdx)
}

func (m EffectRewardRullet) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteInt(m.nRewardJobIdx)
		w.WriteInt(m.nRewardPartIdx)
		w.WriteInt(m.nRewardLevIdx)
		return w.Bytes()
	}
}

func (m *EffectRewardRullet) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		m.nRewardJobIdx = r.ReadUint32()
		m.nRewardPartIdx = r.ReadUint32()
		m.nRewardLevIdx = r.ReadUint32()
	}
}
