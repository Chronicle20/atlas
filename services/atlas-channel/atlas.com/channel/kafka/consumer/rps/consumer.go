// Package rps translates atlas-rps EVENT_TOPIC_RPS events into the
// clientbound RPS_GAME frames (rps.RPSGameOpenBody/RPSGameResultBody/
// RPSGameEndBody, libs/atlas-packet/rps/operation_body.go) and announces
// them to the acting character's session. See
// docs/tasks/task-132-rps-npc-game for the event -> frame mapping and the
// IDA-verified straightVictoryCount sign encoding.
package rps

import (
	rpsmsg "atlas-channel/kafka/message/rps"
	consumer2 "atlas-channel/kafka/consumer"
	"atlas-channel/listener"
	"atlas-channel/server"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"
	"math"

	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	rpspkt "github.com/Chronicle20/atlas/libs/atlas-packet/rps"
	rpscb "github.com/Chronicle20/atlas/libs/atlas-packet/rps/clientbound"
	"github.com/Chronicle20/atlas/libs/atlas-socket/packet"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("rps_game_event")(rpsmsg.EnvEventTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser), consumer.SetStartOffset(kafka.LastOffset))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
	return func(sc server.Model) func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
		return func(wp writer.Producer) func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
			return func(rf func(topic string, handler handler.Handler) (string, error)) ([]listener.HandlerHandle, error) {
				var t string
				var handles []listener.HandlerHandle
				t, _ = topic.EnvProvider(l)(rpsmsg.EnvEventTopic)()
				id, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleGameOpenedEvent(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleRoundStartedEvent(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleRoundResultEvent(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleGameEndedEvent(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				return handles, nil
			}
		}
	}
}

// rpsAnnouncer is the channel-side seam that resolves the character's
// session and announces the given RPS_GAME writer body to it. Package-level
// var so tests can swap in a recording stub without a live net.Conn or a
// real writer registry (mirrors the mount consumer's seam pattern).
var rpsAnnouncer = func(l logrus.FieldLogger, ctx context.Context, wp writer.Producer, sc server.Model, characterId uint32, body packet.Encode) {
	err := session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(characterId,
		session.Announce(l)(ctx)(wp)(rpscb.RPSGameWriter)(body))
	if err != nil {
		l.WithError(err).Errorf("Unable to announce RPS_GAME frame to character [%d].", characterId)
	}
}

// handleGameOpenedEvent translates a GAME_OPENED event into the OPEN frame.
// Body: the NPC template id - Decode4 on the client, which loads the dealer's
// face (Npc/{id}.img) for the fee-confirm dialog. NOT the ante (a static string
// with no amount); sending the ante here makes the client look up a
// non-existent Npc.img and crash (STG_E_FILENOTFOUND).
func handleGameOpenedEvent(sc server.Model, wp writer.Producer) message.Handler[rpsmsg.Event[rpsmsg.GameOpenedEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e rpsmsg.Event[rpsmsg.GameOpenedEventBody]) {
		if e.Type != rpsmsg.EventTypeGameOpened {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId) {
			return
		}

		l.Debugf("Received RPS GAME_OPENED event for character [%d], NPC [%d], ante [%d].", e.CharacterId, e.Body.NpcId, e.Body.Ante)

		rpsAnnouncer(l, ctx, wp, sc, e.CharacterId, rpspkt.RPSGameOpenBody(e.Body.NpcId))
	}
}

// handleRoundStartedEvent translates a ROUND_STARTED event into the
// START_SELECT frame (mode 9). The frame is bodyless - on receipt the client
// enables its R/P/S buttons and arms the 30s selection timer. atlas-rps emits
// ROUND_STARTED on the first round (the player's START sub-op) and on each
// round it advances to via CONTINUE; a tie does NOT emit it (the client
// re-enables selection locally), so a tie produces only a RESULT frame.
func handleRoundStartedEvent(sc server.Model, wp writer.Producer) message.Handler[rpsmsg.Event[rpsmsg.RoundStartedEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e rpsmsg.Event[rpsmsg.RoundStartedEventBody]) {
		if e.Type != rpsmsg.EventTypeRoundStarted {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId) {
			return
		}

		l.Debugf("Received RPS ROUND_STARTED event for character [%d], rung [%d].", e.CharacterId, e.Body.Rung)

		rpsAnnouncer(l, ctx, wp, sc, e.CharacterId, rpspkt.RPSGameStartSelectBody())
	}
}

// handleRoundResultEvent translates a ROUND_RESULT event into the RESULT
// frame. npcThrow is the raw (unremapped) OpponentThrow byte.
// straightVictoryCount is a SIGNED int8 whose SIGN is what the client keys
// win/lose on (IDA-verified, CRPSGameDlg::OnPacket#RESULT):
//   - Win/Tie  -> +Rung (non-negative; the current/unchanged streak)
//   - Lose     -> -1 (negative; magnitude is display-only, sign is what
//     matters)
func handleRoundResultEvent(sc server.Model, wp writer.Producer) message.Handler[rpsmsg.Event[rpsmsg.RoundResultEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e rpsmsg.Event[rpsmsg.RoundResultEventBody]) {
		if e.Type != rpsmsg.EventTypeRoundResult {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId) {
			return
		}

		var straightVictoryCount int8
		switch e.Body.Outcome {
		case rpsmsg.OutcomeWin, rpsmsg.OutcomeTie:
			straightVictoryCount = clampVictoryCount(e.Body.Rung)
		default: // rpsmsg.OutcomeLose
			straightVictoryCount = -1
		}

		l.Debugf("Received RPS ROUND_RESULT event for character [%d], outcome [%d], rung [%d], npcThrow [%d] -> straightVictoryCount [%d].",
			e.CharacterId, e.Body.Outcome, e.Body.Rung, e.Body.OpponentThrow, straightVictoryCount)

		rpsAnnouncer(l, ctx, wp, sc, e.CharacterId, rpspkt.RPSGameResultBody(e.Body.OpponentThrow, straightVictoryCount))
	}
}

// clampVictoryCount converts a non-negative winning/tie rung into the signed
// int8 straightVictoryCount the RESULT frame carries, clamping to
// math.MaxInt8 (127) so a large tenant-configured ladder can never overflow
// int8 and flip the SIGN negative - which the client reads as a LOSS (see
// libs/atlas-packet/rps/clientbound/operation.go RESULT comment: the client
// branches on `straightVictoryCount < 0`). Rung is always >= 0 on a
// win/tie; the clamp only guards the upper bound. The magnitude is
// display-only, so saturating at 127 is safe.
func clampVictoryCount(rung int) int8 {
	if rung > math.MaxInt8 {
		return math.MaxInt8
	}
	if rung < 0 {
		return 0
	}
	return int8(rung)
}

// handleGameEndedEvent translates a GAME_ENDED event into the bodyless END
// frame. Reason/GrantedPrize are not wire fields - the payout (if any) was
// already granted by the collect saga; the END frame itself carries no data.
func handleGameEndedEvent(sc server.Model, wp writer.Producer) message.Handler[rpsmsg.Event[rpsmsg.GameEndedEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e rpsmsg.Event[rpsmsg.GameEndedEventBody]) {
		if e.Type != rpsmsg.EventTypeGameEnded {
			return
		}

		if !sc.Is(tenant.MustFromContext(ctx), e.WorldId, e.ChannelId) {
			return
		}

		l.Debugf("Received RPS GAME_ENDED event for character [%d], reason [%s].", e.CharacterId, e.Body.Reason)

		rpsAnnouncer(l, ctx, wp, sc, e.CharacterId, rpspkt.RPSGameEndBody())
	}
}
