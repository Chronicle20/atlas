package minigame

import (
	"atlas-channel/character"
	consumer2 "atlas-channel/kafka/consumer"
	minigame2 "atlas-channel/kafka/message/minigame"
	"atlas-channel/listener"
	_map "atlas-channel/map"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/model"
	"atlas-channel/socket/writer"
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	"github.com/Chronicle20/atlas/libs/atlas-constants/miniroom"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	atlasmodel "github.com/Chronicle20/atlas/libs/atlas-model/model"
	interactionpkt "github.com/Chronicle20/atlas/libs/atlas-packet/interaction"
	interactioncb "github.com/Chronicle20/atlas/libs/atlas-packet/interaction/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

const (
	// gameRoomCapacity is m_nMaxUsers for both game dialogs (design §5;
	// ida-notes §G5 room-enter blob).
	gameRoomCapacity = byte(2)
	// retreatStoneCount is the number of stones the client pops from the move
	// history on an accepted retreat. Per ida-notes §G2 the wire supports any
	// N, and the server (atlas-mini-games Task 15) always pops exactly one
	// stone and returns the turn to the requester.
	retreatStoneCount = byte(1)
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...atlasmodel.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...atlasmodel.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("mini_game_status_event")(minigame2.EnvEventTopicStatus)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
				var handles []listener.HandlerHandle
				t, _ := topic.EnvProvider(l)(minigame2.EnvEventTopicStatus)()
				for _, h := range []handler.Handler{
					message.AdaptHandler(message.PersistentConfig(handleCreatedEvent(sc, wp))),
					message.AdaptHandler(message.PersistentConfig(handleErrorEvent(sc, wp))),
					message.AdaptHandler(message.PersistentConfig(handleEnteredEvent(sc, wp))),
					message.AdaptHandler(message.PersistentConfig(handleLeftEvent(sc, wp))),
					message.AdaptHandler(message.PersistentConfig(handleRoomClosedEvent(sc, wp))),
					message.AdaptHandler(message.PersistentConfig(handleChatEvent(sc, wp))),
					message.AdaptHandler(message.PersistentConfig(handleBodylessGameEvent(sc, wp))),
					message.AdaptHandler(message.PersistentConfig(handleStartedEvent(sc, wp))),
					message.AdaptHandler(message.PersistentConfig(handleStonePlacedEvent(sc, wp))),
					message.AdaptHandler(message.PersistentConfig(handleCardFlippedEvent(sc, wp))),
					message.AdaptHandler(message.PersistentConfig(handleAnswerEvent(sc, wp))),
					message.AdaptHandler(message.PersistentConfig(handleSkippedEvent(sc, wp))),
					message.AdaptHandler(message.PersistentConfig(handleGameEndedEvent(sc, wp))),
					message.AdaptHandler(message.PersistentConfig(handleBalloonEvent(sc, wp))),
				} {
					id, err := rf(t, h)
					if err != nil {
						return nil, err
					}
					handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				}
				return handles, nil
			}
		}
	}
}

// guard applies the tenant/world/channel ownership check every handler runs
// before acting on an event.
func guard[E any](sc server.Model, ctx context.Context, e minigame2.StatusEvent[E]) bool {
	return sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId)
}

// fieldOf rebuilds the event's field for map-wide balloon broadcasts.
func fieldOf[E any](e minigame2.StatusEvent[E]) field.Model {
	return field.NewBuilder(e.WorldId, e.ChannelId, e.MapId).SetInstance(e.Instance).Build()
}

// opponentOf resolves the other occupant of the room relative to the acting
// character (owner slot 0 / visitor slot 1). Returns 0 when there is none.
func opponentOf[E any](e minigame2.StatusEvent[E]) uint32 {
	if e.CharacterId == e.OwnerId {
		return e.VisitorId
	}
	return e.OwnerId
}

// slotOf resolves a character's slot in the room (0 owner / 1 visitor).
func slotOf[E any](e minigame2.StatusEvent[E], characterId uint32) byte {
	if characterId == e.OwnerId {
		return 0
	}
	return 1
}

// announceTo sends one CharacterInteraction body to a single character's
// session, when present on this channel.
func announceTo(l logrus.FieldLogger, ctx context.Context, sc server.Model, wp writer.Producer, characterId uint32, body packet.Encode) {
	if characterId == 0 {
		return
	}
	_ = session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(characterId, session.Announce(l)(ctx)(wp)(interactioncb.CharacterInteractionWriter)(body))
}

// announceToRoom sends one CharacterInteraction body to the room owner and,
// when present, the visitor. The envelope's OwnerId/VisitorId identify the
// recipients (LEFT events carry the departed character in VisitorId).
func announceToRoom[E any](l logrus.FieldLogger, ctx context.Context, sc server.Model, wp writer.Producer, e minigame2.StatusEvent[E], body packet.Encode) {
	announceTo(l, ctx, sc, wp, e.OwnerId, body)
	if e.VisitorId != e.OwnerId {
		announceTo(l, ctx, sc, wp, e.VisitorId, body)
	}
}

// announceBalloon broadcasts a MiniRoom (UPDATE_CHAR_BOX) balloon body to every
// session in the room's field.
func announceBalloon[E any](l logrus.FieldLogger, ctx context.Context, wp writer.Producer, e minigame2.StatusEvent[E], body packet.Encode) {
	if err := _map.NewProcessor(l, ctx).ForSessionsInMap(fieldOf(e), session.Announce(l)(ctx)(wp)(interactionpkt.MiniRoomWriter)(body)); err != nil {
		l.WithError(err).Errorf("Unable to broadcast mini-game balloon for room [%d] in map [%d].", e.RoomId, e.MapId)
	}
}

// gameTypeCode maps a record GameType string to the int the client's 20-byte
// record leads with (Cosmic getMiniGame marker: 1 omok / 2 match cards —
// design §6.1). The markers are the shared mini-room type bytes
// (libs/atlas-constants/miniroom), the same values interaction.RoomType uses.
func gameTypeCode(gameType string) uint32 {
	switch gameType {
	case "OMOK":
		return uint32(miniroom.Omok)
	case "MATCH_CARDS":
		return uint32(miniroom.MatchCards)
	default:
		return 0
	}
}

// gameRecord converts a status-event RecordBody plus the session score into
// the packet GameRecord (Unknown = game-type marker, Points = session score).
func gameRecord(rb minigame2.RecordBody, score int32) interactionpkt.GameRecord {
	return interactionpkt.GameRecord{
		Unknown: gameTypeCode(rb.GameType),
		Wins:    rb.Wins,
		Ties:    rb.Ties,
		Losses:  rb.Losses,
		Points:  uint32(score),
	}
}

// gameRoomPlayer resolves a character into a MiniGameRoomPlayer (avatar, name,
// job code and record) for the room-enter / enter encodes.
func gameRoomPlayer(l logrus.FieldLogger, ctx context.Context, slot byte, characterId uint32, rec interactionpkt.GameRecord) (interactioncb.MiniGameRoomPlayer, error) {
	cp := character.NewProcessor(l, ctx)
	c, err := cp.GetById(cp.InventoryDecorator)(characterId)
	if err != nil {
		return interactioncb.MiniGameRoomPlayer{}, err
	}
	return interactioncb.MiniGameRoomPlayer{
		Slot:    slot,
		Avatar:  model.NewFromCharacter(c, false),
		Name:    c.Name(),
		JobCode: uint16(c.JobId()),
		Record:  rec,
	}, nil
}

// handleCreatedEvent sends the room-enter snapshot (yourSlot 0) to the owner.
// The balloon spawn arrives as a separate BALLOON_UPDATED event.
func handleCreatedEvent(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger, ctx context.Context, e minigame2.StatusEvent[minigame2.CreatedEventBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, e minigame2.StatusEvent[minigame2.CreatedEventBody]) {
		if e.Type != minigame2.EventTypeCreated {
			return
		}
		if !guard(sc, ctx, e) {
			return
		}
		l.Debugf("Mini-game room [%d] created by character [%d]. roomType [%d], title [%s].", e.RoomId, e.OwnerId, e.Body.RoomType, e.Body.Title)
		owner, err := gameRoomPlayer(l, ctx, 0, e.OwnerId, gameRecord(e.Body.OwnerRecord, 0))
		if err != nil {
			l.WithError(err).Errorf("Unable to resolve owner [%d] for mini-game room [%d].", e.OwnerId, e.RoomId)
			return
		}
		body := interactioncb.CharacterInteractionMiniGameRoomBody(interactionpkt.RoomType(e.Body.RoomType), gameRoomCapacity, 0, []interactioncb.MiniGameRoomPlayer{owner}, e.Body.Title, e.Body.PieceType, false, 0)
		announceTo(l, ctx, sc, wp, e.OwnerId, body)
	}
}

// handleErrorEvent routes CREATE_ERROR / ENTER_ERROR to the acting character.
// The body carries the enterError KEY string, resolved to the per-version
// numeric code by the tenant enterError table inside the body func.
func handleErrorEvent(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger, ctx context.Context, e minigame2.StatusEvent[minigame2.ErrorEventBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, e minigame2.StatusEvent[minigame2.ErrorEventBody]) {
		if e.Type != minigame2.EventTypeCreateError && e.Type != minigame2.EventTypeEnterError {
			return
		}
		if !guard(sc, ctx, e) {
			return
		}
		l.Debugf("Mini-game [%s] for character [%d]. code [%s].", e.Type, e.CharacterId, e.Body.Code)
		announceTo(l, ctx, sc, wp, e.CharacterId, interactioncb.CharacterInteractionEnterResultErrorBody(e.Body.Code))
	}
}

// handleEnteredEvent sends the full room snapshot (yourSlot = the visitor's
// slot) to the joining visitor, and the game ENTER (avatar + record) to the
// owner. The balloon occupancy update arrives as a separate BALLOON_UPDATED.
func handleEnteredEvent(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger, ctx context.Context, e minigame2.StatusEvent[minigame2.EnteredEventBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, e minigame2.StatusEvent[minigame2.EnteredEventBody]) {
		if e.Type != minigame2.EventTypeEntered {
			return
		}
		if !guard(sc, ctx, e) {
			return
		}
		l.Debugf("Character [%d] entered mini-game room [%d] at slot [%d].", e.VisitorId, e.RoomId, e.Body.Slot)
		owner, err := gameRoomPlayer(l, ctx, 0, e.OwnerId, gameRecord(e.Body.OwnerRecord, e.Body.OwnerScore))
		if err != nil {
			l.WithError(err).Errorf("Unable to resolve owner [%d] for mini-game room [%d].", e.OwnerId, e.RoomId)
			return
		}
		visitor, err := gameRoomPlayer(l, ctx, e.Body.Slot, e.VisitorId, gameRecord(e.Body.VisitorRecord, e.Body.VisitorScore))
		if err != nil {
			l.WithError(err).Errorf("Unable to resolve visitor [%d] for mini-game room [%d].", e.VisitorId, e.RoomId)
			return
		}
		roomBody := interactioncb.CharacterInteractionMiniGameRoomBody(interactionpkt.RoomType(e.Body.RoomType), gameRoomCapacity, e.Body.Slot, []interactioncb.MiniGameRoomPlayer{owner, visitor}, e.Body.Title, e.Body.PieceType, false, 0)
		announceTo(l, ctx, sc, wp, e.VisitorId, roomBody)
		announceTo(l, ctx, sc, wp, e.OwnerId, interactioncb.CharacterInteractionMiniGameEnterBody(visitor))
	}
}

// handleLeftEvent notifies both sides of a departure (4 left / 5 expelled).
// The envelope carries the departed character in VisitorId, so the room
// broadcast reaches the leaver (closing their dialog) and the owner (freeing
// the slot).
func handleLeftEvent(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger, ctx context.Context, e minigame2.StatusEvent[minigame2.LeftEventBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, e minigame2.StatusEvent[minigame2.LeftEventBody]) {
		if e.Type != minigame2.EventTypeLeft {
			return
		}
		if !guard(sc, ctx, e) {
			return
		}
		l.Debugf("Character [%d] left mini-game room [%d]. slot [%d], status [%s].", e.VisitorId, e.RoomId, e.Body.Slot, e.Body.Status)
		announceToRoom(l, ctx, sc, wp, e, interactioncb.CharacterInteractionLeaveReasonBody(e.Body.Slot, e.Body.Status))
	}
}

// handleRoomClosedEvent closes the visitor's dialog when the owner tears the
// room down (leave status 3). The owner's client closed itself on EXIT; the
// balloon remove arrives as a separate BALLOON_UPDATED with Remove set.
func handleRoomClosedEvent(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger, ctx context.Context, e minigame2.StatusEvent[minigame2.RoomClosedEventBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, e minigame2.StatusEvent[minigame2.RoomClosedEventBody]) {
		if e.Type != minigame2.EventTypeRoomClosed {
			return
		}
		if !guard(sc, ctx, e) {
			return
		}
		l.Debugf("Mini-game room [%d] closed by owner [%d]. visitorStatus [%s].", e.RoomId, e.OwnerId, e.Body.VisitorStatus)
		announceTo(l, ctx, sc, wp, e.VisitorId, interactioncb.CharacterInteractionLeaveReasonBody(1, e.Body.VisitorStatus))
	}
}

func handleChatEvent(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger, ctx context.Context, e minigame2.StatusEvent[minigame2.ChatEventBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, e minigame2.StatusEvent[minigame2.ChatEventBody]) {
		if e.Type != minigame2.EventTypeChat {
			return
		}
		if !guard(sc, ctx, e) {
			return
		}
		l.Debugf("Chat in mini-game room [%d] from slot [%d].", e.RoomId, e.Body.Slot)
		announceToRoom(l, ctx, sc, wp, e, interactioncb.CharacterInteractionChatBody(e.Body.Slot, e.Body.Message))
	}
}

// handleBodylessGameEvent routes the four bodyless game events: READY/UNREADY
// go to the whole room; TIE_REQUESTED/RETREAT_REQUESTED go to the OPPONENT of
// the requester only (design §5 — the requester's own client already shows
// the pending state).
func handleBodylessGameEvent(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger, ctx context.Context, e minigame2.StatusEvent[minigame2.EmptyEventBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, e minigame2.StatusEvent[minigame2.EmptyEventBody]) {
		switch e.Type {
		case minigame2.EventTypeReady, minigame2.EventTypeUnready, minigame2.EventTypeTieRequested, minigame2.EventTypeRetreatRequested:
		default:
			return
		}
		if !guard(sc, ctx, e) {
			return
		}
		l.Debugf("Mini-game [%s] in room [%d] by character [%d].", e.Type, e.RoomId, e.CharacterId)
		switch e.Type {
		case minigame2.EventTypeReady:
			announceToRoom(l, ctx, sc, wp, e, interactioncb.CharacterInteractionMiniGameReadyBody())
		case minigame2.EventTypeUnready:
			announceToRoom(l, ctx, sc, wp, e, interactioncb.CharacterInteractionMiniGameUnreadyBody())
		case minigame2.EventTypeTieRequested:
			announceTo(l, ctx, sc, wp, opponentOf(e), interactioncb.CharacterInteractionMiniGameRequestTieBody())
		case minigame2.EventTypeRetreatRequested:
			announceTo(l, ctx, sc, wp, opponentOf(e), interactioncb.CharacterInteractionMiniGameRetreatRequestBody())
		}
	}
}

func handleStartedEvent(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger, ctx context.Context, e minigame2.StatusEvent[minigame2.StartedEventBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, e minigame2.StatusEvent[minigame2.StartedEventBody]) {
		if e.Type != minigame2.EventTypeStarted {
			return
		}
		if !guard(sc, ctx, e) {
			return
		}
		l.Debugf("Mini-game started in room [%d]. roomType [%d], firstMover [%d].", e.RoomId, e.Body.RoomType, e.Body.FirstMover)
		var body packet.Encode
		if interactionpkt.RoomType(e.Body.RoomType) == interactionpkt.MatchCardRoomType {
			body = interactioncb.CharacterInteractionMiniGameStartMatchCardsBody(e.Body.FirstMover, e.Body.Deck)
		} else {
			body = interactioncb.CharacterInteractionMiniGameStartOmokBody(e.Body.FirstMover)
		}
		announceToRoom(l, ctx, sc, wp, e, body)
	}
}

func handleStonePlacedEvent(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger, ctx context.Context, e minigame2.StatusEvent[minigame2.StonePlacedEventBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, e minigame2.StatusEvent[minigame2.StonePlacedEventBody]) {
		if e.Type != minigame2.EventTypeStonePlaced {
			return
		}
		if !guard(sc, ctx, e) {
			return
		}
		l.Debugf("Stone placed in room [%d] at [%d,%d] type [%d].", e.RoomId, e.Body.X, e.Body.Y, e.Body.StoneType)
		announceToRoom(l, ctx, sc, wp, e, interactioncb.CharacterInteractionMiniGameMoveStoneBody(e.Body.X, e.Body.Y, e.Body.StoneType))
	}
}

// handleCardFlippedEvent: the FIRST flip of a turn goes to the opponent of the
// flipper only (their own client flipped locally); the SECOND flip (with match
// resolution) goes to both (design §3.2/§5).
func handleCardFlippedEvent(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger, ctx context.Context, e minigame2.StatusEvent[minigame2.CardFlippedEventBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, e minigame2.StatusEvent[minigame2.CardFlippedEventBody]) {
		if e.Type != minigame2.EventTypeCardFlipped {
			return
		}
		if !guard(sc, ctx, e) {
			return
		}
		l.Debugf("Card flipped in room [%d]. secondFlip [%t], slot [%d].", e.RoomId, e.Body.SecondFlip, e.Body.Slot)
		if !e.Body.SecondFlip {
			announceTo(l, ctx, sc, wp, opponentOf(e), interactioncb.CharacterInteractionMiniGameCardSelectFirstBody(e.Body.Slot))
			return
		}
		announceToRoom(l, ctx, sc, wp, e, interactioncb.CharacterInteractionMiniGameCardSelectSecondBody(e.Body.Slot, e.Body.FirstSlot, e.Body.ResultType))
	}
}

// handleAnswerEvent routes TIE_ANSWERED / RETREAT_ANSWERED. A tie accept ends
// the game — the service emits GAME_ENDED, so only the deny is forwarded (to
// the requester). A retreat accept pops the board on BOTH clients: per
// ida-notes §G2 the server pops exactly retreatStoneCount stones and the turn
// returns to the requester, whose slot is the opponent of the answerer
// (e.CharacterId). A retreat deny is forwarded to the requester only.
func handleAnswerEvent(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger, ctx context.Context, e minigame2.StatusEvent[minigame2.AnswerEventBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, e minigame2.StatusEvent[minigame2.AnswerEventBody]) {
		if e.Type != minigame2.EventTypeTieAnswered && e.Type != minigame2.EventTypeRetreatAnswered {
			return
		}
		if !guard(sc, ctx, e) {
			return
		}
		l.Debugf("Mini-game [%s] in room [%d] by character [%d]. accept [%t].", e.Type, e.RoomId, e.CharacterId, e.Body.Accept)
		if e.Type == minigame2.EventTypeTieAnswered {
			if e.Body.Accept {
				// Accepting a tie resolves the game; the RESULT packet rides
				// the GAME_ENDED event.
				return
			}
			announceTo(l, ctx, sc, wp, opponentOf(e), interactioncb.CharacterInteractionMiniGameAnswerTieBody())
			return
		}
		if e.Body.Accept {
			turnSlot := slotOf(e, opponentOf(e))
			announceToRoom(l, ctx, sc, wp, e, interactioncb.CharacterInteractionMiniGameRetreatAnswerBody(true, retreatStoneCount, turnSlot))
			return
		}
		announceTo(l, ctx, sc, wp, opponentOf(e), interactioncb.CharacterInteractionMiniGameRetreatAnswerBody(false, 0, 0))
	}
}

func handleSkippedEvent(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger, ctx context.Context, e minigame2.StatusEvent[minigame2.SkippedEventBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, e minigame2.StatusEvent[minigame2.SkippedEventBody]) {
		if e.Type != minigame2.EventTypeSkipped {
			return
		}
		if !guard(sc, ctx, e) {
			return
		}
		l.Debugf("Turn skipped in room [%d]. next mover slot [%d].", e.RoomId, e.Body.Who)
		announceToRoom(l, ctx, sc, wp, e, interactioncb.CharacterInteractionMiniGameSkipBody(e.Body.Who))
	}
}

func handleGameEndedEvent(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger, ctx context.Context, e minigame2.StatusEvent[minigame2.GameEndedEventBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, e minigame2.StatusEvent[minigame2.GameEndedEventBody]) {
		if e.Type != minigame2.EventTypeGameEnded {
			return
		}
		if !guard(sc, ctx, e) {
			return
		}
		l.Debugf("Mini-game ended in room [%d]. resultType [%s], winnerSlot [%d].", e.RoomId, e.Body.ResultType, e.Body.WinnerSlot)
		ownerRecord := gameRecord(e.Body.OwnerRecord, e.Body.OwnerScore)
		visitorRecord := gameRecord(e.Body.VisitorRecord, e.Body.VisitorScore)
		announceToRoom(l, ctx, sc, wp, e, interactioncb.CharacterInteractionMiniGameResultBody(e.Body.ResultType, e.Body.WinnerSlot == 1, ownerRecord, visitorRecord))
	}
}

func handleBalloonEvent(sc server.Model, wp writer.Producer) func(l logrus.FieldLogger, ctx context.Context, e minigame2.StatusEvent[minigame2.BalloonEventBody]) {
	return func(l logrus.FieldLogger, ctx context.Context, e minigame2.StatusEvent[minigame2.BalloonEventBody]) {
		if e.Type != minigame2.EventTypeBalloonUpdated {
			return
		}
		if !guard(sc, ctx, e) {
			return
		}
		l.Debugf("Balloon update for mini-game room [%d]. remove [%t], occupancy [%d].", e.RoomId, e.Body.Remove, e.Body.Occupancy)
		if e.Body.Remove {
			announceBalloon(l, ctx, wp, e, interactioncb.MiniRoomBalloonRemoveBody(e.OwnerId))
			return
		}
		announceBalloon(l, ctx, wp, e, interactioncb.MiniRoomBalloonBody(e.OwnerId, e.Body.RoomType, e.RoomId, e.Body.Title, e.Body.HasPassword, e.Body.PieceType, e.Body.Occupancy, gameRoomCapacity, e.Body.InProgress))
	}
}
