package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

const SetQuestTimeWriter = "SetQuestTime"

// QuestTime is one timed-quest entry: the quest id plus its 8-byte FILETIME
// start and end timestamps.
type QuestTime struct {
	questId   uint32
	startTime uint64
	endTime   uint64
}

func NewQuestTime(questId uint32, startTime uint64, endTime uint64) QuestTime {
	return QuestTime{questId: questId, startTime: startTime, endTime: endTime}
}

func (q QuestTime) QuestId() uint32   { return q.questId }
func (q QuestTime) StartTime() uint64 { return q.startTime }
func (q QuestTime) EndTime() uint64   { return q.endTime }

// SetQuestTime is the clientbound CField::OnSetQuestTime packet. A count byte
// followed by that many {questId, startTime(FILETIME), endTime(FILETIME)} entries.
// packet-audit:fname CField::OnSetQuestTime
type SetQuestTime struct {
	quests []QuestTime
}

func NewSetQuestTime(quests []QuestTime) SetQuestTime {
	return SetQuestTime{quests: quests}
}

func (m SetQuestTime) Quests() []QuestTime { return m.quests }

func (m SetQuestTime) Operation() string { return SetQuestTimeWriter }
func (m SetQuestTime) String() string {
	return fmt.Sprintf("quests [%d]", len(m.quests))
}

func (m SetQuestTime) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(byte(len(m.quests)))
		for _, q := range m.quests {
			w.WriteInt(q.questId)
			w.WriteLong(q.startTime)
			w.WriteLong(q.endTime)
		}
		return w.Bytes()
	}
}

func (m *SetQuestTime) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		count := r.ReadByte()
		m.quests = make([]QuestTime, 0, count)
		for i := byte(0); i < count; i++ {
			questId := r.ReadUint32()
			startTime := r.ReadUint64()
			endTime := r.ReadUint64()
			m.quests = append(m.quests, QuestTime{questId: questId, startTime: startTime, endTime: endTime})
		}
	}
}
