package game

import (
	"atlas-rps/kafka/message"
	"atlas-rps/kafka/message/rps"
	"atlas-rps/kafka/producer"
	"context"
	"errors"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
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

	// Select submits the player's throw for the current round, adjudicates
	// it against a drawn opponent throw, and buffers a RoundResult event
	// (plus a terminal GameEnded on a loss).
	Select(mb *message.Buffer, characterId uint32, throw Throw) (Model, error)
	SelectAndEmit(characterId uint32, throw Throw) (Model, error)

	// Continue advances a StatusAwaitingDecision session to the next round,
	// or forces a Collect if the session is already at the ladder's max rung.
	Continue(mb *message.Buffer, characterId uint32) (Model, error)
	ContinueAndEmit(characterId uint32) (Model, error)

	// Collect resolves the prize at the session's current rung, ends the
	// session, and buffers a GameEnded(collected) event.
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
}

// NewProcessor creates a new processor implementation using the
// server-authoritative DefaultThrowSource. Its LadderProvider is
// unconfigured (see ErrLadderNotConfigured) - production bootstrap code and
// tests that need a working ladder should use NewProcessorWithLadder
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
	}
}

// NewProcessorWithLadder creates a new processor implementation with an
// explicit ThrowSource and LadderProvider. This is the constructor used by
// tests (deterministic throw sequencing + a stub ladder, no HTTP server) and
// by production bootstrap code that wires a LadderProvider backed by
// configuration.NewProcessor(l, ctx).GetLadder(tenant.Id()).
func NewProcessorWithLadder(l logrus.FieldLogger, ctx context.Context, throwSource ThrowSource, ladderProvider LadderProvider) Processor {
	t := tenant.MustFromContext(ctx)
	return &ProcessorImpl{
		l:              l,
		ctx:            ctx,
		t:              t,
		throwSource:    throwSource,
		ladderProvider: ladderProvider,
	}
}

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
// StatusOpen session, and buffers a GameOpened event.
func (p *ProcessorImpl) Start(mb *message.Buffer, characterId uint32, worldId world.Id, channelId channel.Id, npcId uint32) (Model, error) {
	p.l.Debugf("Starting RPS session for character [%d] at npc [%d].", characterId, npcId)

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

	if err := mb.Put(rps.EnvEventTopic, gameOpenedEventProvider(characterId, worldId, channelId, npcId)); err != nil {
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

// Collect resolves the prize at the session's current rung, removes the
// session, and buffers a GameEnded{collected, prize} event. Note: the
// payout saga submission (Task 12) is intentionally NOT part of this
// method; that task modifies Collect to add it.
func (p *ProcessorImpl) Collect(mb *message.Buffer, characterId uint32) (Model, error) {
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
	prize, prizeOk := ladder.PrizeAt(m.Rung())

	GetRegistry().Remove(p.ctx, characterId)

	var grantedPrize *rps.Prize
	if prizeOk {
		grantedPrize = &rps.Prize{ItemId: prize.ItemId, Quantity: prize.Quantity, Meso: prize.Meso}
	}

	if err := mb.Put(rps.EnvEventTopic, gameEndedEventProvider(characterId, m.WorldId(), m.ChannelId(), rps.ReasonCollected, grantedPrize)); err != nil {
		return Model{}, err
	}

	return CloneModelBuilder(m).SetStatus(StatusEnded).Build()
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
