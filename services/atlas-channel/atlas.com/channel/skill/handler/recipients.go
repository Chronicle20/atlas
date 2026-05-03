package handler

import (
	"atlas-channel/character"
	"atlas-channel/data/skill/effect"
	_map "atlas-channel/map"
	"atlas-channel/party"
	"atlas-channel/session"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/sirupsen/logrus"
)

// PartyRecipient is the canonical party-member descriptor produced by
// SelectInRangePartyMembers. Embeds enough state for downstream
// handlers (heal: HP / MaxHp; buff: just Id is enough) without forcing
// each call site to refetch.
type PartyRecipient struct {
	Id    uint32
	X     int16
	Y     int16
	Hp    uint16
	MaxHp uint16
}

// SelectInRangePartyMembers returns party members other than the
// caster that satisfy all of:
//   - the bitmap bit for their party slot is set
//   - they are on the same channel + map as the caster
//   - they have a live session in the caster's field
//   - their (x,y) lies inside the LT/RB rectangle around the caster
//   - their Hp() > 0
//
// LT/RB are taken from e. If both LT and RB are zero-valued
// (Point{0,0}), the function returns an empty slice — the caster-only
// fallback. Callers wanting "caster + in-range party" prepend the
// caster themselves.
//
// Errors loading the party return an empty slice (the cast continues
// caster-only). Errors enumerating sessions or fetching individual
// members are logged and the offending member is skipped.
func SelectInRangePartyMembers(
	l logrus.FieldLogger, ctx context.Context,
	f field.Model, casterId uint32, casterX, casterY int16,
	e effect.Model, memberBitmap byte,
) []PartyRecipient {
	if memberBitmap == 0 || memberBitmap >= 128 {
		return nil
	}
	lt, rb := e.LT(), e.RB()
	if lt.X == 0 && lt.Y == 0 && rb.X == 0 && rb.Y == 0 {
		// Missing rectangle — caster-only fallback.
		return nil
	}

	p, err := party.NewProcessor(l, ctx).GetByMemberId(casterId)
	if err != nil {
		return nil
	}

	// Build set of in-map session ids so we exclude offline members.
	inMap := map[uint32]struct{}{}
	_ = _map.NewProcessor(l, ctx).ForSessionsInMap(f, func(s session.Model) error {
		inMap[s.CharacterId()] = struct{}{}
		return nil
	})

	cp := character.NewProcessor(l, ctx)

	out := make([]PartyRecipient, 0, len(p.Members()))
	for i, m := range p.Members() {
		if m.Id() == casterId {
			continue
		}
		if i >= 6 {
			break
		}
		if (memberBitmap>>uint(i))&1 == 0 {
			continue
		}
		if !m.Online() {
			continue
		}
		if m.ChannelId() != f.ChannelId() || m.MapId() != f.MapId() {
			continue
		}
		if _, present := inMap[m.Id()]; !present {
			continue
		}
		mc, mErr := cp.GetById()(m.Id())
		if mErr != nil {
			l.WithError(mErr).Debugf("Skipping party member [%d] from skill recipients: fetch failed.", m.Id())
			continue
		}
		if mc.Hp() == 0 {
			continue
		}
		dx := mc.X() - casterX
		dy := mc.Y() - casterY
		if dx < lt.X || dx > rb.X || dy < lt.Y || dy > rb.Y {
			continue
		}
		out = append(out, PartyRecipient{
			Id:    mc.Id(),
			X:     mc.X(),
			Y:     mc.Y(),
			Hp:    mc.Hp(),
			MaxHp: mc.MaxHp(),
		})
	}
	return out
}
