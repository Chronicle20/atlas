package script

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	saga "github.com/Chronicle20/atlas/libs/atlas-saga"
	"github.com/Chronicle20/atlas/libs/atlas-script-core/operation"
)

// recordingSagaProcessor is a hand-rolled test recorder implementing
// mapactionsaga.Processor; it captures every saga passed to Create so tests
// can assert on how many operations actually ran.
type recordingSagaProcessor struct {
	created []saga.Saga
}

func (r *recordingSagaProcessor) Create(s saga.Saga) error {
	r.created = append(r.created, s)
	return nil
}

func newTestOperationExecutor() (*OperationExecutor, *recordingSagaProcessor) {
	l, _ := test.NewNullLogger()
	rec := &recordingSagaProcessor{}
	e := NewOperationExecutor(l, context.Background())
	e.sagaP = rec
	return e, rec
}

func TestExecuteOperationUnknownType(t *testing.T) {
	tests := []struct {
		name        string
		execute     func(e *OperationExecutor, f field.Model) error
		wantErr     string
		wantCreated int
	}{
		{
			name: "ExecuteOperation errors and creates nothing",
			execute: func(e *OperationExecutor, f field.Model) error {
				op, err := operation.NewBuilder().SetType("play_sound").Build()
				if err != nil {
					t.Fatalf("operation.NewBuilder().Build(): %v", err)
				}
				return e.ExecuteOperation(f, 1, op)
			},
			wantErr:     "unknown operation type [play_sound]",
			wantCreated: 0,
		},
		{
			name: "ExecuteOperations aborts after unknown operation",
			execute: func(e *OperationExecutor, f field.Model) error {
				fieldEffect, err := operation.NewBuilder().
					SetType("field_effect").
					SetParams(map[string]string{"path": "maplemap/enter/1000000"}).
					Build()
				if err != nil {
					t.Fatalf("operation.NewBuilder().Build(): %v", err)
				}
				playSound, err := operation.NewBuilder().SetType("play_sound").Build()
				if err != nil {
					t.Fatalf("operation.NewBuilder().Build(): %v", err)
				}
				unlockUi, err := operation.NewBuilder().SetType("unlock_ui").Build()
				if err != nil {
					t.Fatalf("operation.NewBuilder().Build(): %v", err)
				}
				return e.ExecuteOperations(f, 1, []operation.Model{fieldEffect, playSound, unlockUi})
			},
			wantErr:     "unknown operation type [play_sound]",
			wantCreated: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e, rec := newTestOperationExecutor()
			f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(910510000)).Build()

			err := tt.execute(e, f)
			if err == nil {
				t.Fatalf("error = nil, want non-nil")
			}
			if err.Error() != tt.wantErr {
				t.Errorf("error = %q, want %q", err.Error(), tt.wantErr)
			}

			if len(rec.created) != tt.wantCreated {
				t.Errorf("len(rec.created) = %d, want %d", len(rec.created), tt.wantCreated)
			}
		})
	}
}

func TestExecuteSpawnMonsterCarriesFieldInstance(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "carries field instance"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e, rec := newTestOperationExecutor()

			inst := uuid.MustParse("11111111-2222-3333-4444-555555555555")
			f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(926000000)).SetInstance(inst).Build()

			op, err := operation.NewBuilder().
				SetType("spawn_monster").
				SetParams(map[string]string{"monsterId": "9100013", "x": "82", "y": "200"}).
				Build()
			if err != nil {
				t.Fatalf("operation.NewBuilder().Build(): %v", err)
			}

			if err := e.ExecuteOperation(f, 1, op); err != nil {
				t.Fatalf("ExecuteOperation() error = %v, want nil", err)
			}

			if len(rec.created) != 1 {
				t.Fatalf("len(rec.created) = %d, want 1", len(rec.created))
			}
			payload, ok := rec.created[0].Steps[0].Payload.(saga.SpawnMonsterPayload)
			if !ok {
				t.Fatalf("Steps[0].Payload is not saga.SpawnMonsterPayload")
			}
			if payload.Instance != inst {
				t.Errorf("payload.Instance = %v, want %v", payload.Instance, inst)
			}
			if payload.MapId != _map.Id(926000000) {
				t.Errorf("payload.MapId = %v, want %v", payload.MapId, _map.Id(926000000))
			}
			if payload.MonsterId != 9100013 {
				t.Errorf("payload.MonsterId = %d, want %d", payload.MonsterId, 9100013)
			}
			if payload.X != 82 {
				t.Errorf("payload.X = %d, want %d", payload.X, 82)
			}
			if payload.Y != 200 {
				t.Errorf("payload.Y = %d, want %d", payload.Y, 200)
			}
			if payload.Count != 1 {
				t.Errorf("payload.Count = %d, want %d", payload.Count, 1)
			}
		})
	}
}

func TestExecuteSpawnMonsterMapIdOverrideClearsInstance(t *testing.T) {
	inst := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	fieldMapId := _map.Id(926000000)
	overrideMapId := _map.Id(910510000)

	tests := []struct {
		name         string
		params       map[string]string
		wantMapId    _map.Id
		wantInstance uuid.UUID
	}{
		{
			name:         "no mapId param uses field mapId and field instance",
			params:       map[string]string{"monsterId": "9100013"},
			wantMapId:    fieldMapId,
			wantInstance: inst,
		},
		{
			name:         "mapId param overrides mapId and clears instance",
			params:       map[string]string{"monsterId": "9100013", "mapId": "910510000"},
			wantMapId:    overrideMapId,
			wantInstance: uuid.Nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e, rec := newTestOperationExecutor()

			f := field.NewBuilder(world.Id(0), channel.Id(1), fieldMapId).SetInstance(inst).Build()

			op, err := operation.NewBuilder().SetType("spawn_monster").SetParams(tt.params).Build()
			if err != nil {
				t.Fatalf("operation.NewBuilder().Build(): %v", err)
			}

			if err := e.ExecuteOperation(f, 1, op); err != nil {
				t.Fatalf("ExecuteOperation() error = %v, want nil", err)
			}

			if len(rec.created) != 1 {
				t.Fatalf("len(rec.created) = %d, want 1", len(rec.created))
			}
			payload, ok := rec.created[0].Steps[0].Payload.(saga.SpawnMonsterPayload)
			if !ok {
				t.Fatalf("Steps[0].Payload is not saga.SpawnMonsterPayload")
			}
			if payload.MapId != tt.wantMapId {
				t.Errorf("payload.MapId = %v, want %v", payload.MapId, tt.wantMapId)
			}
			if payload.Instance != tt.wantInstance {
				t.Errorf("payload.Instance = %v, want %v", payload.Instance, tt.wantInstance)
			}
		})
	}
}

func TestExecuteSpawnMonsterSpawnIfAbsent(t *testing.T) {
	tests := []struct {
		name          string
		params        map[string]string
		wantAbsent    bool
		wantErrSubstr string
	}{
		{name: "absent", params: map[string]string{"monsterId": "9100013"}, wantAbsent: false},
		{name: "true", params: map[string]string{"monsterId": "9100013", "spawnIfAbsent": "true"}, wantAbsent: true},
		{name: "false", params: map[string]string{"monsterId": "9100013", "spawnIfAbsent": "false"}, wantAbsent: false},
		{name: "invalid", params: map[string]string{"monsterId": "9100013", "spawnIfAbsent": "yes"}, wantErrSubstr: "invalid spawnIfAbsent [yes]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e, rec := newTestOperationExecutor()

			inst := uuid.MustParse("11111111-2222-3333-4444-555555555555")
			f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(926000000)).SetInstance(inst).Build()

			op, err := operation.NewBuilder().SetType("spawn_monster").SetParams(tt.params).Build()
			if err != nil {
				t.Fatalf("operation.NewBuilder().Build(): %v", err)
			}

			err = e.ExecuteOperation(f, 1, op)
			if tt.wantErrSubstr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErrSubstr) {
					t.Fatalf("ExecuteOperation() error = %v, want containing %q", err, tt.wantErrSubstr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ExecuteOperation() error = %v, want nil", err)
			}

			payload, ok := rec.created[0].Steps[0].Payload.(saga.SpawnMonsterPayload)
			if !ok {
				t.Fatalf("Steps[0].Payload is not saga.SpawnMonsterPayload")
			}
			if payload.SpawnIfAbsent != tt.wantAbsent {
				t.Errorf("payload.SpawnIfAbsent = %t, want %t", payload.SpawnIfAbsent, tt.wantAbsent)
			}
		})
	}
}

func TestExecuteSpawnMonsterMonsterIdsPicksFromSet(t *testing.T) {
	tests := []struct {
		name string
	}{
		{name: "picks from set"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			inst := uuid.MustParse("11111111-2222-3333-4444-555555555555")
			f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(926000000)).SetInstance(inst).Build()

			op, err := operation.NewBuilder().
				SetType("spawn_monster").
				SetParams(map[string]string{"monsterIds": "3300005,3300006,3300007", "x": "-28", "y": "-67"}).
				Build()
			if err != nil {
				t.Fatalf("operation.NewBuilder().Build(): %v", err)
			}

			seen := map[uint32]bool{}
			for i := 0; i < 200; i++ {
				e, rec := newTestOperationExecutor()
				if err := e.ExecuteOperation(f, 1, op); err != nil {
					t.Fatalf("ExecuteOperation() error = %v, want nil", err)
				}
				payload, ok := rec.created[0].Steps[0].Payload.(saga.SpawnMonsterPayload)
				if !ok {
					t.Fatalf("Steps[0].Payload is not saga.SpawnMonsterPayload")
				}
				if payload.MonsterId != 3300005 && payload.MonsterId != 3300006 && payload.MonsterId != 3300007 {
					t.Fatalf("payload.MonsterId = %d, want one of 3300005, 3300006, 3300007", payload.MonsterId)
				}
				seen[payload.MonsterId] = true
				if payload.X != -28 {
					t.Errorf("payload.X = %d, want %d", payload.X, -28)
				}
				if payload.Y != -67 {
					t.Errorf("payload.Y = %d, want %d", payload.Y, -67)
				}
			}

			for _, id := range []uint32{3300005, 3300006, 3300007} {
				if !seen[id] {
					t.Errorf("monsterId %d never chosen across 200 runs", id)
				}
			}
		})
	}
}

func TestExecuteSpawnMonsterIdParamValidation(t *testing.T) {
	tests := []struct {
		name          string
		params        map[string]string
		wantErrSubstr string
	}{
		{name: "neither", params: map[string]string{"x": "0"}, wantErrSubstr: "spawn_monster operation requires exactly one of monsterId or monsterIds"},
		{name: "both", params: map[string]string{"monsterId": "1", "monsterIds": "2,3"}, wantErrSubstr: "spawn_monster operation requires exactly one of monsterId or monsterIds"},
		{name: "empty monsterIds", params: map[string]string{"monsterIds": ""}, wantErrSubstr: "spawn_monster operation requires exactly one of monsterId or monsterIds"},
		{name: "non-numeric entry", params: map[string]string{"monsterIds": "3300005,abc"}, wantErrSubstr: "invalid monsterIds entry [abc]"},
		{name: "non-numeric monsterId", params: map[string]string{"monsterId": "abc"}, wantErrSubstr: "invalid monsterId [abc]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e, _ := newTestOperationExecutor()

			f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(926000000)).Build()

			op, err := operation.NewBuilder().SetType("spawn_monster").SetParams(tt.params).Build()
			if err != nil {
				t.Fatalf("operation.NewBuilder().Build(): %v", err)
			}

			err = e.ExecuteOperation(f, 1, op)
			if err == nil || !strings.Contains(err.Error(), tt.wantErrSubstr) {
				t.Fatalf("ExecuteOperation() error = %v, want containing %q", err, tt.wantErrSubstr)
			}
		})
	}
}

func TestExecuteSetQuestProgress(t *testing.T) {
	e, rec := newTestOperationExecutor()

	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(130030000)).Build()

	op, err := operation.NewBuilder().
		SetType("set_quest_progress").
		SetParams(map[string]string{"questId": "20010", "infoNumber": "20022", "progress": "1"}).
		Build()
	if err != nil {
		t.Fatalf("operation.NewBuilder().Build(): %v", err)
	}

	if err := e.ExecuteOperation(f, 1, op); err != nil {
		t.Fatalf("ExecuteOperation() error = %v, want nil", err)
	}

	if len(rec.created) != 1 {
		t.Fatalf("len(rec.created) = %d, want 1", len(rec.created))
	}
	if len(rec.created[0].Steps) != 1 {
		t.Fatalf("len(Steps) = %d, want 1", len(rec.created[0].Steps))
	}
	if rec.created[0].Steps[0].Action != saga.SetQuestProgress {
		t.Errorf("Steps[0].Action = %v, want saga.SetQuestProgress", rec.created[0].Steps[0].Action)
	}
	payload, ok := rec.created[0].Steps[0].Payload.(saga.SetQuestProgressPayload)
	if !ok {
		t.Fatalf("Steps[0].Payload is not saga.SetQuestProgressPayload")
	}
	want := saga.SetQuestProgressPayload{
		CharacterId: 1,
		WorldId:     world.Id(0),
		QuestId:     20010,
		InfoNumber:  20022,
		Progress:    "1",
	}
	if payload != want {
		t.Errorf("payload = %+v, want %+v", payload, want)
	}
}

func TestExecuteSetQuestProgressParamValidation(t *testing.T) {
	tests := []struct {
		name          string
		params        map[string]string
		wantErrSubstr string
	}{
		{name: "missing questId", params: map[string]string{"infoNumber": "1", "progress": "1"}, wantErrSubstr: "set_quest_progress operation missing questId parameter"},
		{name: "missing infoNumber", params: map[string]string{"questId": "1", "progress": "1"}, wantErrSubstr: "set_quest_progress operation missing infoNumber parameter"},
		{name: "missing progress", params: map[string]string{"questId": "1", "infoNumber": "1"}, wantErrSubstr: "set_quest_progress operation missing progress parameter"},
		{name: "bad questId", params: map[string]string{"questId": "x", "infoNumber": "1", "progress": "1"}, wantErrSubstr: "invalid questId [x]"},
		{name: "bad infoNumber", params: map[string]string{"questId": "1", "infoNumber": "x", "progress": "1"}, wantErrSubstr: "invalid infoNumber [x]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e, _ := newTestOperationExecutor()

			f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(130030000)).Build()

			op, err := operation.NewBuilder().SetType("set_quest_progress").SetParams(tt.params).Build()
			if err != nil {
				t.Fatalf("operation.NewBuilder().Build(): %v", err)
			}

			err = e.ExecuteOperation(f, 1, op)
			if err == nil || !strings.Contains(err.Error(), tt.wantErrSubstr) {
				t.Fatalf("ExecuteOperation() error = %v, want containing %q", err, tt.wantErrSubstr)
			}
		})
	}
}

func TestExecuteStartQuest(t *testing.T) {
	e, rec := newTestOperationExecutor()

	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(130030000)).Build()

	op, err := operation.NewBuilder().
		SetType("start_quest").
		SetParams(map[string]string{"questId": "22015", "npcId": "9010000"}).
		Build()
	if err != nil {
		t.Fatalf("operation.NewBuilder().Build(): %v", err)
	}

	if err := e.ExecuteOperation(f, 1, op); err != nil {
		t.Fatalf("ExecuteOperation() error = %v, want nil", err)
	}

	if len(rec.created) != 1 {
		t.Fatalf("len(rec.created) = %d, want 1", len(rec.created))
	}
	payload, ok := rec.created[0].Steps[0].Payload.(saga.StartQuestPayload)
	if !ok {
		t.Fatalf("Steps[0].Payload is not saga.StartQuestPayload")
	}
	want := saga.StartQuestPayload{
		CharacterId: 1,
		WorldId:     world.Id(0),
		QuestId:     22015,
		NpcId:       9010000,
	}
	if payload.CharacterId != want.CharacterId || payload.WorldId != want.WorldId || payload.QuestId != want.QuestId || payload.NpcId != want.NpcId {
		t.Errorf("payload = %+v, want %+v", payload, want)
	}
	if payload.Rewards != nil {
		t.Errorf("payload.Rewards = %v, want nil", payload.Rewards)
	}
}

func TestExecuteStartQuestDefaultsNpcIdToZero(t *testing.T) {
	e, rec := newTestOperationExecutor()

	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(130030000)).Build()

	op, err := operation.NewBuilder().
		SetType("start_quest").
		SetParams(map[string]string{"questId": "22015"}).
		Build()
	if err != nil {
		t.Fatalf("operation.NewBuilder().Build(): %v", err)
	}

	if err := e.ExecuteOperation(f, 1, op); err != nil {
		t.Fatalf("ExecuteOperation() error = %v, want nil", err)
	}

	payload, ok := rec.created[0].Steps[0].Payload.(saga.StartQuestPayload)
	if !ok {
		t.Fatalf("Steps[0].Payload is not saga.StartQuestPayload")
	}
	if payload.NpcId != 0 {
		t.Errorf("payload.NpcId = %d, want 0", payload.NpcId)
	}
}

func TestExecuteStartQuestParamValidation(t *testing.T) {
	tests := []struct {
		name          string
		params        map[string]string
		wantErrSubstr string
	}{
		{name: "missing questId", params: map[string]string{}, wantErrSubstr: "start_quest operation missing questId parameter"},
		{name: "bad questId", params: map[string]string{"questId": "x"}, wantErrSubstr: "invalid questId [x]"},
		{name: "bad npcId", params: map[string]string{"questId": "1", "npcId": "x"}, wantErrSubstr: "invalid npcId [x]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e, _ := newTestOperationExecutor()

			f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(130030000)).Build()

			op, err := operation.NewBuilder().SetType("start_quest").SetParams(tt.params).Build()
			if err != nil {
				t.Fatalf("operation.NewBuilder().Build(): %v", err)
			}

			err = e.ExecuteOperation(f, 1, op)
			if err == nil || !strings.Contains(err.Error(), tt.wantErrSubstr) {
				t.Fatalf("ExecuteOperation() error = %v, want containing %q", err, tt.wantErrSubstr)
			}
		})
	}
}

func TestExecuteOpenNpc(t *testing.T) {
	e, rec := newTestOperationExecutor()

	inst := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(931000400)).SetInstance(inst).Build()

	op, err := operation.NewBuilder().
		SetType("open_npc").
		SetParams(map[string]string{"npcId": "2159012"}).
		Build()
	if err != nil {
		t.Fatalf("operation.NewBuilder().Build(): %v", err)
	}

	if err := e.ExecuteOperation(f, 1, op); err != nil {
		t.Fatalf("ExecuteOperation() error = %v, want nil", err)
	}

	if len(rec.created) != 1 {
		t.Fatalf("len(rec.created) = %d, want 1", len(rec.created))
	}
	payload, ok := rec.created[0].Steps[0].Payload.(saga.StartNpcConversationPayload)
	if !ok {
		t.Fatalf("Steps[0].Payload is not saga.StartNpcConversationPayload")
	}
	want := saga.StartNpcConversationPayload{
		CharacterId:   1,
		AccountId:     0,
		NpcTemplateId: 2159012,
		WorldId:       world.Id(0),
		ChannelId:     channel.Id(1),
		MapId:         _map.Id(931000400),
		Instance:      inst,
	}
	if payload != want {
		t.Errorf("payload = %+v, want %+v", payload, want)
	}
}

func TestExecuteOpenNpcParamValidation(t *testing.T) {
	tests := []struct {
		name          string
		params        map[string]string
		wantErrSubstr string
	}{
		{name: "missing npcId", params: map[string]string{}, wantErrSubstr: "open_npc operation missing npcId parameter"},
		{name: "bad npcId", params: map[string]string{"npcId": "x"}, wantErrSubstr: "invalid npcId [x]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e, _ := newTestOperationExecutor()

			inst := uuid.MustParse("11111111-2222-3333-4444-555555555555")
			f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(931000400)).SetInstance(inst).Build()

			op, err := operation.NewBuilder().SetType("open_npc").SetParams(tt.params).Build()
			if err != nil {
				t.Fatalf("operation.NewBuilder().Build(): %v", err)
			}

			err = e.ExecuteOperation(f, 1, op)
			if err == nil || !strings.Contains(err.Error(), tt.wantErrSubstr) {
				t.Fatalf("ExecuteOperation() error = %v, want containing %q", err, tt.wantErrSubstr)
			}
		})
	}
}

func TestExecuteUpdateAreaInfo(t *testing.T) {
	e, rec := newTestOperationExecutor()

	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(931000400)).Build()

	op, err := operation.NewBuilder().
		SetType("update_area_info").
		SetParams(map[string]string{"area": "23007", "info": "exp1=1;exp2=1;exp3=1;exp4=1"}).
		Build()
	if err != nil {
		t.Fatalf("operation.NewBuilder().Build(): %v", err)
	}

	if err := e.ExecuteOperation(f, 1, op); err != nil {
		t.Fatalf("ExecuteOperation() error = %v, want nil", err)
	}

	if len(rec.created) != 1 {
		t.Fatalf("len(rec.created) = %d, want 1", len(rec.created))
	}
	if len(rec.created[0].Steps) != 1 {
		t.Fatalf("len(Steps) = %d, want 1", len(rec.created[0].Steps))
	}
	if rec.created[0].Steps[0].Action != saga.UpdateAreaInfo {
		t.Errorf("Steps[0].Action = %v, want saga.UpdateAreaInfo", rec.created[0].Steps[0].Action)
	}
	payload, ok := rec.created[0].Steps[0].Payload.(saga.UpdateAreaInfoPayload)
	if !ok {
		t.Fatalf("Steps[0].Payload is not saga.UpdateAreaInfoPayload")
	}
	want := saga.UpdateAreaInfoPayload{
		CharacterId: 1,
		WorldId:     world.Id(0),
		ChannelId:   channel.Id(1),
		Area:        23007,
		Info:        "exp1=1;exp2=1;exp3=1;exp4=1",
	}
	if payload != want {
		t.Errorf("payload = %+v, want %+v", payload, want)
	}
}

func TestExecuteShowInfo(t *testing.T) {
	e, rec := newTestOperationExecutor()

	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(931000400)).Build()

	op, err := operation.NewBuilder().
		SetType("show_info").
		SetParams(map[string]string{"path": "Effect/OnUserEff.img/guideEffect/resistanceTutorial/userTalk"}).
		Build()
	if err != nil {
		t.Fatalf("operation.NewBuilder().Build(): %v", err)
	}

	if err := e.ExecuteOperation(f, 1, op); err != nil {
		t.Fatalf("ExecuteOperation() error = %v, want nil", err)
	}

	if len(rec.created) != 1 {
		t.Fatalf("len(rec.created) = %d, want 1", len(rec.created))
	}
	if len(rec.created[0].Steps) != 1 {
		t.Fatalf("len(Steps) = %d, want 1", len(rec.created[0].Steps))
	}
	if rec.created[0].Steps[0].Action != saga.ShowInfo {
		t.Errorf("Steps[0].Action = %v, want saga.ShowInfo", rec.created[0].Steps[0].Action)
	}
	payload, ok := rec.created[0].Steps[0].Payload.(saga.ShowInfoPayload)
	if !ok {
		t.Fatalf("Steps[0].Payload is not saga.ShowInfoPayload")
	}
	want := saga.ShowInfoPayload{
		CharacterId: 1,
		WorldId:     world.Id(0),
		ChannelId:   channel.Id(1),
		Path:        "Effect/OnUserEff.img/guideEffect/resistanceTutorial/userTalk",
	}
	if payload != want {
		t.Errorf("payload = %+v, want %+v", payload, want)
	}
}

func TestExecuteAreaInfoParamValidation(t *testing.T) {
	tests := []struct {
		name          string
		opType        string
		params        map[string]string
		wantErrSubstr string
	}{
		{name: "missing area", opType: "update_area_info", params: map[string]string{"info": "a=1"}, wantErrSubstr: "update_area_info operation missing area parameter"},
		{name: "missing info", opType: "update_area_info", params: map[string]string{"area": "1"}, wantErrSubstr: "update_area_info operation missing info parameter"},
		{name: "bad area", opType: "update_area_info", params: map[string]string{"area": "x", "info": "a=1"}, wantErrSubstr: "invalid area [x]"},
		{name: "area overflows uint16", opType: "update_area_info", params: map[string]string{"area": "70000", "info": "a=1"}, wantErrSubstr: "invalid area [70000]"},
		{name: "missing path", opType: "show_info", params: map[string]string{}, wantErrSubstr: "show_info operation missing path parameter"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e, _ := newTestOperationExecutor()

			f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(931000400)).Build()

			op, err := operation.NewBuilder().SetType(tt.opType).SetParams(tt.params).Build()
			if err != nil {
				t.Fatalf("operation.NewBuilder().Build(): %v", err)
			}

			err = e.ExecuteOperation(f, 1, op)
			if err == nil || !strings.Contains(err.Error(), tt.wantErrSubstr) {
				t.Fatalf("ExecuteOperation() error = %v, want containing %q", err, tt.wantErrSubstr)
			}
		})
	}
}

func TestExecuteClearSkill(t *testing.T) {
	e, rec := newTestOperationExecutor()

	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(914000000)).Build()

	op, err := operation.NewBuilder().
		SetType("clear_skill").
		SetParams(map[string]string{"skillId": "20000014"}).
		Build()
	if err != nil {
		t.Fatalf("operation.NewBuilder().Build(): %v", err)
	}

	if err := e.ExecuteOperation(f, 1, op); err != nil {
		t.Fatalf("ExecuteOperation() error = %v, want nil", err)
	}

	if len(rec.created) != 1 {
		t.Fatalf("len(rec.created) = %d, want 1", len(rec.created))
	}
	if len(rec.created[0].Steps) != 1 {
		t.Fatalf("len(Steps) = %d, want 1", len(rec.created[0].Steps))
	}
	if rec.created[0].Steps[0].Action != saga.ClearSkill {
		t.Errorf("Steps[0].Action = %v, want saga.ClearSkill", rec.created[0].Steps[0].Action)
	}
	payload, ok := rec.created[0].Steps[0].Payload.(saga.ClearSkillPayload)
	if !ok {
		t.Fatalf("Steps[0].Payload is not saga.ClearSkillPayload")
	}
	want := saga.ClearSkillPayload{
		CharacterId: 1,
		WorldId:     world.Id(0),
		SkillId:     20000014,
	}
	if payload != want {
		t.Errorf("payload = %+v, want %+v", payload, want)
	}
}

func TestExecuteClearSkillParamValidation(t *testing.T) {
	tests := []struct {
		name          string
		params        map[string]string
		wantErrSubstr string
	}{
		{name: "missing skillId", params: map[string]string{}, wantErrSubstr: "clear_skill operation missing skillId parameter"},
		{name: "bad skillId", params: map[string]string{"skillId": "x"}, wantErrSubstr: "invalid skillId [x]"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e, rec := newTestOperationExecutor()

			f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(914000000)).Build()

			op, err := operation.NewBuilder().SetType("clear_skill").SetParams(tt.params).Build()
			if err != nil {
				t.Fatalf("operation.NewBuilder().Build(): %v", err)
			}

			err = e.ExecuteOperation(f, 1, op)
			if err == nil || !strings.Contains(err.Error(), tt.wantErrSubstr) {
				t.Fatalf("ExecuteOperation() error = %v, want containing %q", err, tt.wantErrSubstr)
			}
			if len(rec.created) != 0 {
				t.Errorf("len(rec.created) = %d, want 0", len(rec.created))
			}
		})
	}
}
