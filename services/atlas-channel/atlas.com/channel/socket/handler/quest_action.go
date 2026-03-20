package handler

import (
	quest2 "atlas-channel/data/quest"
	"atlas-channel/quest"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	quest3 "github.com/Chronicle20/atlas-packet/quest/serverbound"
	"github.com/Chronicle20/atlas-socket/request"
	"github.com/sirupsen/logrus"
)

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
		p := quest3.Action{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())
		action := p.ActionType()
		questId := uint32(p.QuestId())

		q, err := quest2.NewProcessor(l, ctx).GetById(questId)
		if err != nil {
			l.WithError(err).Errorf("Failed to get quest [%d] for character [%d] quest action [%d].", questId, s.CharacterId(), action)
			return
		}

		switch action {
		case QuestActionStart:
			sp := quest3.NewActionStart(q.AutoStart())
			sp.Decode(l, ctx)(r, readerOptions)
			l.Debugf("Character [%d] starting quest [%d] conversation with NPC [%d]. x,y [%d,%d]", s.CharacterId(), questId, sp.NpcId(), sp.X(), sp.Y())
			err := quest.NewProcessor(l, ctx).StartQuest(s.Field(), s.CharacterId(), questId, sp.NpcId(), false)
			if err != nil {
				l.WithError(err).Errorf("Failed to start quest [%d] conversation for character [%d] with NPC [%d].", questId, s.CharacterId(), sp.NpcId())
			}
			return
		case QuestActionScriptStart:
			sp := &quest3.ActionScriptStart{}
			sp.Decode(l, ctx)(r, readerOptions)
			l.Debugf("Character [%d] starting scripted quest [%d] conversation with NPC [%d]. x,y [%d,%d]", s.CharacterId(), questId, sp.NpcId(), sp.X(), sp.Y())
			err := quest.NewProcessor(l, ctx).StartQuestConversation(s.Field(), questId, sp.NpcId(), s.CharacterId())
			if err != nil {
				l.WithError(err).Errorf("Failed to start quest [%d] conversation for character [%d] with NPC [%d].", questId, s.CharacterId(), sp.NpcId())
			}
			return
		case QuestActionComplete:
			sp := quest3.NewActionComplete(q.AutoStart())
			sp.Decode(l, ctx)(r, readerOptions)
			l.Debugf("Character [%d] completing quest [%d] conversation with NPC [%d]. x,y [%d,%d]", s.CharacterId(), questId, sp.NpcId(), sp.X(), sp.Y())
			err := quest.NewProcessor(l, ctx).CompleteQuest(s.Field(), s.CharacterId(), questId, sp.NpcId(), sp.Selection(), false)
			if err != nil {
				l.WithError(err).Errorf("Failed to start quest [%d] completion conversation for character [%d] with NPC [%d].", questId, s.CharacterId(), sp.NpcId())
			}
			return
		case QuestActionScriptEnd:
			sp := &quest3.ActionScriptEnd{}
			sp.Decode(l, ctx)(r, readerOptions)
			l.Debugf("Character [%d] completing scripted quest [%d] conversation with NPC [%d]. x,y [%d,%d]", s.CharacterId(), questId, sp.NpcId(), sp.X(), sp.Y())
			err := quest.NewProcessor(l, ctx).StartQuestConversation(s.Field(), questId, sp.NpcId(), s.CharacterId())
			if err != nil {
				l.WithError(err).Errorf("Failed to start quest [%d] completion conversation for character [%d] with NPC [%d].", questId, s.CharacterId(), sp.NpcId())
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
			sp := &quest3.ActionRestoreLostItem{}
			sp.Decode(l, ctx)(r, readerOptions)
			l.Debugf("Character [%d] restoring lost item [%d] for quest [%d]. unk1 [%d]. rem [%d]", s.CharacterId(), sp.ItemId(), questId, sp.Unk1(), r.Available())
			err := quest.NewProcessor(l, ctx).RestoreItem(s.Field(), s.CharacterId(), questId, sp.ItemId())
			if err != nil {
				l.WithError(err).Errorf("Failed to restore item [%d] for quest [%d] for character [%d].", sp.ItemId(), questId, s.CharacterId())
			}
			return
		}
		l.Warnf("Character [%d] sent unknown quest action [%d] for quest [%d].", s.CharacterId(), action, questId)
	}
}
