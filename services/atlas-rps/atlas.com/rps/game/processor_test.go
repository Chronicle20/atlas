package game_test

import (
	"encoding/json"
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
