package handler

import (
	"atlas-channel/quest"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

const QuestActionHandle = "QuestActionHandle"

// Quest action types
const (
	QuestActionRestoreLostItem byte = 0
	QuestActionStart           byte = 1
	QuestActionComplete        byte = 2
	QuestActionForfeit         byte = 3
	QuestActionScriptStart     byte = 4
	QuestActionScriptEnd       byte = 5
)

func QuestActionHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		action := r.ReadByte()
		questId := uint32(r.ReadUint16())

		switch action {
		case QuestActionStart, QuestActionScriptStart:
			npcId := r.ReadUint32()
			l.Debugf("Character [%d] starting quest [%d] conversation with NPC [%d].", s.CharacterId(), questId, npcId)
			err := quest.NewProcessor(l, ctx).StartQuestConversation(s.Map(), questId, npcId, s.CharacterId())
			if err != nil {
				l.WithError(err).Errorf("Failed to start quest [%d] conversation for character [%d] with NPC [%d].", questId, s.CharacterId(), npcId)
			}

		case QuestActionComplete, QuestActionScriptEnd:
			npcId := r.ReadUint32()
			l.Debugf("Character [%d] completing quest [%d] conversation with NPC [%d].", s.CharacterId(), questId, npcId)
			// Quest completion is handled through the NPC conversation state machine
			// The complete_quest operation will be triggered by the conversation flow
			err := quest.NewProcessor(l, ctx).StartQuestConversation(s.Map(), questId, npcId, s.CharacterId())
			if err != nil {
				l.WithError(err).Errorf("Failed to start quest [%d] completion conversation for character [%d] with NPC [%d].", questId, s.CharacterId(), npcId)
			}

		case QuestActionForfeit:
			l.Debugf("Character [%d] forfeiting quest [%d].", s.CharacterId(), questId)
			// Quest forfeit is handled by atlas-quest service directly
			// TODO: Implement forfeit command producer when needed

		case QuestActionRestoreLostItem:
			l.Debugf("Character [%d] restoring lost item for quest [%d].", s.CharacterId(), questId)
			// Lost item restoration is handled by atlas-quest service directly
			// TODO: Implement restore command producer when needed

		default:
			l.Warnf("Character [%d] sent unknown quest action [%d] for quest [%d].", s.CharacterId(), action, questId)
		}
	}
}
