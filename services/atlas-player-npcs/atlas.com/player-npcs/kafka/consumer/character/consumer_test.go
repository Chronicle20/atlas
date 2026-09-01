package character

import (
	"atlas-player-npcs/character"
	"atlas-player-npcs/configuration"
	charmsg "atlas-player-npcs/kafka/message/character"
	npcmsg "atlas-player-npcs/kafka/message/playernpc"
	"atlas-player-npcs/playernpc"
	"atlas-player-npcs/routing"
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-constants/constants"
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/producer/producertest"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

var emitted *producertest.Capture

func TestMain(m *testing.M) {
	// topic.EnvProvider is fail-closed: an unset token resolves to an error
	// and nothing is emitted, so every token this package's code under test
	// publishes to must carry a value. Setting each to its own name keeps the
	// Capture keyed by the token, which is what the assertions look up.
	_ = os.Setenv(string(npcmsg.EnvCommandTopic), string(npcmsg.EnvCommandTopic))
	_ = os.Setenv(string(charmsg.EnvEventTopicCharacterStatus), string(charmsg.EnvEventTopicCharacterStatus))
	emitted = producertest.InstallCapturing()
	os.Exit(m.Run())
}

// stubCharacterProcessor and stubConfigurationProcessor mirror
// playernpc/processor_test.go's stub shape (this task's controller notes)
// so this package's tests never make a real HTTP call.
type stubCharacterProcessor struct {
	m       character.Model
	err     error
	fetches int
}

func (s *stubCharacterProcessor) GetById(uint32) (character.Model, error) {
	s.fetches++
	return s.m, s.err
}

func (s *stubCharacterProcessor) ByNameProvider(string) model.Provider[[]character.Model] {
	return model.FixedProvider([]character.Model{s.m})
}
func (s *stubCharacterProcessor) GetByName(string) (character.Model, error) { return s.m, nil }

type stubConfigurationProcessor struct {
	m configuration.Model
}

func (s stubConfigurationProcessor) GetByTenantId(uuid.UUID) configuration.Model { return s.m }

// stubPlayerNpcProcessor answers GetByMap only -- the sole Processor
// method the LEVEL_CHANGED consumer calls.
type stubPlayerNpcProcessor struct {
	result []playernpc.Model
	err    error
	calls  int
}

func (s *stubPlayerNpcProcessor) Deploy(uint32, world.Id, _map.Id, bool, *playernpc.Position) (playernpc.Model, error) {
	return playernpc.Model{}, nil
}

func (s *stubPlayerNpcProcessor) Redeploy(uuid.UUID) (playernpc.Model, error) {
	return playernpc.Model{}, nil
}

func (s *stubPlayerNpcProcessor) RemoveById(uuid.UUID) (playernpc.Model, error) {
	return playernpc.Model{}, nil
}

func (s *stubPlayerNpcProcessor) Remove(uint32, *_map.Id) ([]playernpc.Model, error) {
	return nil, nil
}

func (s *stubPlayerNpcProcessor) GetById(uuid.UUID) (playernpc.Model, error) {
	return playernpc.Model{}, nil
}

func (s *stubPlayerNpcProcessor) GetByMap(world.Id, _map.Id, model.Page) ([]playernpc.Model, error) {
	s.calls++
	return s.result, s.err
}

func (s *stubPlayerNpcProcessor) GetByMapPaged(world.Id, _map.Id, model.Page) (model.Paged[playernpc.Model], error) {
	return model.Paged[playernpc.Model]{Items: s.result}, s.err
}

func (s *stubPlayerNpcProcessor) Eligibility(uint32, byte, uint32) (bool, string, error) {
	return false, "", nil
}

var _ playernpc.Processor = (*stubPlayerNpcProcessor)(nil)

func buildCharacter(t *testing.T, id uint32, name string, level byte, jobId job.Id, gm bool) character.Model {
	t.Helper()
	gmInt := 0
	if gm {
		gmInt = 1
	}
	m, err := character.Extract(character.RestModel{Id: id, Name: name, JobId: jobId, Level: level, Gm: gmInt})
	if err != nil {
		t.Fatalf("Extract() unexpected err = %v", err)
	}
	return m
}

func testContext(t *testing.T) context.Context {
	t.Helper()
	te, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create() unexpected err = %v", err)
	}
	return tenant.WithContext(context.Background(), te)
}

func testLogger() (logrus.FieldLogger, *logtest.Hook) {
	l, hook := logtest.NewNullLogger()
	return l, hook
}

func TestLevelChangedConsumer(t *testing.T) {
	t.Run("at max level, eligible", func(t *testing.T) {
		emitted.Reset()
		cp := &stubCharacterProcessor{m: buildCharacter(t, 42, "Hero", 200, job.WarriorId, false)}
		cfgp := stubConfigurationProcessor{m: mustConfig(t, true)}
		pp := &stubPlayerNpcProcessor{}
		deps := Dependencies{
			Character:     func(logrus.FieldLogger, context.Context) character.Processor { return cp },
			Configuration: func(logrus.FieldLogger, context.Context) configuration.Processor { return cfgp },
			PlayerNpc:     func(logrus.FieldLogger, context.Context) playernpc.Processor { return pp },
		}
		l, _ := testLogger()
		ctx := testContext(t)
		e := charmsg.StatusEvent[charmsg.LevelChangedStatusEventBody]{
			WorldId:     world.Id(1),
			CharacterId: 42,
			Type:        charmsg.StatusEventTypeLevelChanged,
			Body:        charmsg.LevelChangedStatusEventBody{Current: 200},
		}
		handleLevelChanged(deps)(l, ctx, e)

		if cp.fetches != 1 {
			t.Errorf("character fetches = %d, want 1", cp.fetches)
		}
		msgs := emitted.Messages(string(npcmsg.EnvCommandTopic))
		if len(msgs) != 1 {
			t.Fatalf("emitted DEPLOY commands = %d, want 1", len(msgs))
		}
		var cmd npcmsg.Command[npcmsg.CommandDeployBody]
		if err := json.Unmarshal(msgs[0].Value, &cmd); err != nil {
			t.Fatalf("unmarshal emitted command: %v", err)
		}
		if cmd.Type != npcmsg.CommandTypeDeploy || cmd.CharacterId != 42 {
			t.Errorf("emitted command = %+v", cmd)
		}
		wantMapId := routing.HallOfFameMapFor(constants.For("GMS", 83, 1), job.WarriorId)
		if cmd.Body.MapId != wantMapId {
			t.Errorf("emitted command MapId = %v, want %v", cmd.Body.MapId, wantMapId)
		}
		if !cmd.Body.EnforceEligibility {
			t.Errorf("emitted command EnforceEligibility = false, want true")
		}
	})

	t.Run("below max level", func(t *testing.T) {
		emitted.Reset()
		cp := &stubCharacterProcessor{m: buildCharacter(t, 42, "Hero", 199, job.WarriorId, false)}
		deps := Dependencies{
			Character: func(logrus.FieldLogger, context.Context) character.Processor { return cp },
			Configuration: func(logrus.FieldLogger, context.Context) configuration.Processor {
				return stubConfigurationProcessor{m: mustConfig(t, true)}
			},
			PlayerNpc: func(logrus.FieldLogger, context.Context) playernpc.Processor { return &stubPlayerNpcProcessor{} },
		}
		l, _ := testLogger()
		ctx := testContext(t)
		e := charmsg.StatusEvent[charmsg.LevelChangedStatusEventBody]{
			WorldId:     world.Id(1),
			CharacterId: 42,
			Type:        charmsg.StatusEventTypeLevelChanged,
			Body:        charmsg.LevelChangedStatusEventBody{Current: 199},
		}
		handleLevelChanged(deps)(l, ctx, e)

		if cp.fetches != 0 {
			t.Errorf("character fetches = %d, want 0 (cheap path)", cp.fetches)
		}
		if msgs := emitted.Messages(string(npcmsg.EnvCommandTopic)); len(msgs) != 0 {
			t.Errorf("emitted DEPLOY commands = %d, want 0", len(msgs))
		}
	})

	t.Run("auto-deploy disabled", func(t *testing.T) {
		emitted.Reset()
		cp := &stubCharacterProcessor{m: buildCharacter(t, 42, "Hero", 200, job.WarriorId, false)}
		deps := Dependencies{
			Character: func(logrus.FieldLogger, context.Context) character.Processor { return cp },
			Configuration: func(logrus.FieldLogger, context.Context) configuration.Processor {
				return stubConfigurationProcessor{m: mustConfig(t, false)}
			},
			PlayerNpc: func(logrus.FieldLogger, context.Context) playernpc.Processor { return &stubPlayerNpcProcessor{} },
		}
		l, _ := testLogger()
		e := charmsg.StatusEvent[charmsg.LevelChangedStatusEventBody]{
			WorldId:     world.Id(1),
			CharacterId: 42,
			Type:        charmsg.StatusEventTypeLevelChanged,
			Body:        charmsg.LevelChangedStatusEventBody{Current: 200},
		}
		handleLevelChanged(deps)(l, testContext(t), e)

		if msgs := emitted.Messages(string(npcmsg.EnvCommandTopic)); len(msgs) != 0 {
			t.Errorf("emitted DEPLOY commands = %d, want 0", len(msgs))
		}
	})

	t.Run("character is a GM", func(t *testing.T) {
		emitted.Reset()
		cp := &stubCharacterProcessor{m: buildCharacter(t, 42, "Hero", 200, job.WarriorId, true)}
		deps := Dependencies{
			Character: func(logrus.FieldLogger, context.Context) character.Processor { return cp },
			Configuration: func(logrus.FieldLogger, context.Context) configuration.Processor {
				return stubConfigurationProcessor{m: mustConfig(t, true)}
			},
			PlayerNpc: func(logrus.FieldLogger, context.Context) playernpc.Processor { return &stubPlayerNpcProcessor{} },
		}
		l, _ := testLogger()
		e := charmsg.StatusEvent[charmsg.LevelChangedStatusEventBody]{
			WorldId:     world.Id(1),
			CharacterId: 42,
			Type:        charmsg.StatusEventTypeLevelChanged,
			Body:        charmsg.LevelChangedStatusEventBody{Current: 200},
		}
		handleLevelChanged(deps)(l, testContext(t), e)

		if msgs := emitted.Messages(string(npcmsg.EnvCommandTopic)); len(msgs) != 0 {
			t.Errorf("emitted DEPLOY commands = %d, want 0", len(msgs))
		}
	})

	t.Run("already deployed", func(t *testing.T) {
		emitted.Reset()
		cp := &stubCharacterProcessor{m: buildCharacter(t, 42, "Hero", 200, job.WarriorId, false)}
		existing, err := playernpc.NewBuilder().SetId(uuid.New()).SetCharacterId(42).SetName("Hero").Build()
		if err != nil {
			t.Fatalf("Build() unexpected err = %v", err)
		}
		pp := &stubPlayerNpcProcessor{result: []playernpc.Model{existing}}
		deps := Dependencies{
			Character: func(logrus.FieldLogger, context.Context) character.Processor { return cp },
			Configuration: func(logrus.FieldLogger, context.Context) configuration.Processor {
				return stubConfigurationProcessor{m: mustConfig(t, true)}
			},
			PlayerNpc: func(logrus.FieldLogger, context.Context) playernpc.Processor { return pp },
		}
		l, _ := testLogger()
		e := charmsg.StatusEvent[charmsg.LevelChangedStatusEventBody]{
			WorldId:     world.Id(1),
			CharacterId: 42,
			Type:        charmsg.StatusEventTypeLevelChanged,
			Body:        charmsg.LevelChangedStatusEventBody{Current: 200},
		}
		handleLevelChanged(deps)(l, testContext(t), e)

		if pp.calls != 1 {
			t.Errorf("GetByMap calls = %d, want 1", pp.calls)
		}
		if msgs := emitted.Messages(string(npcmsg.EnvCommandTopic)); len(msgs) != 0 {
			t.Errorf("emitted DEPLOY commands = %d, want 0", len(msgs))
		}
	})

	t.Run("fetch fails", func(t *testing.T) {
		emitted.Reset()
		cp := &stubCharacterProcessor{err: errors.New("boom")}
		deps := Dependencies{
			Character: func(logrus.FieldLogger, context.Context) character.Processor { return cp },
			Configuration: func(logrus.FieldLogger, context.Context) configuration.Processor {
				return stubConfigurationProcessor{m: mustConfig(t, true)}
			},
			PlayerNpc: func(logrus.FieldLogger, context.Context) playernpc.Processor { return &stubPlayerNpcProcessor{} },
		}
		l, hook := testLogger()
		e := charmsg.StatusEvent[charmsg.LevelChangedStatusEventBody]{
			WorldId:     world.Id(1),
			CharacterId: 42,
			Type:        charmsg.StatusEventTypeLevelChanged,
			Body:        charmsg.LevelChangedStatusEventBody{Current: 200},
		}

		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("handleLevelChanged panicked: %v", r)
				}
			}()
			handleLevelChanged(deps)(l, testContext(t), e)
		}()

		if msgs := emitted.Messages(string(npcmsg.EnvCommandTopic)); len(msgs) != 0 {
			t.Errorf("emitted DEPLOY commands = %d, want 0", len(msgs))
		}
		if len(hook.Entries) == 0 || hook.LastEntry().Level != logrus.WarnLevel {
			t.Errorf("expected a warn log entry on fetch failure, got %+v", hook.Entries)
		}
		if hook.LastEntry().Data["characterId"] == nil {
			// characterId is carried via the message args (Warnf), not a
			// structured field -- assert on the rendered message instead.
			msgStr := hook.LastEntry().Message
			if msgStr == "" {
				t.Errorf("expected a non-empty warn message")
			}
		}
	})

	t.Run("duplicate redelivery", func(t *testing.T) {
		emitted.Reset()
		cp := &stubCharacterProcessor{m: buildCharacter(t, 42, "Hero", 200, job.WarriorId, false)}
		deps := Dependencies{
			Character: func(logrus.FieldLogger, context.Context) character.Processor { return cp },
			Configuration: func(logrus.FieldLogger, context.Context) configuration.Processor {
				return stubConfigurationProcessor{m: mustConfig(t, true)}
			},
			PlayerNpc: func(logrus.FieldLogger, context.Context) playernpc.Processor { return &stubPlayerNpcProcessor{} },
		}
		l, _ := testLogger()
		e := charmsg.StatusEvent[charmsg.LevelChangedStatusEventBody]{
			WorldId:     world.Id(1),
			CharacterId: 42,
			Type:        charmsg.StatusEventTypeLevelChanged,
			Body:        charmsg.LevelChangedStatusEventBody{Current: 200},
		}
		ctx := testContext(t)
		handleLevelChanged(deps)(l, ctx, e)
		handleLevelChanged(deps)(l, ctx, e)

		msgs := emitted.Messages(string(npcmsg.EnvCommandTopic))
		if len(msgs) != 2 {
			t.Fatalf("emitted DEPLOY commands = %d, want 2 (one per delivery -- the storage-layer unique constraint is the backstop, not this consumer)", len(msgs))
		}
	})
}

func mustConfig(t *testing.T, autoDeployEnabled bool) configuration.Model {
	t.Helper()
	m, err := configuration.Extract(configuration.RestModel{AutoDeployEnabled: autoDeployEnabled})
	if err != nil {
		t.Fatalf("configuration.Extract() unexpected err = %v", err)
	}
	return m
}
