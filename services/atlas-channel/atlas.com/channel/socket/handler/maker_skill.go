package handler

import (
	"atlas-channel/maker"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/sirupsen/logrus"

	charpkt "github.com/Chronicle20/atlas/libs/atlas-packet/character"
	charcb "github.com/Chronicle20/atlas/libs/atlas-packet/character/clientbound"
	charsb "github.com/Chronicle20/atlas/libs/atlas-packet/character/serverbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

// makerResultFailedValue is the MAKER_RESULT nResult sent for every rejection
// and every atlas-maker-unreachable outcome. The value 2 is the wire-verified
// FAILED sentinel: libs/atlas-packet/character/clientbound/maker_result_test.go
// (NewMakerResultFailed(2)) asserts the encoded byte fixture `02 00 00 00`
// with length exactly 4 (nResult only, no mode) against IDA evidence that the
// client's nResult guard treats any value outside {0, 1} as the bodyless
// FAILED arm and reads nothing further (docs/tasks/task-285-maker-skill-crafting/plan.md:1455-1461).
// The FAILED arm carries no room to distinguish PRD §5 rejection codes on the
// wire, so a single sentinel is sufficient; the rejection code itself is only
// logged.
const makerResultFailedValue = uint32(2)

// craftRequestFromMakerSkill maps the decoded, UNVALIDATED MakerSkill request
// onto atlas-maker's craft POST body verbatim -- the channel forwards every
// field exactly as decoded (including a gem list the character does not
// hold); atlas-maker alone validates (design §3.3, this task's contract).
func craftRequestFromMakerSkill(p charsb.MakerSkill, worldId byte, channelId byte) maker.CraftRequest {
	return maker.CraftRequest{
		Mode:           p.Mode(),
		WorldId:        worldId,
		ChannelId:      channelId,
		TargetItemId:   p.TargetItemId(),
		UseCatalyst:    p.UseCatalyst(),
		GemItemIds:     p.GemItemIds(),
		LeftoverItemId: p.LeftoverItemId(),
		EquipItemId:    p.ItemId(),
		SlotPos:        int16(p.SlotPos()),
	}
}

// MakerSkillHandleFunc handles CUIItemMaker::RequestItemMake (MAKER_SKILL,
// libs/atlas-packet/character/serverbound.MakerSkillHandle). It decodes the
// mode and arm body, forwards the request verbatim to atlas-maker's POST
// /characters/{id}/maker/crafts, and on rejection or a transport failure
// writes a MAKER_RESULT failure so the client's maker UI is never left
// locked (FR-5.2). On acceptance it writes nothing: the result is written
// when the craft saga reaches a terminal state (design §3.3; Task 26).
func MakerSkillHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := &charsb.MakerSkill{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		req := craftRequestFromMakerSkill(*p, byte(s.WorldId()), byte(s.ChannelId()))
		if _, err := maker.NewProcessor(l, ctx).Create(s.CharacterId(), req); err != nil {
			l.WithError(err).Warnf("Character [%d] maker craft (mode [%d]) rejected or atlas-maker unreachable; code [%s].", s.CharacterId(), p.Mode(), maker.CodeOf(err))
			writeMakerResultFailed(l, ctx, wp, s)
			return
		}
	}
}

// writeMakerResultFailed announces the fixed MAKER_RESULT FAILED arm to s.
func writeMakerResultFailed(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, s session.Model) {
	if err := session.Announce(l)(ctx)(wp)(charcb.MakerResultWriter)(charpkt.MakerResultFailedBody(makerResultFailedValue))(s); err != nil {
		l.WithError(err).Errorf("Unable to announce MAKER_RESULT failure to character [%d].", s.CharacterId())
	}
}
