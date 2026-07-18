package handler

import (
	"atlas-channel/character"
	"atlas-channel/data/skill/effect"
	_map "atlas-channel/map"
	"atlas-channel/party"
	"atlas-channel/session"
	"context"
	"sync"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
)

// PartyRecipient is the canonical party-member descriptor produced by
// the party selectors. Embeds enough state for downstream handlers
// (heal: HP / MaxHp; buff: just Id is enough) without forcing each call
// site to refetch.
type PartyRecipient struct {
	id    uint32
	x     int16
	y     int16
	hp    uint16
	maxHp uint16
	mp    uint16
	maxMp uint16
}

func (r PartyRecipient) Id() uint32    { return r.id }
func (r PartyRecipient) X() int16      { return r.x }
func (r PartyRecipient) Y() int16      { return r.y }
func (r PartyRecipient) Hp() uint16    { return r.hp }
func (r PartyRecipient) MaxHp() uint16 { return r.maxHp }
func (r PartyRecipient) Mp() uint16    { return r.mp }
func (r PartyRecipient) MaxMp() uint16 { return r.maxMp }

// PartyRecipientBuilder is the canonical constructor for PartyRecipient.
type PartyRecipientBuilder struct {
	r PartyRecipient
}

func NewPartyRecipientBuilder() *PartyRecipientBuilder { return &PartyRecipientBuilder{} }

func (b *PartyRecipientBuilder) SetId(v uint32) *PartyRecipientBuilder    { b.r.id = v; return b }
func (b *PartyRecipientBuilder) SetX(v int16) *PartyRecipientBuilder      { b.r.x = v; return b }
func (b *PartyRecipientBuilder) SetY(v int16) *PartyRecipientBuilder      { b.r.y = v; return b }
func (b *PartyRecipientBuilder) SetHp(v uint16) *PartyRecipientBuilder    { b.r.hp = v; return b }
func (b *PartyRecipientBuilder) SetMaxHp(v uint16) *PartyRecipientBuilder { b.r.maxHp = v; return b }
func (b *PartyRecipientBuilder) SetMp(v uint16) *PartyRecipientBuilder    { b.r.mp = v; return b }
func (b *PartyRecipientBuilder) SetMaxMp(v uint16) *PartyRecipientBuilder { b.r.maxMp = v; return b }
func (b *PartyRecipientBuilder) Build() PartyRecipient                    { return b.r }

// loadCasterPartyFunc is the party-load seam tests can replace.
var loadCasterPartyFunc = func(l logrus.FieldLogger, ctx context.Context, casterId uint32) (party.Model, error) {
	return party.NewProcessor(l, ctx).GetByMemberId(casterId)
}

// inMapCharacterIdsFunc is the live-session seam tests can replace. Returns
// the set of character ids with a live session in the caster's field, used
// to exclude members who are in the party + same map per the party service
// but not actually present on this channel.
var inMapCharacterIdsFunc = func(l logrus.FieldLogger, ctx context.Context, f field.Model) map[uint32]struct{} {
	inMap := map[uint32]struct{}{}
	// ForSessionsInMap runs the callback concurrently across sessions, so the
	// map write must be synchronized (a plain map write here fatals the process
	// with "concurrent map writes" once two sessions are present).
	var mu sync.Mutex
	_ = _map.NewProcessor(l, ctx).ForSessionsInMap(f, func(s session.Model) error {
		mu.Lock()
		inMap[s.CharacterId()] = struct{}{}
		mu.Unlock()
		return nil
	})
	return inMap
}

// loadPartyMemberFunc is the per-member character-load seam tests can replace.
var loadPartyMemberFunc = func(l logrus.FieldLogger, ctx context.Context, memberId uint32) (character.Model, error) {
	return character.NewProcessor(l, ctx).GetById()(memberId)
}

// loadMapPlayerFunc is the per-player character-load seam (GM-variant map-wide
// selection) tests can replace.
var loadMapPlayerFunc = func(l logrus.FieldLogger, ctx context.Context, characterId uint32) (character.Model, error) {
	return character.NewProcessor(l, ctx).GetById()(characterId)
}

// SelectInRangePartyMembers returns party members other than the
// caster that satisfy all of:
//   - the bitmap bit for their party slot is set
//   - they are on the same channel + map as the caster
//   - they have a live session in the caster's field
//   - their Hp() > 0
//   - their (x,y) lies inside the LT/RB rectangle around the caster
//
// LT/RB are taken from e. If both LT and RB are zero-valued
// (point.Model with X=0, Y=0), the function returns an empty slice —
// the caster-only fallback. This is the selector for AoE party skills
// like Heal that carry a rectangle in WZ; the missing-rectangle case is
// an anomaly the caller (Heal) logs before falling back to caster-only.
//
// Errors loading the party return an empty slice (the cast continues
// caster-only). Errors fetching individual members are logged and the
// offending member is skipped.
func SelectInRangePartyMembers(
	l logrus.FieldLogger, ctx context.Context,
	f field.Model, casterId uint32, casterX, casterY int16,
	e effect.Model, memberBitmap byte,
) []PartyRecipient {
	lt, rb := e.LT(), e.RB()
	if lt.X() == 0 && lt.Y() == 0 && rb.X() == 0 && rb.Y() == 0 {
		// Missing rectangle — caster-only fallback.
		return nil
	}
	return selectPartyMembers(l, ctx, f, casterId, casterX, casterY, e, memberBitmap, true, false)
}

// SelectDeadInRangePartyMembers is the dead-only counterpart of
// SelectInRangePartyMembers: same bitmap / same-channel-map / live-session /
// LT-RB-rectangle filters, but keeps only members with Hp()==0. Used by Bishop
// Resurrection. Missing rectangle returns nil (no one to revive in range).
func SelectDeadInRangePartyMembers(
	l logrus.FieldLogger, ctx context.Context,
	f field.Model, casterId uint32, casterX, casterY int16,
	e effect.Model, memberBitmap byte,
) []PartyRecipient {
	lt, rb := e.LT(), e.RB()
	if lt.X() == 0 && lt.Y() == 0 && rb.X() == 0 && rb.Y() == 0 {
		return nil
	}
	return selectPartyMembers(l, ctx, f, casterId, casterX, casterY, e, memberBitmap, true, true)
}

// SelectDeadInRangeMapPlayers returns every dead player (Hp()==0) other than the
// caster who has a live session in the caster's field and whose position lies in
// the caster-relative LT/RB rectangle — party-agnostic. Used by GM / SuperGM
// Resurrection. Missing rectangle returns nil.
func SelectDeadInRangeMapPlayers(
	l logrus.FieldLogger, ctx context.Context,
	f field.Model, casterId uint32, casterX, casterY int16,
	e effect.Model,
) []PartyRecipient {
	lt, rb := e.LT(), e.RB()
	if lt.X() == 0 && lt.Y() == 0 && rb.X() == 0 && rb.Y() == 0 {
		return nil
	}
	inMap := inMapCharacterIdsFunc(l, ctx, f)
	out := make([]PartyRecipient, 0, len(inMap))
	for id := range inMap {
		if id == casterId {
			continue
		}
		mc, err := loadMapPlayerFunc(l, ctx, id)
		if err != nil {
			l.WithError(err).Debugf("Skipping map player [%d] from resurrection recipients: fetch failed.", id)
			continue
		}
		if mc.Hp() != 0 {
			continue
		}
		dx := mc.X() - casterX
		dy := mc.Y() - casterY
		if dx < int16(lt.X()) || dx > int16(rb.X()) || dy < int16(lt.Y()) || dy > int16(rb.Y()) {
			continue
		}
		out = append(out, NewPartyRecipientBuilder().
			SetId(mc.Id()).
			SetX(mc.X()).
			SetY(mc.Y()).
			SetHp(mc.Hp()).
			SetMaxHp(mc.MaxHp()).
			Build())
	}
	return out
}

// SelectPartyMembersInMap returns bitmap-selected, same-map, living party
// members other than the caster, WITHOUT any LT/RB rectangle restriction.
//
// Pure party buffs (Sharp Eyes, Hyper Body, Maple Warrior, Bless, Haste,
// Meditation, Rage, ...) carry no LT/RB rectangle in their WZ effect and
// apply to the whole map; the client-sent affected-member bitmap is the
// authority for which members are affected. Using SelectInRangePartyMembers
// for these would always hit the missing-rectangle caster-only fallback and
// never buff anyone but the caster.
func SelectPartyMembersInMap(
	l logrus.FieldLogger, ctx context.Context,
	f field.Model, casterId uint32, memberBitmap byte,
) []PartyRecipient {
	return selectPartyMembers(l, ctx, f, casterId, 0, 0, effect.Model{}, memberBitmap, false, false)
}

// SelectAllCharactersInMap returns a recipient for EVERY character with a live
// session in the field f, irrespective of party membership, HP, or position.
// Unlike the party selectors it applies no bitmap, no LT/RB rectangle, and no
// HP>0 filter — it is the map-wide selector for GM Heal + Dispel, which
// benefits every player in the map INCLUDING the caster.
//
// The in-map id set comes from the same live-session source the spawn paths
// use (ForSessionsInMap via inMapCharacterIdsFunc). Each id is loaded for its
// HP/MP snapshot; a member whose load fails is logged and skipped.
func SelectAllCharactersInMap(l logrus.FieldLogger, ctx context.Context, f field.Model) []PartyRecipient {
	inMap := inMapCharacterIdsFunc(l, ctx, f)
	out := make([]PartyRecipient, 0, len(inMap))
	for id := range inMap {
		mc, err := loadPartyMemberFunc(l, ctx, id)
		if err != nil {
			l.WithError(err).Debugf("SelectAllCharactersInMap: skipping character [%d]: fetch failed.", id)
			continue
		}
		out = append(out, NewPartyRecipientBuilder().
			SetId(mc.Id()).
			SetX(mc.X()).
			SetY(mc.Y()).
			SetHp(mc.Hp()).
			SetMaxHp(mc.MaxHp()).
			SetMp(mc.Mp()).
			SetMaxMp(mc.MaxMp()).
			Build())
	}
	return out
}

// selectPartyMembers is the shared enumeration backing both party-member
// selectors. When requireRect is true the caster-relative LT/RB rectangle
// from e is applied as a final filter; when false the rectangle is ignored
// entirely (map-wide application). When wantDead is true only members with
// Hp()==0 are kept; when false (the normal case) only living members are kept.
func selectPartyMembers(
	l logrus.FieldLogger, ctx context.Context,
	f field.Model, casterId uint32, casterX, casterY int16,
	e effect.Model, memberBitmap byte, requireRect bool, wantDead bool,
) []PartyRecipient {
	if memberBitmap == 0 || memberBitmap >= 128 {
		return nil
	}

	p, err := loadCasterPartyFunc(l, ctx, casterId)
	if err != nil {
		return nil
	}

	// Set of in-map session ids so we exclude offline / not-present members.
	inMap := inMapCharacterIdsFunc(l, ctx, f)

	out := make([]PartyRecipient, 0, len(p.Members()))
	for i, m := range p.Members() {
		if m.Id() == casterId {
			continue
		}
		if i >= 6 {
			break
		}
		// The v83 client packs the affected-member bitmap MSB-first by party
		// slot: CUserLocal::FindParty (IDA 0x96db3f) shifts the accumulator left
		// once per slot 0..5 then ORs bit 0, so slot i lands at bit (5-i). The
		// party member list here is in that same slot order, so member index i
		// maps to bit (5-i) — NOT bit i.
		if (memberBitmap>>uint(5-i))&1 == 0 {
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
		mc, mErr := loadPartyMemberFunc(l, ctx, m.Id())
		if mErr != nil {
			l.WithError(mErr).Debugf("Skipping party member [%d] from skill recipients: fetch failed.", m.Id())
			continue
		}
		if wantDead {
			if mc.Hp() != 0 {
				continue
			}
		} else {
			if mc.Hp() == 0 {
				continue
			}
		}
		if requireRect {
			lt, rb := e.LT(), e.RB()
			dx := mc.X() - casterX
			dy := mc.Y() - casterY
			if dx < int16(lt.X()) || dx > int16(rb.X()) || dy < int16(lt.Y()) || dy > int16(rb.Y()) {
				continue
			}
		}
		out = append(out, NewPartyRecipientBuilder().
			SetId(mc.Id()).
			SetX(mc.X()).
			SetY(mc.Y()).
			SetHp(mc.Hp()).
			SetMaxHp(mc.MaxHp()).
			Build())
	}
	return out
}
