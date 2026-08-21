package playernpc

import (
	msg "atlas-player-npcs/kafka/message/playernpc"
	"atlas-player-npcs/playernpc"
	"context"
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
	return playernpc.Model{}, nil
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
		handleDeploy(providerFor(s))(l, context.Background(), c)

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
		handleDeploy(providerFor(s))(l, context.Background(), c)

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
		handleDeploy(providerFor(s))(l, context.Background(), c)
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
		handleRedeploy(providerFor(s))(l, context.Background(), c)

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
		handleRedeploy(providerFor(s))(l, context.Background(), c)

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
		handleRemove(providerFor(s))(l, context.Background(), c)

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
		handleRemove(providerFor(s))(l, context.Background(), c)

		if len(s.removeCalls) != 1 {
			t.Fatalf("Remove call count = %d, want 1", len(s.removeCalls))
		}
		if s.removeCalls[0].mapId == nil || *s.removeCalls[0].mapId != mapId {
			t.Errorf("Remove called with mapId = %v, want %v", s.removeCalls[0].mapId, mapId)
		}
	})
}
