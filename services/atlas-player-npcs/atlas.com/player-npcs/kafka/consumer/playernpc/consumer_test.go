package playernpc

import (
	msg "atlas-player-npcs/kafka/message/playernpc"
	"atlas-player-npcs/playernpc"
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

// stubProcessor records every call this task's three command handlers can
// make, and returns whatever the test case pre-loaded. It never touches a
// database or an HTTP client -- the Builder pattern (playernpc.Model has
// no exported constructor besides Builder) is how it manufactures the
// fixtures GetByMap returns.
type stubProcessor struct {
	deployCalls   []deployCall
	redeployCalls []uuid.UUID
	removeCalls   []removeCall

	getByMapResult []playernpc.Model
	getByMapErr    error
	deployErr      error
	redeployErr    error
	removeErr      error
}

type deployCall struct {
	characterId        uint32
	worldId            world.Id
	mapId              _map.Id
	enforceEligibility bool
	explicit           *playernpc.Position
}

type removeCall struct {
	characterId uint32
	mapId       *_map.Id
}

func (s *stubProcessor) Deploy(characterId uint32, worldId world.Id, mapId _map.Id, enforceEligibility bool, explicit *playernpc.Position) (playernpc.Model, error) {
	s.deployCalls = append(s.deployCalls, deployCall{characterId, worldId, mapId, enforceEligibility, explicit})
	return playernpc.Model{}, s.deployErr
}

func (s *stubProcessor) Redeploy(id uuid.UUID) (playernpc.Model, error) {
	s.redeployCalls = append(s.redeployCalls, id)
	return playernpc.Model{}, s.redeployErr
}

func (s *stubProcessor) RemoveById(id uuid.UUID) (playernpc.Model, error) {
	return playernpc.Model{}, nil
}

func (s *stubProcessor) Remove(characterId uint32, mapId *_map.Id) ([]playernpc.Model, error) {
	s.removeCalls = append(s.removeCalls, removeCall{characterId, mapId})
	return nil, s.removeErr
}

func (s *stubProcessor) GetById(id uuid.UUID) (playernpc.Model, error) {
	return playernpc.Model{}, nil
}

func (s *stubProcessor) GetByMap(worldId world.Id, mapId _map.Id, page model.Page) ([]playernpc.Model, error) {
	return s.getByMapResult, s.getByMapErr
}

var _ playernpc.Processor = (*stubProcessor)(nil)

func providerFor(s *stubProcessor) ProcessorProvider {
	return func(logrus.FieldLogger, context.Context) playernpc.Processor { return s }
}

func testLogger() (logrus.FieldLogger, *logtest.Hook) {
	l, hook := logtest.NewNullLogger()
	return l, hook
}

// captureEmitter returns an OutcomeEmitter that records every emitted
// event, and a no-op discard emitter for tests that don't assert on
// emissions.
func captureEmitter() (*[]msg.StatusEvent[msg.StatusCommandOutcomeBody], OutcomeEmitter) {
	events := make([]msg.StatusEvent[msg.StatusCommandOutcomeBody], 0)
	return &events, func(e msg.StatusEvent[msg.StatusCommandOutcomeBody]) {
		events = append(events, e)
	}
}

func discardEmitter() OutcomeEmitter {
	return func(msg.StatusEvent[msg.StatusCommandOutcomeBody]) {}
}

// errBoom is an unclassified error (CodeFor's default arm) used to test
// that a GetByMap resolve failure surfaces as CodeInternal.
var errBoom = errors.New("boom")

func buildNpc(t *testing.T, id uuid.UUID, characterId uint32) playernpc.Model {
	t.Helper()
	m, err := playernpc.NewBuilder().
		SetId(id).
		SetCharacterId(characterId).
		SetName("Statue").
		SetWorldId(0).
		SetMapId(102000004).
		SetScriptId(1).
		SetObjectId(1).
		SetJobId(100).
		Build()
	if err != nil {
		t.Fatalf("Build() unexpected err = %v", err)
	}
	return m
}

func TestPlayerNpcCommandConsumer(t *testing.T) {
	t.Run("deploy", func(t *testing.T) {
		s := &stubProcessor{}
		l, _ := testLogger()
		c := msg.Command[msg.CommandDeployBody]{
			CharacterId: 42,
			Type:        msg.CommandTypeDeploy,
			Body: msg.CommandDeployBody{
				WorldId:            world.Id(1),
				MapId:              _map.Id(102000004),
				EnforceEligibility: true,
			},
		}
		handleDeploy(providerFor(s), discardEmitter())(l, context.Background(), c)

		if len(s.deployCalls) != 1 {
			t.Fatalf("Deploy call count = %d, want 1", len(s.deployCalls))
		}
		got := s.deployCalls[0]
		if got.characterId != 42 || got.worldId != world.Id(1) || got.mapId != _map.Id(102000004) || !got.enforceEligibility {
			t.Errorf("Deploy called with %+v", got)
		}
		if got.explicit != nil {
			t.Errorf("Deploy explicit position = %+v, want nil", got.explicit)
		}
	})

	t.Run("deploy with position", func(t *testing.T) {
		s := &stubProcessor{}
		l, _ := testLogger()
		c := msg.Command[msg.CommandDeployBody]{
			CharacterId: 42,
			Type:        msg.CommandTypeDeploy,
			Body: msg.CommandDeployBody{
				WorldId:  world.Id(1),
				MapId:    _map.Id(102000004),
				Position: &msg.CommandPosition{X: 100, Y: 200},
			},
		}
		handleDeploy(providerFor(s), discardEmitter())(l, context.Background(), c)

		if len(s.deployCalls) != 1 {
			t.Fatalf("Deploy call count = %d, want 1", len(s.deployCalls))
		}
		got := s.deployCalls[0].explicit
		if got == nil || got.X != 100 || got.Y != 200 {
			t.Errorf("Deploy explicit position = %+v, want {100 200}", got)
		}
	})

	t.Run("wrong type does nothing", func(t *testing.T) {
		s := &stubProcessor{}
		l, _ := testLogger()
		c := msg.Command[msg.CommandDeployBody]{Type: msg.CommandTypeRedeploy}
		handleDeploy(providerFor(s), discardEmitter())(l, context.Background(), c)
		if len(s.deployCalls) != 0 {
			t.Errorf("Deploy call count = %d, want 0", len(s.deployCalls))
		}
	})

	t.Run("redeploy", func(t *testing.T) {
		id := uuid.New()
		s := &stubProcessor{getByMapResult: []playernpc.Model{buildNpc(t, id, 42)}}
		l, _ := testLogger()
		c := msg.Command[msg.CommandRedeployBody]{
			CharacterId: 42,
			Type:        msg.CommandTypeRedeploy,
			Body:        msg.CommandRedeployBody{WorldId: world.Id(1), MapId: _map.Id(102000004)},
		}
		handleRedeploy(providerFor(s), discardEmitter())(l, context.Background(), c)

		if len(s.redeployCalls) != 1 || s.redeployCalls[0] != id {
			t.Errorf("Redeploy calls = %+v, want [%s]", s.redeployCalls, id)
		}
	})

	t.Run("redeploy not found logs and does not call Redeploy", func(t *testing.T) {
		s := &stubProcessor{getByMapResult: nil}
		l, hook := testLogger()
		c := msg.Command[msg.CommandRedeployBody]{
			CharacterId: 42,
			Type:        msg.CommandTypeRedeploy,
			Body:        msg.CommandRedeployBody{WorldId: world.Id(1), MapId: _map.Id(102000004)},
		}
		handleRedeploy(providerFor(s), discardEmitter())(l, context.Background(), c)

		if len(s.redeployCalls) != 0 {
			t.Errorf("Redeploy calls = %+v, want none", s.redeployCalls)
		}
		if len(hook.Entries) == 0 {
			t.Errorf("expected a warn log entry when no Player NPC is found")
		}
	})

	t.Run("remove", func(t *testing.T) {
		s := &stubProcessor{}
		l, _ := testLogger()
		c := msg.Command[msg.CommandRemoveBody]{
			CharacterId: 42,
			Type:        msg.CommandTypeRemove,
			Body:        msg.CommandRemoveBody{},
		}
		handleRemove(providerFor(s), discardEmitter())(l, context.Background(), c)

		if len(s.removeCalls) != 1 {
			t.Fatalf("Remove call count = %d, want 1", len(s.removeCalls))
		}
		if s.removeCalls[0].characterId != 42 || s.removeCalls[0].mapId != nil {
			t.Errorf("Remove called with %+v, want characterId=42, mapId=nil", s.removeCalls[0])
		}
	})

	t.Run("remove map-scoped", func(t *testing.T) {
		s := &stubProcessor{}
		l, _ := testLogger()
		mapId := _map.Id(102000004)
		c := msg.Command[msg.CommandRemoveBody]{
			CharacterId: 42,
			Type:        msg.CommandTypeRemove,
			Body:        msg.CommandRemoveBody{MapId: &mapId},
		}
		handleRemove(providerFor(s), discardEmitter())(l, context.Background(), c)

		if len(s.removeCalls) != 1 {
			t.Fatalf("Remove call count = %d, want 1", len(s.removeCalls))
		}
		if s.removeCalls[0].mapId == nil || *s.removeCalls[0].mapId != mapId {
			t.Errorf("Remove called with mapId = %v, want %v", s.removeCalls[0].mapId, mapId)
		}
	})
}

func TestPlayerNpcCommandConsumerOutcomeEmission(t *testing.T) {
	txnId := uuid.New()
	requester := &msg.Requester{CharacterId: 7, WorldId: 1, ChannelId: 2, MapId: 999}

	t.Run("deploy success emits COMMAND_SUCCEEDED with empty code", func(t *testing.T) {
		s := &stubProcessor{}
		l, _ := testLogger()
		events, oe := captureEmitter()
		c := msg.Command[msg.CommandDeployBody]{
			CharacterId:   42,
			TransactionId: txnId,
			Type:          msg.CommandTypeDeploy,
			Body: msg.CommandDeployBody{
				WorldId: world.Id(1),
				MapId:   _map.Id(102000004),
			},
		}
		handleDeploy(providerFor(s), oe)(l, context.Background(), c)

		if len(*events) != 1 {
			t.Fatalf("emitted event count = %d, want 1", len(*events))
		}
		got := (*events)[0]
		if got.Type != msg.EventTypeCommandSucceeded {
			t.Errorf("Type = %q, want %q", got.Type, msg.EventTypeCommandSucceeded)
		}
		if got.Body.CommandType != msg.CommandTypeDeploy || got.Body.CharacterId != 42 || got.Body.Code != "" {
			t.Errorf("Body = %+v", got.Body)
		}
	})

	t.Run("deploy failure emits COMMAND_FAILED with the classified code", func(t *testing.T) {
		s := &stubProcessor{deployErr: playernpc.ErrPoolExhausted}
		l, _ := testLogger()
		events, oe := captureEmitter()
		c := msg.Command[msg.CommandDeployBody]{
			CharacterId:   42,
			TransactionId: txnId,
			Type:          msg.CommandTypeDeploy,
			Body: msg.CommandDeployBody{
				WorldId: world.Id(1),
				MapId:   _map.Id(102000004),
			},
		}
		handleDeploy(providerFor(s), oe)(l, context.Background(), c)

		if len(*events) != 1 {
			t.Fatalf("emitted event count = %d, want 1", len(*events))
		}
		got := (*events)[0]
		if got.Type != msg.EventTypeCommandFailed {
			t.Errorf("Type = %q, want %q", got.Type, msg.EventTypeCommandFailed)
		}
		if got.Body.Code != playernpc.CodePoolExhausted {
			t.Errorf("Code = %q, want %q", got.Body.Code, playernpc.CodePoolExhausted)
		}
		if got.Body.Message != playernpc.ErrPoolExhausted.Error() {
			t.Errorf("Message = %q, want %q", got.Body.Message, playernpc.ErrPoolExhausted.Error())
		}
	})

	t.Run("redeploy resolve failure emits COMMAND_FAILED", func(t *testing.T) {
		s := &stubProcessor{getByMapErr: errBoom}
		l, _ := testLogger()
		events, oe := captureEmitter()
		c := msg.Command[msg.CommandRedeployBody]{
			CharacterId:   42,
			TransactionId: txnId,
			Type:          msg.CommandTypeRedeploy,
			Body:          msg.CommandRedeployBody{WorldId: world.Id(1), MapId: _map.Id(102000004)},
		}
		handleRedeploy(providerFor(s), oe)(l, context.Background(), c)

		if len(*events) != 1 {
			t.Fatalf("emitted event count = %d, want 1", len(*events))
		}
		got := (*events)[0]
		if got.Type != msg.EventTypeCommandFailed || got.Body.Code != playernpc.CodeInternal {
			t.Errorf("Body = %+v", got.Body)
		}
	})

	t.Run("redeploy not found emits COMMAND_FAILED with CodeUnresolvable", func(t *testing.T) {
		s := &stubProcessor{getByMapResult: nil}
		l, _ := testLogger()
		events, oe := captureEmitter()
		c := msg.Command[msg.CommandRedeployBody]{
			CharacterId:   42,
			TransactionId: txnId,
			Type:          msg.CommandTypeRedeploy,
			Body:          msg.CommandRedeployBody{WorldId: world.Id(1), MapId: _map.Id(102000004)},
		}
		handleRedeploy(providerFor(s), oe)(l, context.Background(), c)

		if len(*events) != 1 {
			t.Fatalf("emitted event count = %d, want 1", len(*events))
		}
		got := (*events)[0]
		if got.Type != msg.EventTypeCommandFailed {
			t.Errorf("Type = %q, want %q", got.Type, msg.EventTypeCommandFailed)
		}
		if got.Body.Code != playernpc.CodeUnresolvable {
			t.Errorf("Code = %q, want %q", got.Body.Code, playernpc.CodeUnresolvable)
		}
		if got.Body.Message == "" {
			t.Errorf("Message = %q, want a message naming the character and map", got.Body.Message)
		}
	})

	t.Run("remove success emits COMMAND_SUCCEEDED", func(t *testing.T) {
		s := &stubProcessor{}
		l, _ := testLogger()
		events, oe := captureEmitter()
		c := msg.Command[msg.CommandRemoveBody]{
			CharacterId:   42,
			TransactionId: txnId,
			Type:          msg.CommandTypeRemove,
			Body:          msg.CommandRemoveBody{},
		}
		handleRemove(providerFor(s), oe)(l, context.Background(), c)

		if len(*events) != 1 {
			t.Fatalf("emitted event count = %d, want 1", len(*events))
		}
		got := (*events)[0]
		if got.Type != msg.EventTypeCommandSucceeded || got.Body.CommandType != msg.CommandTypeRemove {
			t.Errorf("Body = %+v", got.Body)
		}
	})

	t.Run("Requester and TransactionId round-trip onto the outcome event", func(t *testing.T) {
		s := &stubProcessor{}
		l, _ := testLogger()
		events, oe := captureEmitter()
		c := msg.Command[msg.CommandRemoveBody]{
			CharacterId:   42,
			TransactionId: txnId,
			Type:          msg.CommandTypeRemove,
			Requester:     requester,
			Body:          msg.CommandRemoveBody{},
		}
		handleRemove(providerFor(s), oe)(l, context.Background(), c)

		if len(*events) != 1 {
			t.Fatalf("emitted event count = %d, want 1", len(*events))
		}
		got := (*events)[0].Body
		if got.TransactionId != txnId {
			t.Errorf("TransactionId = %s, want %s", got.TransactionId, txnId)
		}
		if got.Requester == nil || *got.Requester != *requester {
			t.Errorf("Requester = %+v, want %+v", got.Requester, requester)
		}
	})

	t.Run("nobody listening -> no emit", func(t *testing.T) {
		s := &stubProcessor{}
		l, _ := testLogger()
		events, oe := captureEmitter()
		c := msg.Command[msg.CommandRemoveBody]{
			CharacterId: 42,
			Type:        msg.CommandTypeRemove,
			Body:        msg.CommandRemoveBody{},
		}
		handleRemove(providerFor(s), oe)(l, context.Background(), c)

		if len(*events) != 0 {
			t.Errorf("emitted event count = %d, want 0 when TransactionId is uuid.Nil and Requester is nil", len(*events))
		}
	})
}
