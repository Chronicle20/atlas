package handler

import (
	quest2 "atlas-channel/data/quest"
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
	QuestActionRestoreLostItem byte = 0 // CQuest::OnCompleteQuestFailed
	QuestActionStart           byte = 1 // CQuest::StartQuest
	QuestActionComplete        byte = 2 // CQuest::StartQuest
	QuestActionForfeit         byte = 3 // CWvsContext::ResignQuest
	QuestActionScriptStart     byte = 4 // CQuest::StartQuest
	QuestActionScriptEnd       byte = 5 // CQuest::StartQuest
)

func QuestActionHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		action := r.ReadByte()
		questId := uint32(r.ReadUint16())

		q, err := quest2.NewProcessor(l, ctx).GetById(questId)
		if err != nil {
			l.WithError(err).Errorf("Failed to get quest [%d] for character [%d] quest action [%d].", questId, s.CharacterId(), action)
			return
		}

		switch action {
		case QuestActionStart:
			npcId := r.ReadUint32()
			x := int16(-1)
			y := int16(-1)
			if q.AutoStart() {
				x = r.ReadInt16()
				y = r.ReadInt16()
			}
			l.Debugf("Character [%d] starting quest [%d] conversation with NPC [%d]. x,y [%d,%d]", s.CharacterId(), questId, npcId, x, y)
			err := quest.NewProcessor(l, ctx).StartQuest(s.Field(), s.CharacterId(), questId, npcId, false)
			if err != nil {
				l.WithError(err).Errorf("Failed to start quest [%d] conversation for character [%d] with NPC [%d].", questId, s.CharacterId(), npcId)
			}
			return
		case QuestActionScriptStart:
			npcId := r.ReadUint32()
			x := r.ReadInt16()
			y := r.ReadInt16()
			l.Debugf("Character [%d] starting scripted quest [%d] conversation with NPC [%d]. x,y [%d,%d]", s.CharacterId(), questId, npcId, x, y)
			err := quest.NewProcessor(l, ctx).StartQuestConversation(s.Field(), questId, npcId, s.CharacterId())
			if err != nil {
				l.WithError(err).Errorf("Failed to start quest [%d] conversation for character [%d] with NPC [%d].", questId, s.CharacterId(), npcId)
			}
			return
		case QuestActionComplete:
			npcId := r.ReadUint32()
			x := int16(-1)
			y := int16(-1)
			if q.AutoStart() {
				x = r.ReadInt16()
				y = r.ReadInt16()
			}
			selection := r.ReadInt32()
			l.Debugf("Character [%d] completing quest [%d] conversation with NPC [%d]. x,y [%d,%d]", s.CharacterId(), questId, npcId, x, y)
			err := quest.NewProcessor(l, ctx).CompleteQuest(s.Field(), s.CharacterId(), questId, npcId, selection, false)
			if err != nil {
				l.WithError(err).Errorf("Failed to start quest [%d] completion conversation for character [%d] with NPC [%d].", questId, s.CharacterId(), npcId)
			}
			return
		case QuestActionScriptEnd:
			npcId := r.ReadUint32()
			x := r.ReadInt16()
			y := r.ReadInt16()
			l.Debugf("Character [%d] completing scripted quest [%d] conversation with NPC [%d]. x,y [%d,%d]", s.CharacterId(), questId, npcId, x, y)
			err := quest.NewProcessor(l, ctx).StartQuestConversation(s.Field(), questId, npcId, s.CharacterId())
			if err != nil {
				l.WithError(err).Errorf("Failed to start quest [%d] completion conversation for character [%d] with NPC [%d].", questId, s.CharacterId(), npcId)
			}
			return
		case QuestActionForfeit:
			l.Debugf("Character [%d] forfeiting quest [%d].", s.CharacterId(), questId)
			err := quest.NewProcessor(l, ctx).ForfeitQuest(s.Field(), s.CharacterId(), questId)
			if err != nil {
				l.WithError(err).Errorf("Failed to forfeit quest [%d] for character [%d].", questId, s.CharacterId())
			}
			return
		case QuestActionRestoreLostItem:
			unk1 := r.ReadUint32()
			itemId := r.ReadUint32()
			l.Debugf("Character [%d] restoring lost item [%d] for quest [%d]. unk1 [%d]. rem [%d]", s.CharacterId(), itemId, questId, unk1, r.Available())
			err := quest.NewProcessor(l, ctx).RestoreItem(s.Field(), s.CharacterId(), questId, itemId)
			if err != nil {
				l.WithError(err).Errorf("Failed to restore item [%d] for quest [%d] for character [%d].", itemId, questId, s.CharacterId())
			}
			return
		}
		l.Warnf("Character [%d] sent unknown quest action [%d] for quest [%d].", s.CharacterId(), action, questId)
	}
}
