package handler

import (
	"atlas-channel/character"
	"atlas-channel/maps/location"
	"atlas-channel/message"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"errors"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	characterconst "github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	chat "github.com/Chronicle20/atlas/libs/atlas-packet/chat/serverbound"
	fieldcb "github.com/Chronicle20/atlas/libs/atlas-packet/field/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

func CharacterChatWhisperHandleFunc(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
	return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
		p := chat.Whisper{}
		p.Decode(l, ctx)(r, readerOptions)
		l.Debugf("[%s] read [%s]", p.Operation(), p.String())

		if p.Mode() == chat.WhisperModeFind || p.Mode() == chat.WhisperModeBuddyWindowFind {
			_ = produceFindResultBody(l)(ctx)(wp)(p.Mode(), p.TargetName())(s)
			return
		}
		if p.Mode() == chat.WhisperModeChat {
			err := message.NewProcessor(l, ctx).WhisperChat(s.Field(), s.CharacterId(), p.Msg(), p.TargetName())
			if err != nil {
				_ = session.Announce(l)(ctx)(wp)(fieldcb.WhisperWriter)(fieldcb.NewWhisperSendResult(0x0A, p.TargetName(), false).Encode)(s)
				return
			}
			return
		}
		l.Warnf("Character [%d] using unhandled whisper mode [%d]. Target [%s], Message [%s], UpdateTime [%d]", s.CharacterId(), p.Mode(), p.TargetName(), p.Msg(), p.UpdateTime())
	}
}

// The /find path's three lookups, exposed as package-level seams so the
// decision table below is unit-testable. This mirrors
// checkNameChangeValidityFunc in cash_shop_check_name_change.go; tests swap
// them and restore via t.Cleanup.
var findCharacterByNameFunc = func(l logrus.FieldLogger, ctx context.Context, name string) (character.Model, error) {
	return character.NewProcessor(l, ctx).GetByName(name)
}

var findLocalSessionFunc = func(l logrus.FieldLogger, ctx context.Context, ch channel.Model, characterId uint32) (session.Model, error) {
	return session.NewProcessor(l, ctx).GetByCharacterId(ch)(characterId)
}

// findCharacterLocationFunc returns location.Model rather than field.Model
// because field.Model has nowhere to carry the presence state. Note this is
// location.Get, NOT location.ResolveMapId: ResolveMapId collapses every
// failure to map id 0, which is exactly how a transport failure renders as a
// real location today.
var findCharacterLocationFunc = func(l logrus.FieldLogger, ctx context.Context, characterId uint32) (location.Model, error) {
	return location.Get(l, ctx, characterId)
}

// findOutcomeKind discriminates the four wire shapes /find can answer with.
type findOutcomeKind int

const (
	findOutcomeError findOutcomeKind = iota
	findOutcomeCashShop
	findOutcomeMap
	findOutcomeChannel
)

// findOutcome is the result of the decision table. branch names the rule that
// matched, so the four wire-identical error branches stay separable in logs
// (FR-13).
type findOutcome struct {
	kind      findOutcomeKind
	branch    string
	name      string
	mapId     uint32
	x         int32
	y         int32
	channelId uint32
	err       error // set only for branch "lookup-failed"
}

// findDecision is the pure core of /find: PRD §4.1's rule ordering with the
// FR-4/FR-6/FR-7 sources corrected to read atlas-maps's presence state.
// Evaluated in order; first match wins. No writer, no session mutation, no
// packet types — every FR-1..FR-7 test lands here.
func findDecision(l logrus.FieldLogger, ctx context.Context, s session.Model, targetName string) findOutcome {
	tc, err := findCharacterByNameFunc(l, ctx, targetName)
	if err != nil {
		// FR-1. Echo the REQUESTED name: there is no resolved one.
		return findOutcome{kind: findOutcomeError, branch: "unresolved", name: targetName}
	}

	// FR-2. Cross-world targets are not findable. Wire-identical to FR-1 by
	// design (PRD §8) but a distinct branch in the logs.
	if tc.WorldId() != s.Field().WorldId() {
		return findOutcome{kind: findOutcomeError, branch: "cross-world", name: targetName}
	}

	// FR-3. GM visibility is a boolean predicate on level > 0, not a tier
	// comparison: any GM sees any GM, no GM is hidden from another GM.
	if tc.Gm() && !s.Gm() {
		return findOutcome{kind: findOutcomeError, branch: "gm-concealed", name: targetName}
	}

	// The local session is consulted before the location row. A live local
	// session is by construction authoritative about both liveness and
	// cash-scene, and costs nothing.
	if ls, lerr := findLocalSessionFunc(l, ctx, s.Field().Channel(), tc.Id()); lerr == nil {
		if ls.CashScene() != session.CashSceneNone {
			// FR-4a. CashSceneMts folds in here: the ITC renders inside the
			// cash-shop CStage and the client has no separate shape for it.
			return findOutcome{kind: findOutcomeCashShop, branch: "cash-shop-local", name: tc.Name()}
		}
		// FR-5. Same channel, on a map. The map id comes from the location
		// row; if that lookup fails we answer the error shape rather than
		// map 0 — "not findable" for a demonstrably online player is wrong but
		// recoverable, whereas map 0 is confidently wrong.
		loc, lerr2 := findCharacterLocationFunc(l, ctx, tc.Id())
		if lerr2 != nil {
			return locationErrorOutcome(targetName, lerr2)
		}
		return findOutcome{
			kind:   findOutcomeMap,
			branch: "map-local",
			name:   tc.Name(),
			mapId:  uint32(loc.MapId()),
			x:      int32(tc.X()),
			y:      int32(tc.Y()),
		}
	}

	loc, err := findCharacterLocationFunc(l, ctx, tc.Id())
	if err != nil {
		return locationErrorOutcome(targetName, err)
	}

	switch loc.State() {
	case characterconst.PresenceStateInCashShop:
		// FR-4b.
		return findOutcome{kind: findOutcomeCashShop, branch: "cash-shop-remote", name: tc.Name()}
	case characterconst.PresenceStateInField:
		// FR-6. channel.Id is already the 0-based internal value and the codec
		// writes it unadjusted; the client adds one for display.
		return findOutcome{
			kind:      findOutcomeChannel,
			branch:    "channel-remote",
			name:      tc.Name(),
			channelId: uint32(loc.ChannelId()),
		}
	default:
		// FR-7a. OFFLINE, and anything unrecognised, which ParsePresenceState
		// has already narrowed to OFFLINE.
		return findOutcome{kind: findOutcomeError, branch: "offline", name: targetName}
	}
}

// locationErrorOutcome separates "no row at all" (a character who has never
// logged in) from an infrastructure failure. Both answer the same wire shape;
// only the log line and level differ.
func locationErrorOutcome(targetName string, err error) findOutcome {
	if errors.Is(err, location.ErrNotFound) {
		return findOutcome{kind: findOutcomeError, branch: "never-logged-in", name: targetName}
	}
	return findOutcome{kind: findOutcomeError, branch: "lookup-failed", name: targetName, err: err}
}

func produceFindResultBody(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(mode chat.WhisperMode, targetName string) model.Operator[session.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(mode chat.WhisperMode, targetName string) model.Operator[session.Model] {
		return func(wp writer.Producer) func(mode chat.WhisperMode, targetName string) model.Operator[session.Model] {
			return func(mode chat.WhisperMode, targetName string) model.Operator[session.Model] {
				return func(s session.Model) error {
					var resultMode byte
					if mode == chat.WhisperModeBuddyWindowFind {
						resultMode = 0x48
					} else {
						resultMode = 0x09
					}

					o := findDecision(l, ctx, s, targetName)

					// FR-13. One line per /find, carrying the branch, so the
					// four wire-identical error branches stay separable.
					entry := l.WithFields(logrus.Fields{
						"requester_id": s.CharacterId(),
						"target_name":  targetName,
						"arm":          resultMode,
						"branch":       o.branch,
					})
					if o.err != nil {
						entry.WithError(o.err).Error("/find location lookup failed")
					} else {
						entry.Debug("/find resolved")
					}

					af := session.Announce(l)(ctx)(wp)(fieldcb.WhisperWriter)
					switch o.kind {
					case findOutcomeCashShop:
						return af(fieldcb.NewWhisperFindResultCashShop(resultMode, o.name).Encode)(s)
					case findOutcomeChannel:
						return af(fieldcb.NewWhisperFindResultChannel(resultMode, o.name, o.channelId).Encode)(s)
					case findOutcomeMap:
						// The 0x09 arm carries x/y; 0x48 does not. Confirmed
						// against the client on every version — the read is
						// gated on the mode being odd AND findMode == 1.
						if resultMode == 0x09 {
							return af(fieldcb.NewWhisperFindResultMapWithXY(resultMode, o.name, o.mapId, o.x, o.y).Encode)(s)
						}
						return af(fieldcb.NewWhisperFindResultMap(resultMode, o.name, o.mapId).Encode)(s)
					default:
						return af(fieldcb.NewWhisperFindResultError(resultMode, o.name).Encode)(s)
					}
				}
			}
		}
	}
}
