package conversation

import (
	"atlas-npc-conversations/conversation/script"
	registry2 "atlas-npc-conversations/conversation/script/registry"
	"context"
	"errors"
	"github.com/Chronicle20/atlas-constants/field"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
)

type Processor interface {
	Start(field field.Model, npcId uint32, characterId uint32) error
	Continue(npcId uint32, characterId uint32, action byte, lastMessageType byte, selection int32) error
	ContinueViaEvent(characterId uint32, action byte, referenceId int32) error
	End(characterId uint32) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	t   tenant.Model
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		t:   tenant.MustFromContext(ctx),
	}
}

func (p *ProcessorImpl) Start(field field.Model, npcId uint32, characterId uint32) error {
	p.l.Debugf("Starting conversation with NPC [%d] with character [%d] in map [%d].", npcId, characterId, field.MapId())
	pctx, err := GetRegistry().GetPreviousContext(p.t, characterId)
	if err == nil {
		p.l.Debugf("Previous conversation between character [%d] and npc [%d] exists, avoiding starting new conversation with [%d].", characterId, pctx.ctx.NPCId, npcId)
		return errors.New("another conversation exists")
	}

	s, err := registry2.GetRegistry().GetScript(p.t, npcId)
	if err != nil {
		p.l.Errorf("Script for npc [%d] is not implemented.", npcId)
		return errors.New("not implemented")
	}

	sctx := script.Context{
		Field:       field,
		CharacterId: characterId,
		NPCId:       npcId,
	}
	ns := (*s).Initial(p.l)(p.ctx)(sctx)

	if ns != nil {
		GetRegistry().SetContext(p.t, characterId, sctx, ns)
	} else {
		GetRegistry().ClearContext(p.t, characterId)
	}

	return nil
}

func (p *ProcessorImpl) Continue(npcId uint32, characterId uint32, action byte, lastMessageType byte, selection int32) error {
	s, err := GetRegistry().GetPreviousContext(p.t, characterId)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to retrieve conversation context for [%d].", characterId)
		return errors.New("conversation context not found")
	}
	sctx := s.ctx
	state := s.ns

	p.l.Debugf("Continuing conversation with NPC [%d] with character [%d] in map [%d].", sctx.NPCId, characterId, sctx.Field.MapId())
	p.l.Debugf("Calling continue for NPC [%d] conversation with: mode [%d], type [%d], selection [%d].", sctx.NPCId, action, lastMessageType, selection)
	ns := state(p.l)(p.ctx)(sctx, action, lastMessageType, selection)
	if ns != nil {
		GetRegistry().SetContext(p.t, characterId, sctx, ns)
	} else {
		GetRegistry().ClearContext(p.t, characterId)
	}
	return nil
}

func (p *ProcessorImpl) ContinueViaEvent(characterId uint32, action byte, referenceId int32) error {
	s, err := GetRegistry().GetPreviousContext(p.t, characterId)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to retrieve conversation context for [%d].", characterId)
		return errors.New("conversation context not found")
	}
	sctx := s.ctx
	state := s.ns

	p.l.Debugf("Continuing conversation with NPC [%d] with character [%d] in map [%d].", sctx.NPCId, characterId, sctx.Field.MapId())
	p.l.Debugf("Calling continue for NPC [%d] conversation with: mode [%d], type [%d], selection [%d].", sctx.NPCId, action, 0, referenceId)
	ns := state(p.l)(p.ctx)(sctx, action, 0, referenceId)
	if ns != nil {
		GetRegistry().SetContext(p.t, characterId, sctx, ns)
	} else {
		GetRegistry().ClearContext(p.t, characterId)
	}
	return nil
}

func (p *ProcessorImpl) End(characterId uint32) error {
	p.l.Debugf("Ending conversation with character [%d].", characterId)
	GetRegistry().ClearContext(p.t, characterId)
	return nil
}
