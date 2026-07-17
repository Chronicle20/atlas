package game

import (
	"atlas-rps/kafka/message"
	"atlas-rps/kafka/message/rps"
	"context"
	"errors"
	"fmt"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	sharedsaga "github.com/Chronicle20/atlas/libs/atlas-saga"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// ErrSessionNotFound is returned when an operation targets a character with
// no active RPS session in the registry.
var ErrSessionNotFound = errors.New("game: no active session for character")

// ErrInvalidStatus is returned when an operation is attempted against a
// session whose current Status does not permit it.
var ErrInvalidStatus = errors.New("game: session is not in a valid status for this operation")

// ErrLadderNotConfigured is returned by the zero-value LadderProvider
// installed by NewProcessor when no ladder source has been wired in. See the
// LadderProvider doc for why this cannot simply be loaded automatically.
var ErrLadderNotConfigured = errors.New("game: no LadderProvider configured; use NewProcessorWithLadder")

// ErrSagaSubmitterNotConfigured is returned by the zero-value SagaSubmitter
// installed by NewProcessor when no saga submission target has been wired
// in. See the SagaSubmitter doc for why this cannot simply be called
// directly against the local "atlas-rps/saga" package.
var ErrSagaSubmitterNotConfigured = errors.New("game: no SagaSubmitter configured; use NewProcessorWithLadder")

// LadderProvider resolves the reward ladder for the processor's tenant.
//
// It is injected rather than loaded via a direct call to the configuration
// package: configuration.Processor.GetLadder imports "atlas-rps/game" for the
// Ladder/Rung types, so this package importing "atlas-rps/configuration"
// back would create an import cycle. Production wiring (the kafka consumer
// group bootstrap / REST handler bootstrap) is expected to construct the
// processor via NewProcessorWithLadder, supplying a LadderProvider backed by
// configuration.NewProcessor(l, ctx).GetLadder(tenant.Id()). Tests supply a
// fixed stub Ladder directly - no HTTP server required.
type LadderProvider func() (Ladder, error)

// SagaSubmitter submits a fully-built payout saga.Saga to
// atlas-saga-orchestrator's command topic.
//
// It is injected for the same reason LadderProvider is: the local
// "atlas-rps/saga" package wraps this package's Kafka producer composition
// (topic + producer.ProviderImpl), and this package importing it back would
// be unnecessary coupling - "atlas-rps/game" only needs to build a
// libs/atlas-saga Saga value, which it may import directly since that
// shared library has no dependency on atlas-rps. Production wiring
// (main.go's REST factory, kafka/consumer/rps.SagaSubmitterFor) supplies a
// closure backed by saga.NewProcessor(l, ctx).Create(s). Tests supply a
// capturing stub - no live Kafka producer required.
type SagaSubmitter func(s sharedsaga.Saga) error

// ProcessorFactory builds a fully-wired Processor for a single
// request/command, given the caller's logger and context. It exists so that
// composition roots which CAN import both "atlas-rps/game" and
// "atlas-rps/configuration" (main.go, kafka/consumer/rps) can hand this
// package a ready-made constructor - e.g. the REST resource in
// game/resource.go - without game itself ever importing configuration (see
// LadderProvider's doc for why that import is forbidden).
type ProcessorFactory func(l logrus.FieldLogger, ctx context.Context) Processor

// Processor defines the RPS game state-machine operations. Each buffered
// Method(mb, ...) is a pure state transition that also buffers the events it
// produces onto mb; the corresponding MethodAndEmit(...) wraps the buffered
// method via message.EmitWithResult so the buffered events are emitted
// atomically after a successful transition.
type Processor interface {
	// Get returns the active session for the given character, together with
	// the prize currently resolved at its rung (ok=false if the session is
	// fresh (rung 0) or no prize is configured at the current rung). Returns
	// ErrSessionNotFound if no active session exists for the character.
	Get(characterId uint32) (Model, Rung, bool, error)

	// Start disposes any stale session for the character, opens a new
	// rung-0 StatusOpen session, and buffers a GameOpened event.
	Start(mb *message.Buffer, characterId uint32, worldId world.Id, channelId channel.Id, npcId uint32) (Model, error)
	StartAndEmit(characterId uint32, worldId world.Id, channelId channel.Id, npcId uint32) (Model, error)

	// Begin opens the first round of a StatusOpen session (the player clicked
	// "Start" on the board), transitions it to StatusAwaitingSelect, and
	// buffers a RoundStarted event. The client keeps its R/P/S buttons disabled
	// until it receives the START_SELECT frame that RoundStarted drives, so
	// without Begin the round cannot start.
	Begin(mb *message.Buffer, characterId uint32) (Model, error)
	BeginAndEmit(characterId uint32) (Model, error)

	// Select submits the player's throw for the current round, adjudicates
	// it against a drawn opponent throw, and buffers a RoundResult event
	// (plus a terminal GameEnded on a loss).
	Select(mb *message.Buffer, characterId uint32, throw Throw) (Model, error)
	SelectAndEmit(characterId uint32, throw Throw) (Model, error)

	// Continue advances a StatusAwaitingDecision session to the next round,
	// or forces a Collect if the session is already at the ladder's max rung.
	Continue(mb *message.Buffer, characterId uint32) (Model, error)
	ContinueAndEmit(characterId uint32) (Model, error)

	// Collect ends the session from any active status - the client's only
	// "leave" action has no dedicated collect sub-op. From
	// StatusAwaitingDecision it pays the resolved prize at the current rung
	// and buffers GameEnded(collected); from any other active status
	// (nothing won yet, or a rung still being risked) it forfeits with no
	// payout and buffers GameEnded(quit).
	Collect(mb *message.Buffer, characterId uint32) (Model, error)
	CollectAndEmit(characterId uint32) (Model, error)

	// Quit ends the session with no payout and buffers a GameEnded(quit) event.
	Quit(mb *message.Buffer, characterId uint32) (Model, error)
	QuitAndEmit(characterId uint32) (Model, error)

	// Dispose ends the session with no payout, buffering a
	// GameEnded(disconnected) event. If the session is already gone, it is a
	// no-op: no error, no event.
	Dispose(mb *message.Buffer, characterId uint32) (Model, error)
	DisposeAndEmit(characterId uint32) (Model, error)
}

// ProcessorImpl implements the Processor interface.
type ProcessorImpl struct {
	l              logrus.FieldLogger
	ctx            context.Context
	t              tenant.Model
	throwSource    ThrowSource
	ladderProvider LadderProvider
	sagaSubmitter  SagaSubmitter
}

// NewProcessor creates a new processor implementation using the
// server-authoritative DefaultThrowSource. Its LadderProvider and
// SagaSubmitter are unconfigured (see ErrLadderNotConfigured /
// ErrSagaSubmitterNotConfigured) - production bootstrap code and tests that
// need a working ladder/saga submission should use NewProcessorWithLadder
// instead.
func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	t := tenant.MustFromContext(ctx)
	return &ProcessorImpl{
		l:           l,
		ctx:         ctx,
		t:           t,
		throwSource: DefaultThrowSource,
		ladderProvider: func() (Ladder, error) {
			return Ladder{}, ErrLadderNotConfigured
		},
		sagaSubmitter: func(sharedsaga.Saga) error {
			return ErrSagaSubmitterNotConfigured
		},
	}
}

// NewProcessorWithLadder creates a new processor implementation with an
// explicit ThrowSource, LadderProvider, and SagaSubmitter. This is the
// constructor used by tests (deterministic throw sequencing + a stub ladder
// + a capturing/no-op saga submitter, no HTTP server or Kafka broker) and by
// production bootstrap code that wires a LadderProvider backed by
// configuration.NewProcessor(l, ctx).GetLadder(tenant.Id()) and a
// SagaSubmitter backed by saga.NewProcessor(l, ctx).Create(s) (see
// kafka/consumer/rps.SagaSubmitterFor).
func NewProcessorWithLadder(l logrus.FieldLogger, ctx context.Context, throwSource ThrowSource, ladderProvider LadderProvider, sagaSubmitter SagaSubmitter) Processor {
	t := tenant.MustFromContext(ctx)
	return &ProcessorImpl{
		l:              l,
		ctx:            ctx,
		t:              t,
		throwSource:    throwSource,
		ladderProvider: ladderProvider,
		sagaSubmitter:  sagaSubmitter,
	}
}

var _ Processor = (*ProcessorImpl)(nil)

// Get returns the active session for the given character, together with the
// prize currently resolved at its rung.
func (p *ProcessorImpl) Get(characterId uint32) (Model, Rung, bool, error) {
	m, ok := GetRegistry().Get(p.ctx, characterId)
	if !ok {
		return Model{}, Rung{}, false, ErrSessionNotFound
	}

	ladder, err := p.ladderProvider()
	if err != nil {
		return Model{}, Rung{}, false, err
	}
	prize, prizeOk := ladder.PrizeAt(m.Rung())

	return m, prize, prizeOk, nil
}

// Start disposes any stale session for the character, opens a new rung-0
// StatusOpen session, and buffers a GameOpened event. Start now resolves the
// reward ladder to source the ante (entry cost) carried on GameOpened - a
// ladder-provider failure is a real error (the config is unavailable), and
// Start returns it so the entry saga can compensate/refund; it never opens a
// session with a silently-wrong ante.
func (p *ProcessorImpl) Start(mb *message.Buffer, characterId uint32, worldId world.Id, channelId channel.Id, npcId uint32) (Model, error) {
	p.l.Debugf("Starting RPS session for character [%d] at npc [%d].", characterId, npcId)

	ladder, err := p.ladderProvider()
	if err != nil {
		return Model{}, err
	}

	// Dispose of any stale session left behind by a prior game (e.g. an
	// abandoned session the sweeper hasn't reclaimed yet). This is a silent,
	// idempotent cleanup - no GameEnded event is buffered for it, since from
	// the player's perspective they are opening a fresh game.
	GetRegistry().Remove(p.ctx, characterId)

	m, err := NewModelBuilder(p.t).
		SetCharacterId(characterId).
		SetWorldId(worldId).
		SetChannelId(channelId).
		SetNpcId(npcId).
		SetRung(0).
		SetStatus(StatusOpen).
		Build()
	if err != nil {
		return Model{}, err
	}

	GetRegistry().Put(p.ctx, m)

	if err := mb.Put(rps.EnvEventTopic, gameOpenedEventProvider(characterId, worldId, channelId, npcId, ladder.EntryCostMeso)); err != nil {
		return Model{}, err
	}

	return m, nil
}

type startInput struct {
	characterId uint32
	worldId     world.Id
	channelId   channel.Id
	npcId       uint32
}

// StartAndEmit starts a new RPS session and emits its buffered events.
func (p *ProcessorImpl) StartAndEmit(characterId uint32, worldId world.Id, channelId channel.Id, npcId uint32) (Model, error) {
	return message.EmitWithResult[Model, startInput](
		producer.ProviderImpl(p.l)(p.ctx),
	)(func(mb *message.Buffer) func(input startInput) (Model, error) {
		return func(input startInput) (Model, error) {
			return p.Start(mb, input.characterId, input.worldId, input.channelId, input.npcId)
		}
	})(startInput{characterId: characterId, worldId: worldId, channelId: channelId, npcId: npcId})
}

// Begin opens the first round of a StatusOpen session: the player clicked
// "Start" on the board (serverbound RPS_ACTION sub-op 0). It transitions the
// session to StatusAwaitingSelect and buffers a RoundStarted event so the
// channel writes the clientbound START_SELECT frame that enables the client's
// R/P/S buttons. Only valid from StatusOpen - a Begin against any other status
// is a no-op error (the round is already underway or the game is over), which
// the command handler logs without sending a frame.
func (p *ProcessorImpl) Begin(mb *message.Buffer, characterId uint32) (Model, error) {
	m, ok := GetRegistry().Get(p.ctx, characterId)
	if !ok {
		return Model{}, ErrSessionNotFound
	}
	if m.Status() != StatusOpen {
		return Model{}, ErrInvalidStatus
	}

	updated, err := CloneModelBuilder(m).SetStatus(StatusAwaitingSelect).Build()
	if err != nil {
		return Model{}, err
	}
	GetRegistry().Put(p.ctx, updated)

	if err := mb.Put(rps.EnvEventTopic, roundStartedEventProvider(characterId, m.WorldId(), m.ChannelId(), m.Rung())); err != nil {
		return Model{}, err
	}
	return updated, nil
}

// BeginAndEmit opens the first round and emits its buffered events.
func (p *ProcessorImpl) BeginAndEmit(characterId uint32) (Model, error) {
	return message.EmitWithResult[Model, uint32](
		producer.ProviderImpl(p.l)(p.ctx),
	)(func(mb *message.Buffer) func(characterId uint32) (Model, error) {
		return func(characterId uint32) (Model, error) {
			return p.Begin(mb, characterId)
		}
	})(characterId)
}

// Select submits the player's throw, draws an opponent throw via the
// injected ThrowSource, adjudicates the round, and applies the resulting
// state transition:
//   - win: rung+1, StatusAwaitingDecision, buffers RoundResult{win, prize at new rung}.
//   - tie: rung unchanged, StatusAwaitingSelect, buffers RoundResult{tie}.
//   - loss: session removed, buffers RoundResult{lose} then GameEnded{lost}.
func (p *ProcessorImpl) Select(mb *message.Buffer, characterId uint32, throw Throw) (Model, error) {
	m, ok := GetRegistry().Get(p.ctx, characterId)
	if !ok {
		return Model{}, ErrSessionNotFound
	}
	if m.Status() != StatusOpen && m.Status() != StatusAwaitingSelect {
		return Model{}, ErrInvalidStatus
	}

	opponentThrow := p.throwSource()
	outcome := Adjudicate(throw, opponentThrow)

	p.l.Debugf("Character [%d] threw [%d] against opponent [%d] in RPS session, outcome [%d].", characterId, throw, opponentThrow, outcome)

	switch outcome {
	case OutcomeWin:
		ladder, err := p.ladderProvider()
		if err != nil {
			return Model{}, err
		}
		newRung := m.Rung() + 1
		prize, prizeOk := ladder.PrizeAt(newRung)

		updated, err := CloneModelBuilder(m).
			SetRung(newRung).
			SetStatus(StatusAwaitingDecision).
			SetLastThrow(throw).
			Build()
		if err != nil {
			return Model{}, err
		}
		GetRegistry().Put(p.ctx, updated)

		if err := mb.Put(rps.EnvEventTopic, roundResultEventProvider(characterId, m.WorldId(), m.ChannelId(), opponentThrow, outcome, newRung, toEventPrize(prize, prizeOk))); err != nil {
			return Model{}, err
		}
		return updated, nil

	case OutcomeTie:
		updated, err := CloneModelBuilder(m).
			SetStatus(StatusAwaitingSelect).
			SetLastThrow(throw).
			Build()
		if err != nil {
			return Model{}, err
		}
		GetRegistry().Put(p.ctx, updated)

		if err := mb.Put(rps.EnvEventTopic, roundResultEventProvider(characterId, m.WorldId(), m.ChannelId(), opponentThrow, outcome, m.Rung(), rps.Prize{})); err != nil {
			return Model{}, err
		}
		return updated, nil

	default: // OutcomeLose
		GetRegistry().Remove(p.ctx, characterId)

		if err := mb.Put(rps.EnvEventTopic, roundResultEventProvider(characterId, m.WorldId(), m.ChannelId(), opponentThrow, outcome, m.Rung(), rps.Prize{})); err != nil {
			return Model{}, err
		}
		if err := mb.Put(rps.EnvEventTopic, gameEndedEventProvider(characterId, m.WorldId(), m.ChannelId(), rps.ReasonLost, nil)); err != nil {
			return Model{}, err
		}

		return CloneModelBuilder(m).SetStatus(StatusEnded).SetLastThrow(throw).Build()
	}
}

type selectInput struct {
	characterId uint32
	throw       Throw
}

// SelectAndEmit submits the player's throw for the current round and emits
// its buffered events.
func (p *ProcessorImpl) SelectAndEmit(characterId uint32, throw Throw) (Model, error) {
	return message.EmitWithResult[Model, selectInput](
		producer.ProviderImpl(p.l)(p.ctx),
	)(func(mb *message.Buffer) func(input selectInput) (Model, error) {
		return func(input selectInput) (Model, error) {
			return p.Select(mb, input.characterId, input.throw)
		}
	})(selectInput{characterId: characterId, throw: throw})
}

// Continue advances a StatusAwaitingDecision session to the next round. If
// the session is already at the ladder's max configured rung, there is no
// further round to play, so Continue forces a Collect instead.
func (p *ProcessorImpl) Continue(mb *message.Buffer, characterId uint32) (Model, error) {
	m, ok := GetRegistry().Get(p.ctx, characterId)
	if !ok {
		return Model{}, ErrSessionNotFound
	}
	if m.Status() != StatusAwaitingDecision {
		return Model{}, ErrInvalidStatus
	}

	ladder, err := p.ladderProvider()
	if err != nil {
		return Model{}, err
	}

	if ladder.IsMax(m.Rung()) {
		p.l.Debugf("Character [%d] is at max rung [%d]; forcing collect on continue.", characterId, m.Rung())
		return p.Collect(mb, characterId)
	}

	updated, err := CloneModelBuilder(m).SetStatus(StatusAwaitingSelect).Build()
	if err != nil {
		return Model{}, err
	}
	GetRegistry().Put(p.ctx, updated)

	// Open the next round: the channel writes START_SELECT (mode 9), which
	// re-enables the client's R/P/S buttons for the new rung. Mirrors Begin's
	// event for the first round.
	if err := mb.Put(rps.EnvEventTopic, roundStartedEventProvider(characterId, m.WorldId(), m.ChannelId(), updated.Rung())); err != nil {
		return Model{}, err
	}
	return updated, nil
}

// ContinueAndEmit advances the session to the next round (or forces a
// collect at max rung) and emits any buffered events.
func (p *ProcessorImpl) ContinueAndEmit(characterId uint32) (Model, error) {
	return message.EmitWithResult[Model, uint32](
		producer.ProviderImpl(p.l)(p.ctx),
	)(func(mb *message.Buffer) func(characterId uint32) (Model, error) {
		return func(characterId uint32) (Model, error) {
			return p.Continue(mb, characterId)
		}
	})(characterId)
}

// Collect ends the session for the given character from any active status.
// The client's only "leave the game" action (Exit) has no dedicated
// "collect" sub-op - it maps to Collect regardless of the session's current
// status (IDA-verified: there is no separate collect sub-op). Behavior
// depends on the status found:
//
//   - StatusAwaitingDecision (a win, not yet risked further): resolves the
//     prize at the session's current rung, submits the payout saga for any
//     non-zero prize components, removes the session, and buffers a
//     GameEnded{collected, prize} event. If saga submission fails, the
//     session is left in place (not removed, no event buffered) so a
//     retried Collect can attempt the payout again.
//   - StatusOpen or StatusAwaitingSelect (nothing won yet, or the current
//     rung is still being risked mid-round): no payout - the session is
//     removed and a GameEnded{quit} event is buffered. StatusEnded, which
//     should never actually be found in the registry, is also treated as
//     this forfeit branch defensively.
func (p *ProcessorImpl) Collect(mb *message.Buffer, characterId uint32) (Model, error) {
	m, ok := GetRegistry().Get(p.ctx, characterId)
	if !ok {
		return Model{}, ErrSessionNotFound
	}

	if m.Status() != StatusAwaitingDecision {
		GetRegistry().Remove(p.ctx, characterId)

		if err := mb.Put(rps.EnvEventTopic, gameEndedEventProvider(characterId, m.WorldId(), m.ChannelId(), rps.ReasonQuit, nil)); err != nil {
			return Model{}, err
		}

		return CloneModelBuilder(m).SetStatus(StatusEnded).Build()
	}

	ladder, err := p.ladderProvider()
	if err != nil {
		return Model{}, err
	}
	prize, prizeOk := ladder.PrizeAt(m.Rung())

	var grantedPrize *rps.Prize
	if prizeOk {
		grantedPrize = &rps.Prize{ItemId: prize.ItemId, Quantity: prize.Quantity, Meso: prize.Meso}
		if s, hasSteps := buildPayoutSaga(m, prize); hasSteps {
			if err := p.sagaSubmitter(s); err != nil {
				return Model{}, err
			}
		}
	}

	GetRegistry().Remove(p.ctx, characterId)

	if err := mb.Put(rps.EnvEventTopic, gameEndedEventProvider(characterId, m.WorldId(), m.ChannelId(), rps.ReasonCollected, grantedPrize)); err != nil {
		return Model{}, err
	}

	return CloneModelBuilder(m).SetStatus(StatusEnded).Build()
}

// buildPayoutSaga constructs the payout saga for a resolved prize: an
// AwardMesos step is included only when the prize grants a positive meso
// amount, and an AwardAsset step only when it grants a positive item
// quantity. hasSteps is false (and the returned Saga is the zero value) when
// the prize grants neither - e.g. a rung configured with meso=0 and
// itemId=0 - in which case no saga should be submitted.
func buildPayoutSaga(m Model, prize Rung) (sharedsaga.Saga, bool) {
	b := sharedsaga.NewBuilder().
		SetTransactionId(uuid.New()).
		SetSagaType(sharedsaga.InventoryTransaction).
		SetInitiatedBy(fmt.Sprintf("NPC_%d_rps_payout", m.NpcId()))

	steps := 0
	if prize.Meso > 0 {
		b.AddStep("award_mesos", sharedsaga.Pending, sharedsaga.AwardMesos, sharedsaga.AwardMesosPayload{
			CharacterId: m.CharacterId(),
			WorldId:     m.WorldId(),
			ChannelId:   m.ChannelId(),
			ActorId:     m.NpcId(),
			ActorType:   "NPC",
			Amount:      int32(prize.Meso),
			ShowEffect:  true,
		})
		steps++
	}
	if prize.ItemId != 0 && prize.Quantity > 0 {
		b.AddStep("award_asset", sharedsaga.Pending, sharedsaga.AwardAsset, sharedsaga.AwardItemActionPayload{
			CharacterId: m.CharacterId(),
			Item: sharedsaga.ItemPayload{
				TemplateId: uint32(prize.ItemId),
				Quantity:   prize.Quantity,
			},
			ShowEffect: true,
		})
		steps++
	}
	if steps == 0 {
		return sharedsaga.Saga{}, false
	}
	return b.Build(), true
}

// CollectAndEmit resolves and banks the current prize, ending the session,
// and emits its buffered events.
func (p *ProcessorImpl) CollectAndEmit(characterId uint32) (Model, error) {
	return message.EmitWithResult[Model, uint32](
		producer.ProviderImpl(p.l)(p.ctx),
	)(func(mb *message.Buffer) func(characterId uint32) (Model, error) {
		return func(characterId uint32) (Model, error) {
			return p.Collect(mb, characterId)
		}
	})(characterId)
}

// Quit ends the session with no payout, buffering a GameEnded{quit} event.
func (p *ProcessorImpl) Quit(mb *message.Buffer, characterId uint32) (Model, error) {
	m, ok := GetRegistry().Get(p.ctx, characterId)
	if !ok {
		return Model{}, ErrSessionNotFound
	}

	GetRegistry().Remove(p.ctx, characterId)

	if err := mb.Put(rps.EnvEventTopic, gameEndedEventProvider(characterId, m.WorldId(), m.ChannelId(), rps.ReasonQuit, nil)); err != nil {
		return Model{}, err
	}

	return CloneModelBuilder(m).SetStatus(StatusEnded).Build()
}

// QuitAndEmit ends the session with no payout and emits its buffered events.
func (p *ProcessorImpl) QuitAndEmit(characterId uint32) (Model, error) {
	return message.EmitWithResult[Model, uint32](
		producer.ProviderImpl(p.l)(p.ctx),
	)(func(mb *message.Buffer) func(characterId uint32) (Model, error) {
		return func(characterId uint32) (Model, error) {
			return p.Quit(mb, characterId)
		}
	})(characterId)
}

// Dispose ends the session with no payout, buffering a
// GameEnded{disconnected} event. If the session is already gone (e.g. it was
// already collected, quit, or reclaimed by the TTL sweeper), Dispose is a
// no-op: it returns the zero Model with no error and buffers no event.
func (p *ProcessorImpl) Dispose(mb *message.Buffer, characterId uint32) (Model, error) {
	m, ok := GetRegistry().Get(p.ctx, characterId)
	if !ok {
		return Model{}, nil
	}

	GetRegistry().Remove(p.ctx, characterId)

	if err := mb.Put(rps.EnvEventTopic, gameEndedEventProvider(characterId, m.WorldId(), m.ChannelId(), rps.ReasonDisconnected, nil)); err != nil {
		return Model{}, err
	}

	return CloneModelBuilder(m).SetStatus(StatusEnded).Build()
}

// DisposeAndEmit disposes of the session (if any) on disconnect and emits
// any buffered event.
func (p *ProcessorImpl) DisposeAndEmit(characterId uint32) (Model, error) {
	return message.EmitWithResult[Model, uint32](
		producer.ProviderImpl(p.l)(p.ctx),
	)(func(mb *message.Buffer) func(characterId uint32) (Model, error) {
		return func(characterId uint32) (Model, error) {
			return p.Dispose(mb, characterId)
		}
	})(characterId)
}

// toEventPrize converts a resolved ladder Rung into the kafka event Prize
// shape. If no prize was configured at the resolved rung (ok=false), it
// returns the zero-value Prize - the round was still won, it simply carries
// no reward at this rung.
func toEventPrize(r Rung, ok bool) rps.Prize {
	if !ok {
		return rps.Prize{}
	}
	return rps.Prize{ItemId: r.ItemId, Quantity: r.Quantity, Meso: r.Meso}
}
