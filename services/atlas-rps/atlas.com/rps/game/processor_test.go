package game_test

import (
	"encoding/json"
	"errors"
	"testing"

	"atlas-rps/game"
	"atlas-rps/kafka/message"
	"atlas-rps/kafka/message/rps"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	sharedsaga "github.com/Chronicle20/atlas/libs/atlas-saga"
	"github.com/segmentio/kafka-go"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fixedThrows returns a game.ThrowSource that plays back seq in order,
// repeating the final entry once exhausted, for deterministic test control
// over the opponent's draws across a multi-round scenario.
func fixedThrows(seq ...game.Throw) game.ThrowSource {
	i := 0
	return func() game.Throw {
		t := seq[i]
		if i < len(seq)-1 {
			i++
		}
		return t
	}
}

func testLogger() logrus.FieldLogger {
	l := logrus.New()
	l.SetLevel(logrus.PanicLevel)
	return l
}

// twoRungLadder has prizes configured at rungs 1 and 2 (max rung 2).
func twoRungLadder() game.Ladder {
	return game.Ladder{
		EntryCostMeso: 1000,
		Rungs: []game.Rung{
			{Rung: 1, ItemId: item.Id(4000000), Quantity: 1, Meso: 0},
			{Rung: 2, ItemId: item.Id(4000001), Quantity: 1, Meso: 500},
		},
	}
}

// oneRungLadder has a single prize at rung 1, so rung 1 is the max rung.
func oneRungLadder() game.Ladder {
	return game.Ladder{
		EntryCostMeso: 1000,
		Rungs: []game.Rung{
			{Rung: 1, ItemId: item.Id(4000000), Quantity: 1, Meso: 0},
		},
	}
}

func ladderProviderFor(l game.Ladder) game.LadderProvider {
	return func() (game.Ladder, error) {
		return l, nil
	}
}

// erroringLadderProvider is a game.LadderProvider stub that always fails,
// for tests asserting a caller correctly propagates a ladder-resolution
// error rather than proceeding with a zero-value ladder.
func erroringLadderProvider(err error) game.LadderProvider {
	return func() (game.Ladder, error) {
		return game.Ladder{}, err
	}
}

// noopSagaSubmitter is a game.SagaSubmitter stub for tests that don't
// exercise the payout path (it must still be non-nil to satisfy
// NewProcessorWithLadder).
func noopSagaSubmitter() game.SagaSubmitter {
	return func(sharedsaga.Saga) error { return nil }
}

// capturingSagaSubmitter is a game.SagaSubmitter stub that appends every
// submitted Saga to *dst, for tests asserting on the payout saga's shape.
func capturingSagaSubmitter(dst *[]sharedsaga.Saga) game.SagaSubmitter {
	return func(s sharedsaga.Saga) error {
		*dst = append(*dst, s)
		return nil
	}
}

// erroringSagaSubmitter is a game.SagaSubmitter stub that always fails and
// records how many times it was invoked, for the payout submit-failure
// retry-safety test.
func erroringSagaSubmitter(err error, calls *int) game.SagaSubmitter {
	return func(sharedsaga.Saga) error {
		*calls++
		return err
	}
}

// countingSagaSubmitter is a game.SagaSubmitter stub that only records how
// many times it was invoked, for tests asserting a saga was NOT submitted.
func countingSagaSubmitter(calls *int) game.SagaSubmitter {
	return func(sharedsaga.Saga) error {
		*calls++
		return nil
	}
}

type eventEnvelope struct {
	Type string `json:"type"`
}

func decodeEventType(t *testing.T, msg kafka.Message) string {
	t.Helper()
	var e eventEnvelope
	require.NoError(t, json.Unmarshal(msg.Value, &e))
	return e.Type
}

func decodeGameOpened(t *testing.T, msg kafka.Message) rps.Event[rps.GameOpenedEventBody] {
	t.Helper()
	var e rps.Event[rps.GameOpenedEventBody]
	require.NoError(t, json.Unmarshal(msg.Value, &e))
	return e
}

func decodeRoundResult(t *testing.T, msg kafka.Message) rps.Event[rps.RoundResultEventBody] {
	t.Helper()
	var e rps.Event[rps.RoundResultEventBody]
	require.NoError(t, json.Unmarshal(msg.Value, &e))
	return e
}

func decodeGameEnded(t *testing.T, msg kafka.Message) rps.Event[rps.GameEndedEventBody] {
	t.Helper()
	var e rps.Event[rps.GameEndedEventBody]
	require.NoError(t, json.Unmarshal(msg.Value, &e))
	return e
}

const (
	testWorldId   = world.Id(0)
	testChannelId = channel.Id(1)
	testNpcId     = uint32(9020000)
)

// TestProcessor_FullHappyPath drives Start -> Select(win) -> Continue ->
// Select(tie) -> Select(win) -> Collect against a 2-rung ladder, asserting
// the rung/status transitions at each step and the full ordered sequence of
// buffered events.
func TestProcessor_FullHappyPath(t *testing.T) {
	setupRegistryTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)
	characterId := uint32(1001)

	// Rock always beats Scissors (win) and ties Rock (tie).
	throws := fixedThrows(game.ThrowScissors, game.ThrowRock, game.ThrowScissors)
	p := game.NewProcessorWithLadder(testLogger(), ctx, throws, ladderProviderFor(twoRungLadder()), noopSagaSubmitter())

	mb := message.NewBuffer()

	m, err := p.Start(mb, characterId, testWorldId, testChannelId, testNpcId)
	require.NoError(t, err)
	assert.Equal(t, game.StatusOpen, m.Status())
	assert.Equal(t, 0, m.Rung())

	m, err = p.Select(mb, characterId, game.ThrowRock) // win
	require.NoError(t, err)
	assert.Equal(t, game.StatusAwaitingDecision, m.Status())
	assert.Equal(t, 1, m.Rung())

	m, err = p.Continue(mb, characterId)
	require.NoError(t, err)
	assert.Equal(t, game.StatusAwaitingSelect, m.Status())
	assert.Equal(t, 1, m.Rung())

	m, err = p.Select(mb, characterId, game.ThrowRock) // tie
	require.NoError(t, err)
	assert.Equal(t, game.StatusAwaitingSelect, m.Status())
	assert.Equal(t, 1, m.Rung())

	m, err = p.Select(mb, characterId, game.ThrowRock) // win
	require.NoError(t, err)
	assert.Equal(t, game.StatusAwaitingDecision, m.Status())
	assert.Equal(t, 2, m.Rung())

	m, err = p.Collect(mb, characterId)
	require.NoError(t, err)
	assert.Equal(t, game.StatusEnded, m.Status())

	_, found := game.GetRegistry().Get(ctx, characterId)
	assert.False(t, found, "session should be removed from the registry after Collect")

	msgs := mb.GetAll()[rps.EnvEventTopic]
	require.Len(t, msgs, 5, "expected GameOpened, RoundResult(win), RoundResult(tie), RoundResult(win), GameEnded(collected)")

	assert.Equal(t, rps.EventTypeGameOpened, decodeEventType(t, msgs[0]))
	opened := decodeGameOpened(t, msgs[0])
	assert.Equal(t, testNpcId, opened.Body.NpcId)
	assert.Equal(t, uint32(1000), opened.Body.Ante, "ante should be sourced from the ladder's EntryCostMeso")

	assert.Equal(t, rps.EventTypeRoundResult, decodeEventType(t, msgs[1]))
	round1 := decodeRoundResult(t, msgs[1])
	assert.Equal(t, byte(game.ThrowScissors), round1.Body.OpponentThrow)
	assert.Equal(t, int(game.OutcomeWin), round1.Body.Outcome)
	assert.Equal(t, 1, round1.Body.Rung)
	assert.Equal(t, uint32(4000000), uint32(round1.Body.Prize.ItemId))

	assert.Equal(t, rps.EventTypeRoundResult, decodeEventType(t, msgs[2]))
	round2 := decodeRoundResult(t, msgs[2])
	assert.Equal(t, byte(game.ThrowRock), round2.Body.OpponentThrow)
	assert.Equal(t, int(game.OutcomeTie), round2.Body.Outcome)
	assert.Equal(t, 1, round2.Body.Rung)

	assert.Equal(t, rps.EventTypeRoundResult, decodeEventType(t, msgs[3]))
	round3 := decodeRoundResult(t, msgs[3])
	assert.Equal(t, byte(game.ThrowScissors), round3.Body.OpponentThrow)
	assert.Equal(t, int(game.OutcomeWin), round3.Body.Outcome)
	assert.Equal(t, 2, round3.Body.Rung)
	assert.Equal(t, uint32(4000001), uint32(round3.Body.Prize.ItemId))

	assert.Equal(t, rps.EventTypeGameEnded, decodeEventType(t, msgs[4]))
	ended := decodeGameEnded(t, msgs[4])
	assert.Equal(t, rps.ReasonCollected, ended.Body.Reason)
	if assert.NotNil(t, ended.Body.GrantedPrize) {
		assert.Equal(t, uint32(4000001), uint32(ended.Body.GrantedPrize.ItemId))
	}
}

// TestProcessor_Select_Loss verifies that a losing round removes the
// session and buffers RoundResult{lose} followed by GameEnded{lost}, with no
// granted prize.
func TestProcessor_Select_Loss(t *testing.T) {
	setupRegistryTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)
	characterId := uint32(1002)

	// Paper beats Rock, so the player (throwing Rock) loses.
	throws := fixedThrows(game.ThrowPaper)
	p := game.NewProcessorWithLadder(testLogger(), ctx, throws, ladderProviderFor(twoRungLadder()), noopSagaSubmitter())

	mb := message.NewBuffer()
	_, err := p.Start(mb, characterId, testWorldId, testChannelId, testNpcId)
	require.NoError(t, err)

	m, err := p.Select(mb, characterId, game.ThrowRock)
	require.NoError(t, err)
	assert.Equal(t, game.StatusEnded, m.Status())

	_, found := game.GetRegistry().Get(ctx, characterId)
	assert.False(t, found, "session should be removed from the registry after a loss")

	msgs := mb.GetAll()[rps.EnvEventTopic]
	require.Len(t, msgs, 3, "expected GameOpened, RoundResult(lose), GameEnded(lost)")

	assert.Equal(t, rps.EventTypeRoundResult, decodeEventType(t, msgs[1]))
	round := decodeRoundResult(t, msgs[1])
	assert.Equal(t, int(game.OutcomeLose), round.Body.Outcome)

	assert.Equal(t, rps.EventTypeGameEnded, decodeEventType(t, msgs[2]))
	ended := decodeGameEnded(t, msgs[2])
	assert.Equal(t, rps.ReasonLost, ended.Body.Reason)
	assert.Nil(t, ended.Body.GrantedPrize)
}

// TestProcessor_Continue_ForcesCollectAtMaxRung verifies that Continue, when
// called at the ladder's highest configured rung, transparently performs a
// Collect instead of reopening the AWAITING_SELECT state.
func TestProcessor_Continue_ForcesCollectAtMaxRung(t *testing.T) {
	setupRegistryTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)
	characterId := uint32(1003)

	throws := fixedThrows(game.ThrowScissors) // win
	p := game.NewProcessorWithLadder(testLogger(), ctx, throws, ladderProviderFor(oneRungLadder()), noopSagaSubmitter())

	mb := message.NewBuffer()
	_, err := p.Start(mb, characterId, testWorldId, testChannelId, testNpcId)
	require.NoError(t, err)

	m, err := p.Select(mb, characterId, game.ThrowRock) // win -> rung 1, which is max
	require.NoError(t, err)
	assert.Equal(t, game.StatusAwaitingDecision, m.Status())
	assert.Equal(t, 1, m.Rung())

	m, err = p.Continue(mb, characterId)
	require.NoError(t, err)
	assert.Equal(t, game.StatusEnded, m.Status(), "Continue at max rung should force a Collect")

	_, found := game.GetRegistry().Get(ctx, characterId)
	assert.False(t, found, "session should be removed from the registry after the forced collect")

	msgs := mb.GetAll()[rps.EnvEventTopic]
	require.Len(t, msgs, 3, "expected GameOpened, RoundResult(win), GameEnded(collected) - no separate Continue event")

	assert.Equal(t, rps.EventTypeGameEnded, decodeEventType(t, msgs[2]))
	ended := decodeGameEnded(t, msgs[2])
	assert.Equal(t, rps.ReasonCollected, ended.Body.Reason)
	if assert.NotNil(t, ended.Body.GrantedPrize) {
		assert.Equal(t, uint32(4000000), uint32(ended.Body.GrantedPrize.ItemId))
	}
}

// TestProcessor_Quit_NoPayout verifies that Quit ends the session with no
// granted prize regardless of rung.
func TestProcessor_Quit_NoPayout(t *testing.T) {
	setupRegistryTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)
	characterId := uint32(1004)

	throws := fixedThrows(game.ThrowScissors) // win
	p := game.NewProcessorWithLadder(testLogger(), ctx, throws, ladderProviderFor(twoRungLadder()), noopSagaSubmitter())

	mb := message.NewBuffer()
	_, err := p.Start(mb, characterId, testWorldId, testChannelId, testNpcId)
	require.NoError(t, err)

	_, err = p.Select(mb, characterId, game.ThrowRock) // win -> rung 1, AWAITING_DECISION
	require.NoError(t, err)

	m, err := p.Quit(mb, characterId)
	require.NoError(t, err)
	assert.Equal(t, game.StatusEnded, m.Status())

	_, found := game.GetRegistry().Get(ctx, characterId)
	assert.False(t, found, "session should be removed from the registry after Quit")

	msgs := mb.GetAll()[rps.EnvEventTopic]
	require.Len(t, msgs, 3, "expected GameOpened, RoundResult(win), GameEnded(quit)")

	assert.Equal(t, rps.EventTypeGameEnded, decodeEventType(t, msgs[2]))
	ended := decodeGameEnded(t, msgs[2])
	assert.Equal(t, rps.ReasonQuit, ended.Body.Reason)
	assert.Nil(t, ended.Body.GrantedPrize, "Quit must never grant a payout")
}

// TestProcessor_Dispose_NoSessionIsNoop verifies Dispose is silent (no error,
// no event) when the character has no active session, per the brief.
func TestProcessor_Dispose_NoSessionIsNoop(t *testing.T) {
	setupRegistryTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)
	characterId := uint32(1005)

	p := game.NewProcessorWithLadder(testLogger(), ctx, fixedThrows(game.ThrowRock), ladderProviderFor(twoRungLadder()), noopSagaSubmitter())

	mb := message.NewBuffer()
	m, err := p.Dispose(mb, characterId)
	require.NoError(t, err)
	assert.Equal(t, game.Model{}, m)
	assert.Len(t, mb.GetAll()[rps.EnvEventTopic], 0, "Dispose on a missing session must not buffer any event")
}

// TestProcessor_Dispose_EndsActiveSessionAsDisconnected verifies Dispose ends
// an active session with reason "disconnected" and no payout.
func TestProcessor_Dispose_EndsActiveSessionAsDisconnected(t *testing.T) {
	setupRegistryTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)
	characterId := uint32(1006)

	p := game.NewProcessorWithLadder(testLogger(), ctx, fixedThrows(game.ThrowRock), ladderProviderFor(twoRungLadder()), noopSagaSubmitter())

	mb := message.NewBuffer()
	_, err := p.Start(mb, characterId, testWorldId, testChannelId, testNpcId)
	require.NoError(t, err)

	m, err := p.Dispose(mb, characterId)
	require.NoError(t, err)
	assert.Equal(t, game.StatusEnded, m.Status())

	_, found := game.GetRegistry().Get(ctx, characterId)
	assert.False(t, found)

	msgs := mb.GetAll()[rps.EnvEventTopic]
	require.Len(t, msgs, 2, "expected GameOpened, GameEnded(disconnected)")
	ended := decodeGameEnded(t, msgs[1])
	assert.Equal(t, rps.ReasonDisconnected, ended.Body.Reason)
	assert.Nil(t, ended.Body.GrantedPrize)
}

// TestProcessor_Select_NoSessionReturnsError verifies Select rejects a
// character with no active session.
func TestProcessor_Select_NoSessionReturnsError(t *testing.T) {
	setupRegistryTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)
	characterId := uint32(1007)

	p := game.NewProcessorWithLadder(testLogger(), ctx, fixedThrows(game.ThrowRock), ladderProviderFor(twoRungLadder()), noopSagaSubmitter())

	mb := message.NewBuffer()
	_, err := p.Select(mb, characterId, game.ThrowRock)
	assert.ErrorIs(t, err, game.ErrSessionNotFound)
}

// TestProcessor_Select_InvalidStatusReturnsError verifies Select rejects a
// session that is AWAITING_DECISION (a win pending Continue/Collect) rather
// than OPEN/AWAITING_SELECT.
func TestProcessor_Select_InvalidStatusReturnsError(t *testing.T) {
	setupRegistryTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)
	characterId := uint32(1008)

	p := game.NewProcessorWithLadder(testLogger(), ctx, fixedThrows(game.ThrowScissors), ladderProviderFor(twoRungLadder()), noopSagaSubmitter())

	mb := message.NewBuffer()
	_, err := p.Start(mb, characterId, testWorldId, testChannelId, testNpcId)
	require.NoError(t, err)

	// Rock beats Scissors: win leaves the session AWAITING_DECISION.
	m, err := p.Select(mb, characterId, game.ThrowRock)
	require.NoError(t, err)
	require.Equal(t, game.StatusAwaitingDecision, m.Status())

	_, err = p.Select(mb, characterId, game.ThrowRock)
	assert.ErrorIs(t, err, game.ErrInvalidStatus)
}

// TestProcessor_Continue_InvalidStatusReturnsError verifies Continue rejects
// a session that is not AWAITING_DECISION (e.g. freshly opened).
func TestProcessor_Continue_InvalidStatusReturnsError(t *testing.T) {
	setupRegistryTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)
	characterId := uint32(1009)

	p := game.NewProcessorWithLadder(testLogger(), ctx, fixedThrows(game.ThrowRock), ladderProviderFor(twoRungLadder()), noopSagaSubmitter())

	mb := message.NewBuffer()
	_, err := p.Start(mb, characterId, testWorldId, testChannelId, testNpcId)
	require.NoError(t, err)

	_, err = p.Continue(mb, characterId)
	assert.ErrorIs(t, err, game.ErrInvalidStatus)
}

// TestProcessor_Start_PropagatesLadderError verifies Start now resolves the
// ladder to source the GameOpened ante, and fails loudly (no session opened,
// no event buffered) rather than silently proceeding with a zero ante when
// the ladder provider errors - a ladder-load failure is a real error the
// entry saga must be able to compensate/refund against.
func TestProcessor_Start_PropagatesLadderError(t *testing.T) {
	setupRegistryTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)
	characterId := uint32(1099)

	ladderErr := errors.New("boom: config unavailable")
	p := game.NewProcessorWithLadder(testLogger(), ctx, fixedThrows(game.ThrowRock), erroringLadderProvider(ladderErr), noopSagaSubmitter())

	mb := message.NewBuffer()
	_, err := p.Start(mb, characterId, testWorldId, testChannelId, testNpcId)
	require.ErrorIs(t, err, ladderErr)

	_, found := game.GetRegistry().Get(ctx, characterId)
	assert.False(t, found, "no session should be opened when the ladder cannot be resolved")

	msgs := mb.GetAll()[rps.EnvEventTopic]
	assert.Empty(t, msgs, "no GameOpened event should be buffered when the ladder cannot be resolved")
}

// TestProcessor_StartAndEmit_PropagatesBuildError verifies the *AndEmit
// wrapper surfaces an error from the underlying buffered Method before ever
// reaching the kafka producer (standing up a broker is out of scope for this
// package's unit tests; the buffered Method(mb, ...) path is exercised
// directly in the tests above).
func TestProcessor_StartAndEmit_PropagatesBuildError(t *testing.T) {
	setupRegistryTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)

	p := game.NewProcessorWithLadder(testLogger(), ctx, fixedThrows(game.ThrowRock), ladderProviderFor(twoRungLadder()), noopSagaSubmitter())

	// characterId 0 fails ModelBuilder.Build()'s required-field validation.
	_, err := p.StartAndEmit(0, testWorldId, testChannelId, testNpcId)
	assert.Error(t, err)
}

// winToRungOne drives a fresh session through Start -> Select(win) so it
// lands on rung 1 in StatusAwaitingDecision, ready for a Collect test.
// Scissors always loses to the player's Rock throw.
func winToRungOne(t *testing.T, p game.Processor, mb *message.Buffer, characterId uint32) {
	t.Helper()
	_, err := p.Start(mb, characterId, testWorldId, testChannelId, testNpcId)
	require.NoError(t, err)
	m, err := p.Select(mb, characterId, game.ThrowRock)
	require.NoError(t, err)
	require.Equal(t, game.StatusAwaitingDecision, m.Status())
	require.Equal(t, 1, m.Rung())
}

// TestProcessor_Collect_SubmitsPayoutSaga_MesoAndItem verifies Collect builds
// and submits a two-step payout saga (AwardMesos then AwardAsset) when the
// resolved rung grants both a meso amount and an item.
func TestProcessor_Collect_SubmitsPayoutSaga_MesoAndItem(t *testing.T) {
	setupRegistryTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)
	characterId := uint32(2101)

	ladder := game.Ladder{
		EntryCostMeso: 1000,
		Rungs: []game.Rung{
			{Rung: 1, ItemId: item.Id(4001000), Quantity: 3, Meso: 250},
		},
	}

	var captured []sharedsaga.Saga
	p := game.NewProcessorWithLadder(testLogger(), ctx, fixedThrows(game.ThrowScissors), ladderProviderFor(ladder), capturingSagaSubmitter(&captured))

	mb := message.NewBuffer()
	winToRungOne(t, p, mb, characterId)

	m, err := p.Collect(mb, characterId)
	require.NoError(t, err)
	assert.Equal(t, game.StatusEnded, m.Status())

	require.Len(t, captured, 1, "expected exactly one payout saga submitted")
	s := captured[0]
	require.Len(t, s.Steps, 2, "expected AwardMesos + AwardAsset steps")

	assert.Equal(t, sharedsaga.AwardMesos, s.Steps[0].Action)
	mesoPayload, ok := s.Steps[0].Payload.(sharedsaga.AwardMesosPayload)
	require.True(t, ok, "expected AwardMesosPayload")
	assert.Equal(t, characterId, mesoPayload.CharacterId)
	assert.Equal(t, testWorldId, mesoPayload.WorldId)
	assert.Equal(t, testChannelId, mesoPayload.ChannelId)
	assert.Equal(t, int32(250), mesoPayload.Amount, "payout meso must be positive")

	assert.Equal(t, sharedsaga.AwardAsset, s.Steps[1].Action)
	itemPayload, ok := s.Steps[1].Payload.(sharedsaga.AwardItemActionPayload)
	require.True(t, ok, "expected AwardItemActionPayload")
	assert.Equal(t, characterId, itemPayload.CharacterId)
	assert.Equal(t, uint32(4001000), itemPayload.Item.TemplateId)
	assert.Equal(t, uint32(3), itemPayload.Item.Quantity)
}

// TestProcessor_Collect_SubmitsPayoutSaga_MesoOnly verifies Collect submits a
// single-step payout saga when the resolved rung grants only mesos (no
// item).
func TestProcessor_Collect_SubmitsPayoutSaga_MesoOnly(t *testing.T) {
	setupRegistryTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)
	characterId := uint32(2102)

	ladder := game.Ladder{
		EntryCostMeso: 1000,
		Rungs: []game.Rung{
			{Rung: 1, ItemId: 0, Quantity: 0, Meso: 300},
		},
	}

	var captured []sharedsaga.Saga
	p := game.NewProcessorWithLadder(testLogger(), ctx, fixedThrows(game.ThrowScissors), ladderProviderFor(ladder), capturingSagaSubmitter(&captured))

	mb := message.NewBuffer()
	winToRungOne(t, p, mb, characterId)

	_, err := p.Collect(mb, characterId)
	require.NoError(t, err)

	require.Len(t, captured, 1)
	s := captured[0]
	require.Len(t, s.Steps, 1, "expected only an AwardMesos step")
	assert.Equal(t, sharedsaga.AwardMesos, s.Steps[0].Action)
	mesoPayload, ok := s.Steps[0].Payload.(sharedsaga.AwardMesosPayload)
	require.True(t, ok)
	assert.Equal(t, int32(300), mesoPayload.Amount)
}

// TestProcessor_Collect_SubmitsPayoutSaga_ItemOnly verifies Collect submits a
// single-step payout saga when the resolved rung grants only an item (no
// meso).
func TestProcessor_Collect_SubmitsPayoutSaga_ItemOnly(t *testing.T) {
	setupRegistryTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)
	characterId := uint32(2103)

	ladder := game.Ladder{
		EntryCostMeso: 1000,
		Rungs: []game.Rung{
			{Rung: 1, ItemId: item.Id(4002000), Quantity: 5, Meso: 0},
		},
	}

	var captured []sharedsaga.Saga
	p := game.NewProcessorWithLadder(testLogger(), ctx, fixedThrows(game.ThrowScissors), ladderProviderFor(ladder), capturingSagaSubmitter(&captured))

	mb := message.NewBuffer()
	winToRungOne(t, p, mb, characterId)

	_, err := p.Collect(mb, characterId)
	require.NoError(t, err)

	require.Len(t, captured, 1)
	s := captured[0]
	require.Len(t, s.Steps, 1, "expected only an AwardAsset step")
	assert.Equal(t, sharedsaga.AwardAsset, s.Steps[0].Action)
	itemPayload, ok := s.Steps[0].Payload.(sharedsaga.AwardItemActionPayload)
	require.True(t, ok)
	assert.Equal(t, uint32(4002000), itemPayload.Item.TemplateId)
	assert.Equal(t, uint32(5), itemPayload.Item.Quantity)
}

// TestProcessor_Collect_NoPrizeSubmitsNoSaga verifies Collect submits no
// saga at all when the resolved rung has no configured prize.
func TestProcessor_Collect_NoPrizeSubmitsNoSaga(t *testing.T) {
	setupRegistryTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)
	characterId := uint32(2104)

	// A rung-5 prize with nothing configured at rung 1 means PrizeAt(1)
	// resolves ok=false, while MaxRung() is still 5 so Continue's
	// forced-collect-at-max branch isn't triggered.
	ladder := game.Ladder{
		EntryCostMeso: 1000,
		Rungs: []game.Rung{
			{Rung: 5, ItemId: item.Id(4003000), Quantity: 1, Meso: 100},
		},
	}

	var captured []sharedsaga.Saga
	p := game.NewProcessorWithLadder(testLogger(), ctx, fixedThrows(game.ThrowScissors), ladderProviderFor(ladder), capturingSagaSubmitter(&captured))

	mb := message.NewBuffer()
	winToRungOne(t, p, mb, characterId)

	_, err := p.Collect(mb, characterId)
	require.NoError(t, err)

	assert.Len(t, captured, 0, "no prize at this rung means no payout saga")
}

// TestProcessor_Collect_SubmitFailureIsRetrySafe is the money-safety
// regression: if the payout saga submission fails, Collect must (a) propagate
// the error to the caller, (b) leave the session in the registry (NOT
// removed), and (c) buffer NO GameEnded event. This proves a failed payout
// leaves the game collectible again on retry rather than silently consuming
// the session (and the prize) on a transient Kafka failure.
func TestProcessor_Collect_SubmitFailureIsRetrySafe(t *testing.T) {
	setupRegistryTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)
	characterId := uint32(2105)

	ladder := game.Ladder{
		EntryCostMeso: 1000,
		Rungs: []game.Rung{
			{Rung: 1, ItemId: item.Id(4004000), Quantity: 2, Meso: 400},
		},
	}

	submitErr := errors.New("saga submission failed")
	calls := 0
	p := game.NewProcessorWithLadder(testLogger(), ctx, fixedThrows(game.ThrowScissors), ladderProviderFor(ladder), erroringSagaSubmitter(submitErr, &calls))

	mb := message.NewBuffer()
	winToRungOne(t, p, mb, characterId)

	// Collect into a fresh buffer so any GameEnded it would buffer is
	// isolated from winToRungOne's GameOpened/RoundResult events.
	collectBuf := message.NewBuffer()
	_, err := p.Collect(collectBuf, characterId)

	// (a) the submission error is propagated to the caller.
	require.Error(t, err)
	assert.ErrorIs(t, err, submitErr)
	assert.Equal(t, 1, calls, "the saga submitter should have been invoked exactly once")

	// (b) the session is still present in the registry (NOT removed), and
	// remains collectible (still AWAITING_DECISION at its rung).
	m, found := game.GetRegistry().Get(ctx, characterId)
	require.True(t, found, "a failed payout must leave the session in the registry for retry")
	assert.Equal(t, game.StatusAwaitingDecision, m.Status())
	assert.Equal(t, 1, m.Rung())

	// (c) no GameEnded event was buffered.
	assert.Len(t, collectBuf.GetAll()[rps.EnvEventTopic], 0, "a failed payout must buffer no GameEnded event")
}

// TestProcessor_Collect_PrizePresentButGrantsNothing exercises the steps==0
// branch reached via prizeOk=true: the rung IS configured (PrizeAt returns
// ok=true) but grants neither meso nor an item (ItemId=0 && Meso=0). Collect
// must submit no saga, still remove the session, and still buffer a
// GameEnded{collected} carrying a zero/empty prize. This is distinct from the
// already-tested PrizeAt->ok=false path (TestProcessor_Collect_NoPrizeSubmitsNoSaga).
func TestProcessor_Collect_PrizePresentButGrantsNothing(t *testing.T) {
	setupRegistryTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)
	characterId := uint32(2106)

	// Rung 1 is explicitly present (so PrizeAt(1) returns ok=true) but grants
	// nothing.
	ladder := game.Ladder{
		EntryCostMeso: 1000,
		Rungs: []game.Rung{
			{Rung: 1, ItemId: 0, Quantity: 0, Meso: 0},
		},
	}

	calls := 0
	p := game.NewProcessorWithLadder(testLogger(), ctx, fixedThrows(game.ThrowScissors), ladderProviderFor(ladder), countingSagaSubmitter(&calls))

	mb := message.NewBuffer()
	winToRungOne(t, p, mb, characterId)

	collectBuf := message.NewBuffer()
	m, err := p.Collect(collectBuf, characterId)
	require.NoError(t, err)
	assert.Equal(t, game.StatusEnded, m.Status())

	// No saga submitted (steps==0 branch), reached via prizeOk=true.
	assert.Equal(t, 0, calls, "a present-but-empty prize must submit no payout saga")

	// The session is removed.
	_, found := game.GetRegistry().Get(ctx, characterId)
	assert.False(t, found, "the session should be removed after a zero-prize collect")

	// A GameEnded{collected} is still buffered, carrying an empty prize.
	msgs := collectBuf.GetAll()[rps.EnvEventTopic]
	require.Len(t, msgs, 1, "expected a single GameEnded event")
	ended := decodeGameEnded(t, msgs[0])
	assert.Equal(t, rps.ReasonCollected, ended.Body.Reason)
	if assert.NotNil(t, ended.Body.GrantedPrize, "collected always carries a granted prize (even if empty)") {
		assert.Equal(t, uint32(0), uint32(ended.Body.GrantedPrize.ItemId))
		assert.Equal(t, uint32(0), ended.Body.GrantedPrize.Quantity)
		assert.Equal(t, uint32(0), ended.Body.GrantedPrize.Meso)
	}
}

// TestProcessor_Collect_FromStatusOpenForfeits verifies that Collect, when
// invoked against a freshly opened session (rung 0, nothing played yet), is
// treated as the client's Exit action: the session is forfeited with no
// payout rather than erroring. The client has no dedicated "collect" sub-op
// - Exit(4) always maps to Collect - so this status must be handled rather
// than rejected with ErrInvalidStatus.
func TestProcessor_Collect_FromStatusOpenForfeits(t *testing.T) {
	setupRegistryTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)
	characterId := uint32(2107)

	calls := 0
	p := game.NewProcessorWithLadder(testLogger(), ctx, fixedThrows(game.ThrowRock), ladderProviderFor(twoRungLadder()), countingSagaSubmitter(&calls))

	mb := message.NewBuffer()
	m, err := p.Start(mb, characterId, testWorldId, testChannelId, testNpcId)
	require.NoError(t, err)
	require.Equal(t, game.StatusOpen, m.Status())
	require.Equal(t, 0, m.Rung())

	collectBuf := message.NewBuffer()
	m, err = p.Collect(collectBuf, characterId)
	require.NoError(t, err, "Collect must not error from StatusOpen - it is the client's only leave action")
	assert.Equal(t, game.StatusEnded, m.Status())

	assert.Equal(t, 0, calls, "an unplayed rung must never submit a payout saga")

	_, found := game.GetRegistry().Get(ctx, characterId)
	assert.False(t, found, "the session must be removed on forfeit")

	msgs := collectBuf.GetAll()[rps.EnvEventTopic]
	require.Len(t, msgs, 1, "expected a single GameEnded event")
	ended := decodeGameEnded(t, msgs[0])
	assert.Equal(t, rps.ReasonQuit, ended.Body.Reason)
	assert.Nil(t, ended.Body.GrantedPrize, "a forfeited collect grants no prize")
}

// TestProcessor_Collect_FromStatusAwaitingSelectForfeits verifies that
// Collect, when invoked mid-round (after a Continue or a tie, before the
// next Select is resolved), forfeits the ALREADY-WON rung N rather than
// paying it out - the rung is being risked and is not yet banked, so exiting
// mid-risk must not pay.
func TestProcessor_Collect_FromStatusAwaitingSelectForfeits(t *testing.T) {
	setupRegistryTest(t)
	ten := setupTestTenant(t)
	ctx := testCtx(ten)
	characterId := uint32(2108)

	calls := 0
	p := game.NewProcessorWithLadder(testLogger(), ctx, fixedThrows(game.ThrowScissors), ladderProviderFor(twoRungLadder()), countingSagaSubmitter(&calls))

	mb := message.NewBuffer()
	winToRungOne(t, p, mb, characterId)

	m, err := p.Continue(mb, characterId)
	require.NoError(t, err)
	require.Equal(t, game.StatusAwaitingSelect, m.Status())
	require.Equal(t, 1, m.Rung(), "rung 1 was won but is now being risked, not yet collected")

	collectBuf := message.NewBuffer()
	m, err = p.Collect(collectBuf, characterId)
	require.NoError(t, err, "Collect must not error from StatusAwaitingSelect - it is the client's only leave action")
	assert.Equal(t, game.StatusEnded, m.Status())

	assert.Equal(t, 0, calls, "the mid-risk rung must not be paid out on forfeit")

	_, found := game.GetRegistry().Get(ctx, characterId)
	assert.False(t, found, "the session must be removed on forfeit")

	msgs := collectBuf.GetAll()[rps.EnvEventTopic]
	require.Len(t, msgs, 1, "expected a single GameEnded event")
	ended := decodeGameEnded(t, msgs[0])
	assert.Equal(t, rps.ReasonQuit, ended.Body.Reason)
	assert.Nil(t, ended.Body.GrantedPrize, "a forfeited collect grants no prize")
}
