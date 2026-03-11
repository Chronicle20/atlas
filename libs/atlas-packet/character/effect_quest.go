package character

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas-socket/request"
	"github.com/Chronicle20/atlas-socket/response"
	"github.com/sirupsen/logrus"
)

// QuestReward represents an item reward from a quest effect.
type QuestReward struct {
	ItemId uint32
	Amount int32
}

// EffectQuest - mode, rewards, message, nEffect
type EffectQuest struct {
	mode    byte
	rewards []QuestReward
	message string
	nEffect uint32
}

func NewEffectQuest(mode byte, message string, nEffect uint32, rewards []QuestReward) EffectQuest {
	return EffectQuest{mode: mode, message: message, nEffect: nEffect, rewards: rewards}
}

func (m EffectQuest) Mode() byte            { return m.mode }
func (m EffectQuest) Rewards() []QuestReward { return m.rewards }
func (m EffectQuest) Message() string        { return m.message }
func (m EffectQuest) NEffect() uint32        { return m.nEffect }
func (m EffectQuest) Operation() string      { return CharacterEffectWriter }

func (m EffectQuest) String() string {
	return fmt.Sprintf("quest effect rewards [%d] message [%s]", len(m.rewards), m.message)
}

func (m EffectQuest) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(m.mode)
		w.WriteByte(byte(len(m.rewards)))
		if len(m.rewards) == 0 {
			w.WriteAsciiString(m.message)
			w.WriteInt(m.nEffect)
		} else {
			for _, reward := range m.rewards {
				w.WriteInt(reward.ItemId)
				w.WriteInt32(reward.Amount)
			}
		}
		return w.Bytes()
	}
}

func (m *EffectQuest) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.mode = r.ReadByte()
		count := r.ReadByte()
		if count == 0 {
			m.message = r.ReadAsciiString()
			m.nEffect = r.ReadUint32()
		} else {
			m.rewards = make([]QuestReward, count)
			for i := byte(0); i < count; i++ {
				m.rewards[i].ItemId = r.ReadUint32()
				m.rewards[i].Amount = r.ReadInt32()
			}
		}
	}
}

// EffectQuestForeign - characterId + mode, rewards, message, nEffect
type EffectQuestForeign struct {
	characterId uint32
	mode        byte
	rewards     []QuestReward
	message     string
	nEffect     uint32
}

func NewEffectQuestForeign(characterId uint32, mode byte, message string, nEffect uint32, rewards []QuestReward) EffectQuestForeign {
	return EffectQuestForeign{characterId: characterId, mode: mode, message: message, nEffect: nEffect, rewards: rewards}
}

func (m EffectQuestForeign) CharacterId() uint32  { return m.characterId }
func (m EffectQuestForeign) Mode() byte            { return m.mode }
func (m EffectQuestForeign) Rewards() []QuestReward { return m.rewards }
func (m EffectQuestForeign) Message() string        { return m.message }
func (m EffectQuestForeign) NEffect() uint32        { return m.nEffect }
func (m EffectQuestForeign) Operation() string      { return CharacterEffectWriter }

func (m EffectQuestForeign) String() string {
	return fmt.Sprintf("foreign quest effect characterId [%d] rewards [%d] message [%s]", m.characterId, len(m.rewards), m.message)
}

func (m EffectQuestForeign) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteInt(m.characterId)
		w.WriteByte(m.mode)
		w.WriteByte(byte(len(m.rewards)))
		if len(m.rewards) == 0 {
			w.WriteAsciiString(m.message)
			w.WriteInt(m.nEffect)
		} else {
			for _, reward := range m.rewards {
				w.WriteInt(reward.ItemId)
				w.WriteInt32(reward.Amount)
			}
		}
		return w.Bytes()
	}
}

func (m *EffectQuestForeign) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		m.characterId = r.ReadUint32()
		m.mode = r.ReadByte()
		count := r.ReadByte()
		if count == 0 {
			m.message = r.ReadAsciiString()
			m.nEffect = r.ReadUint32()
		} else {
			m.rewards = make([]QuestReward, count)
			for i := byte(0); i < count; i++ {
				m.rewards[i].ItemId = r.ReadUint32()
				m.rewards[i].Amount = r.ReadInt32()
			}
		}
	}
}
