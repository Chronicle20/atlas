package playernpc

import (
	"atlas-player-npcs/character"
	"atlas-player-npcs/configuration"
	"atlas-player-npcs/inventory"
	"atlas-player-npcs/ranking"
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"gorm.io/gorm"

	mapdata "atlas-player-npcs/data/map"
	npcdata "atlas-player-npcs/data/npc"

	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// -- stub read clients -------------------------------------------------
//
// Deploy/Redeploy take their read clients as injected interfaces (mirroring
// snapshot.Capture's own shape) precisely so this test never makes a real
// HTTP call.

type stubCharacterProcessor struct {
	m character.Model
}

func (s stubCharacterProcessor) GetById(uint32) (character.Model, error) { return s.m, nil }
func (s stubCharacterProcessor) ByNameProvider(string) model.Provider[[]character.Model] {
	return model.FixedProvider([]character.Model{s.m})
}
func (s stubCharacterProcessor) GetByName(string) (character.Model, error) { return s.m, nil }

type stubInventoryProcessor struct{}

func (stubInventoryProcessor) GetByCharacterId(uint32) (inventory.Model, error) {
	return inventory.Extract(inventory.RestModel{})
}

type stubRankingProcessor struct{}

func (stubRankingProcessor) GetByCharacterId(uint32, world.Id) (ranking.Model, error) {
	return ranking.Extract(ranking.RestModel{Rank: 5, JobRank: 3})
}

type stubConfigurationProcessor struct {
	m configuration.Model
}

func (s stubConfigurationProcessor) GetByTenantId(uuid.UUID) configuration.Model { return s.m }

// stubNpcProcessor answers the pool's usable-set build. usable == nil
// treats every candidate id as usable (a real, imitate-flagged template) --
// the convenient default for tests that only care about which id gets
// chosen from an unrestricted pool. A non-nil map restricts usability to
// exactly the ids it lists, for the pool-exhaustion test.
type stubNpcProcessor struct {
	usable map[uint32]bool
}

func (s stubNpcProcessor) GetById(id uint32) (npcdata.Model, error) {
	imitate := true
	if s.usable != nil {
		imitate = s.usable[id]
	}
	return npcdata.Extract(npcdata.RestModel{Id: id, Name: "Statue", Imitate: imitate})
}

type stubMapProcessor struct {
	m        mapdata.Model
	groundFn func([]mapdata.GroundPoint) ([]mapdata.GroundResult, error)
}

func (s stubMapProcessor) GetById(_map.Id) (mapdata.Model, error) { return s.m, nil }
func (s stubMapProcessor) Ground(_ _map.Id, points []mapdata.GroundPoint) ([]mapdata.GroundResult, error) {
	return s.groundFn(points)
}

// identityGround snaps every candidate to itself at foothold 1 -- a
// ground service that always finds solid footing directly below the
// probe, which is all the positioner-geometry tests need.
func identityGround(points []mapdata.GroundPoint) ([]mapdata.GroundResult, error) {
	out := make([]mapdata.GroundResult, 0, len(points))
	for _, pt := range points {
		r, err := mapdata.ExtractGroundResult(mapdata.GroundResultRestModel{X: pt.X(), Y: pt.Y(), Fh: 1, Found: true})
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, nil
}

// -- fixtures ------------------------------------------------------------

func buildCharacterModel(t *testing.T, id uint32, name string, level byte, jobId job.Id, gm bool) character.Model {
	t.Helper()
	gmFlag := 0
	if gm {
		gmFlag = 1
	}
	m, err := character.Extract(character.RestModel{
		Id: id, Name: name, Gender: 0, SkinColor: 1, Face: 20000, Hair: 30000,
		JobId: jobId, Level: level, Gm: gmFlag,
	})
	if err != nil {
		t.Fatalf("character.Extract() unexpected err = %v", err)
	}
	return m
}

func buildConfigModel(t *testing.T, areaX, areaY int16, areaSteps int, organize bool) configuration.Model {
	t.Helper()
	m, err := configuration.Extract(configuration.RestModel{
		InitialX: 0, InitialY: 0, AreaX: areaX, AreaY: areaY, AreaSteps: areaSteps,
		OrganizeArea: organize, AutoDeployEnabled: true,
	})
	if err != nil {
		t.Fatalf("configuration.Extract() unexpected err = %v", err)
	}
	return m
}

func buildMapModel(t *testing.T, mapId uint32, w, h int16) mapdata.Model {
	t.Helper()
	m, err := mapdata.Extract(mapdata.RestModel{
		Id:      mapId,
		MapArea: &mapdata.RectangleRestModel{X: 0, Y: 0, Width: w, Height: h},
	})
	if err != nil {
		t.Fatalf("mapdata.Extract() unexpected err = %v", err)
	}
	return m
}

// deployTestEnv bundles the stub processors a Deploy/Redeploy call needs,
// plus the recorded events, so each test case only overrides what it cares
// about.
type deployTestEnv struct {
	db     *gorm.DB
	ctx    context.Context
	cfg    configuration.Model
	mapM   mapdata.Model
	ground func([]mapdata.GroundPoint) ([]mapdata.GroundResult, error)
	usable map[uint32]bool
	events []Event
}

func newDeployTestEnv(t *testing.T, mapId uint32) *deployTestEnv {
	t.Helper()
	db := testDatabase(t)
	te := testTenant(t)
	ctx := tenant.WithContext(context.Background(), te)
	return &deployTestEnv{
		db:     db,
		ctx:    ctx,
		cfg:    buildConfigModel(t, 100, 100, 2, true),
		mapM:   buildMapModel(t, mapId, 100, 100),
		ground: identityGround,
	}
}

func (e *deployTestEnv) processorFor(c character.Model) Processor {
	return NewProcessor(testLogger(), e.ctx, e.db,
		stubCharacterProcessor{m: c},
		stubInventoryProcessor{},
		stubRankingProcessor{},
		stubConfigurationProcessor{m: e.cfg},
		stubNpcProcessor{usable: e.usable},
		stubMapProcessor{m: e.mapM, groundFn: e.ground},
		func(ev Event) { e.events = append(e.events, ev) },
	)
}

// nonHallMapId is any map id that routing.IsHallOfFameMap doesn't
// recognize, so allocation.BranchFor falls to its GM-deploy formula
// (26 + 4*(mapId/100000000)) regardless of job -- the tests below don't
// need real Hall of Fame routing, only a deterministic branch.
const nonHallMapId = uint32(555000004)

func TestDeploy(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		env := newDeployTestEnv(t, nonHallMapId)
		c := buildCharacterModel(t, 1, "Hero", 200, job.WarriorId, false)
		p := env.processorFor(c)

		created, err := p.Deploy(1, world.Id(0), _map.Id(nonHallMapId), true, nil)
		if err != nil {
			t.Fatalf("Deploy() unexpected err = %v", err)
		}
		if created.Id() == uuid.Nil {
			t.Fatalf("Deploy() did not assign an id")
		}
		// branch 46 -> [9904600, 9904699] (26 + 4*(555000004/100000000) = 46)
		if created.ScriptId() < 9904600 || created.ScriptId() > 9904699 {
			t.Fatalf("ScriptId() = %v, want in [9904600, 9904699]", created.ScriptId())
		}
		if created.X() != 0 || created.Cy() != 0 {
			t.Fatalf("X()/Cy() = %v/%v, want 0/0 (the positioner's single step-0 slot)", created.X(), created.Cy())
		}

		var count int64
		if err := env.db.WithContext(env.ctx).Model(&Entity{}).Count(&count).Error; err != nil {
			t.Fatalf("count unexpected err = %v", err)
		}
		if count != 1 {
			t.Fatalf("player_npcs row count = %v, want 1", count)
		}

		if len(env.events) != 1 || env.events[0].Type != EventTypeDeployed {
			t.Fatalf("events = %+v, want exactly one DEPLOYED", env.events)
		}
		if len(env.events[0].Models) != 1 || env.events[0].Models[0].Id() != created.Id() {
			t.Fatalf("DEPLOYED event model = %+v, want the created row", env.events[0].Models)
		}
	})

	t.Run("rank ordinals", func(t *testing.T) {
		env := newDeployTestEnv(t, nonHallMapId)
		first := buildCharacterModel(t, 1, "First", 200, job.WarriorId, false)
		second := buildCharacterModel(t, 2, "Second", 200, job.WarriorId, false)

		m1, err := env.processorFor(first).Deploy(1, world.Id(0), _map.Id(nonHallMapId), true, nil)
		if err != nil {
			t.Fatalf("first Deploy() unexpected err = %v", err)
		}
		if m1.WorldJobRank() != 1 {
			t.Fatalf("first WorldJobRank() = %v, want 1", m1.WorldJobRank())
		}

		m2, err := env.processorFor(second).Deploy(2, world.Id(0), _map.Id(nonHallMapId), true, nil)
		if err != nil {
			t.Fatalf("second Deploy() unexpected err = %v", err)
		}
		if m2.WorldJobRank() != 2 {
			t.Fatalf("second WorldJobRank() = %v, want 2", m2.WorldJobRank())
		}
	})

	t.Run("overall job ordinal", func(t *testing.T) {
		env := newDeployTestEnv(t, nonHallMapId)
		mapB := buildMapModel(t, nonHallMapId+1, 100, 100)
		first := buildCharacterModel(t, 1, "First", 200, job.WarriorId, false)
		second := buildCharacterModel(t, 2, "Second", 200, job.WarriorId, false)

		m1, err := env.processorFor(first).Deploy(1, world.Id(0), _map.Id(nonHallMapId), true, nil)
		if err != nil {
			t.Fatalf("first Deploy() unexpected err = %v", err)
		}
		if m1.OverallJobRank() != 1 {
			t.Fatalf("first OverallJobRank() = %v, want 1", m1.OverallJobRank())
		}

		// A different world and a different map -- overall_job_rank is a
		// tenant-wide counter over job category, independent of both.
		env.mapM = mapB
		m2, err := env.processorFor(second).Deploy(2, world.Id(1), _map.Id(nonHallMapId+1), true, nil)
		if err != nil {
			t.Fatalf("second Deploy() unexpected err = %v", err)
		}
		if m2.OverallJobRank() != 2 {
			t.Fatalf("second OverallJobRank() = %v, want 2", m2.OverallJobRank())
		}
		if m2.WorldJobRank() != 1 {
			t.Fatalf("second WorldJobRank() = %v, want 1 (a different world's own counter)", m2.WorldJobRank())
		}
	})

	t.Run("duplicate", func(t *testing.T) {
		env := newDeployTestEnv(t, nonHallMapId)
		c := buildCharacterModel(t, 1, "Hero", 200, job.WarriorId, false)
		p := env.processorFor(c)

		if _, err := p.Deploy(1, world.Id(0), _map.Id(nonHallMapId), true, nil); err != nil {
			t.Fatalf("first Deploy() unexpected err = %v", err)
		}
		_, err := p.Deploy(1, world.Id(0), _map.Id(nonHallMapId), true, nil)
		if !errors.Is(err, ErrDuplicate) {
			t.Fatalf("second Deploy() err = %v, want ErrDuplicate", err)
		}

		var count int64
		if err := env.db.WithContext(env.ctx).Model(&Entity{}).Count(&count).Error; err != nil {
			t.Fatalf("count unexpected err = %v", err)
		}
		if count != 1 {
			t.Fatalf("player_npcs row count = %v, want 1 (no row from the rejected duplicate)", count)
		}
	})

	t.Run("pool exhausted", func(t *testing.T) {
		env := newDeployTestEnv(t, nonHallMapId)
		env.usable = map[uint32]bool{9905000: true}
		c := buildCharacterModel(t, 1, "Hero", 200, job.WarriorId, false)

		// Occupy the sole usable id directly through the administrator so
		// the pool is genuinely exhausted without needing the full
		// [9901000, 9906599] range usable.
		existing := buildDeployedNpc(t, 0, nonHallMapId, "Occupant", 9905000, 1)
		te := tenant.MustFromContext(env.ctx)
		if _, err := createPlayerNpc(env.db.WithContext(env.ctx), te.Id(), existing); err != nil {
			t.Fatalf("seeding the occupied id unexpected err = %v", err)
		}

		_, err := env.processorFor(c).Deploy(1, world.Id(0), _map.Id(nonHallMapId), true, nil)
		if !errors.Is(err, ErrPoolExhausted) {
			t.Fatalf("Deploy() err = %v, want ErrPoolExhausted", err)
		}
		if len(env.events) != 0 {
			t.Fatalf("events = %+v, want none", env.events)
		}
	})

	t.Run("map full", func(t *testing.T) {
		env := newDeployTestEnv(t, nonHallMapId)
		env.cfg = buildConfigModel(t, 100, 100, 2, false) // no organizeArea, so no reorganize attempt
		env.mapM = buildMapModel(t, nonHallMapId, 100, 0) // zero-height bounds -> no candidate slot at any step
		c := buildCharacterModel(t, 1, "Hero", 200, job.WarriorId, false)

		_, err := env.processorFor(c).Deploy(1, world.Id(0), _map.Id(nonHallMapId), true, nil)
		if !errors.Is(err, ErrMapFull) {
			t.Fatalf("Deploy() err = %v, want ErrMapFull", err)
		}
		if len(env.events) != 0 {
			t.Fatalf("events = %+v, want none", env.events)
		}
		var count int64
		if err := env.db.WithContext(env.ctx).Model(&Entity{}).Count(&count).Error; err != nil {
			t.Fatalf("count unexpected err = %v", err)
		}
		if count != 0 {
			t.Fatalf("player_npcs row count = %v, want 0", count)
		}
	})

	t.Run("ineligible on the checked path", func(t *testing.T) {
		env := newDeployTestEnv(t, nonHallMapId)
		c := buildCharacterModel(t, 1, "Hero", 10, job.WarriorId, false) // level 10 < max 200

		_, err := env.processorFor(c).Deploy(1, world.Id(0), _map.Id(nonHallMapId), true, nil)
		if !errors.Is(err, ErrIneligible) {
			t.Fatalf("Deploy() err = %v, want ErrIneligible", err)
		}
	})

	t.Run("GM path bypasses level", func(t *testing.T) {
		env := newDeployTestEnv(t, nonHallMapId)
		c := buildCharacterModel(t, 1, "Hero", 10, job.WarriorId, false) // level 10 < max 200

		created, err := env.processorFor(c).Deploy(1, world.Id(0), _map.Id(nonHallMapId), false, nil)
		if err != nil {
			t.Fatalf("Deploy() unexpected err = %v", err)
		}
		if created.Id() == uuid.Nil {
			t.Fatalf("Deploy() did not assign an id")
		}
	})

	t.Run("allocation does not leak on failure", func(t *testing.T) {
		env := newDeployTestEnv(t, nonHallMapId)
		env.cfg = buildConfigModel(t, 100, 100, 2, false)
		env.mapM = buildMapModel(t, nonHallMapId, 100, 0) // forces map_full after allocation
		c := buildCharacterModel(t, 1, "Hero", 200, job.WarriorId, false)

		if _, err := env.processorFor(c).Deploy(1, world.Id(0), _map.Id(nonHallMapId), true, nil); !errors.Is(err, ErrMapFull) {
			t.Fatalf("first Deploy() err = %v, want ErrMapFull", err)
		}

		// Now retry on a map with room. The branch/pool scan is
		// deterministic ascending, so a leaked id would make this second
		// deploy skip 9903000; it doesn't, because the failed transaction
		// rolled back and freed it.
		env.mapM = buildMapModel(t, nonHallMapId, 100, 100)
		created, err := env.processorFor(c).Deploy(1, world.Id(0), _map.Id(nonHallMapId), true, nil)
		if err != nil {
			t.Fatalf("second Deploy() unexpected err = %v", err)
		}
		if created.ScriptId() != 9904600 {
			t.Fatalf("ScriptId() = %v, want 9904600 (the branch's first id, unleaked)", created.ScriptId())
		}
	})

	t.Run("re-deploy in place", func(t *testing.T) {
		env := newDeployTestEnv(t, nonHallMapId)
		c := buildCharacterModel(t, 1, "Hero", 200, job.WarriorId, false)
		p := env.processorFor(c)

		created, err := p.Deploy(1, world.Id(0), _map.Id(nonHallMapId), true, nil)
		if err != nil {
			t.Fatalf("Deploy() unexpected err = %v", err)
		}
		env.events = nil

		updated, err := p.Redeploy(created.Id())
		if err != nil {
			t.Fatalf("Redeploy() unexpected err = %v", err)
		}
		if updated.ScriptId() != created.ScriptId() {
			t.Fatalf("ScriptId() changed on redeploy: %v -> %v", created.ScriptId(), updated.ScriptId())
		}
		if updated.ObjectId() != created.ObjectId() {
			t.Fatalf("ObjectId() changed on redeploy: %v -> %v", created.ObjectId(), updated.ObjectId())
		}
		if updated.X() != created.X() || updated.Cy() != created.Cy() {
			t.Fatalf("position changed on redeploy: (%v,%v) -> (%v,%v)", created.X(), created.Cy(), updated.X(), updated.Cy())
		}
		if len(env.events) != 1 || env.events[0].Type != EventTypeUpdated {
			t.Fatalf("events = %+v, want exactly one UPDATED", env.events)
		}
	})

	t.Run("remove", func(t *testing.T) {
		env := newDeployTestEnv(t, nonHallMapId)
		c := buildCharacterModel(t, 1, "Hero", 200, job.WarriorId, false)
		p := env.processorFor(c)

		created, err := p.Deploy(1, world.Id(0), _map.Id(nonHallMapId), true, nil)
		if err != nil {
			t.Fatalf("Deploy() unexpected err = %v", err)
		}
		env.events = nil

		removed, err := p.RemoveById(created.Id())
		if err != nil {
			t.Fatalf("RemoveById() unexpected err = %v", err)
		}
		if removed.Id() != created.Id() {
			t.Fatalf("RemoveById() returned %v, want %v", removed.Id(), created.Id())
		}

		var rowCount, eqCount int64
		if err := env.db.WithContext(env.ctx).Model(&Entity{}).Count(&rowCount).Error; err != nil {
			t.Fatalf("count unexpected err = %v", err)
		}
		if err := env.db.WithContext(env.ctx).Model(&EquipmentEntity{}).Count(&eqCount).Error; err != nil {
			t.Fatalf("count unexpected err = %v", err)
		}
		if rowCount != 0 || eqCount != 0 {
			t.Fatalf("rowCount/eqCount = %v/%v, want 0/0", rowCount, eqCount)
		}
		if len(env.events) != 1 || env.events[0].Type != EventTypeRemoved {
			t.Fatalf("events = %+v, want exactly one REMOVED", env.events)
		}
	})

	t.Run("reorganize", func(t *testing.T) {
		env := newDeployTestEnv(t, nonHallMapId)
		first := buildCharacterModel(t, 1, "First", 200, job.WarriorId, false)
		second := buildCharacterModel(t, 2, "Second", 200, job.WarriorId, false)

		firstDeployed, err := env.processorFor(first).Deploy(1, world.Id(0), _map.Id(nonHallMapId), true, nil)
		if err != nil {
			t.Fatalf("first Deploy() unexpected err = %v", err)
		}
		if firstDeployed.Step() != 0 {
			t.Fatalf("first Step() = %v, want 0", firstDeployed.Step())
		}
		env.events = nil

		secondDeployed, err := env.processorFor(second).Deploy(2, world.Id(0), _map.Id(nonHallMapId), true, nil)
		if err != nil {
			t.Fatalf("second Deploy() unexpected err = %v", err)
		}
		if secondDeployed.Step() != 1 {
			t.Fatalf("second Step() = %v, want 1 (reorganized)", secondDeployed.Step())
		}

		var repositioned, deployed []Event
		for _, ev := range env.events {
			switch ev.Type {
			case EventTypeRepositioned:
				repositioned = append(repositioned, ev)
			case EventTypeDeployed:
				deployed = append(deployed, ev)
			}
		}
		if len(repositioned) != 1 {
			t.Fatalf("REPOSITIONED events = %v, want exactly 1", len(repositioned))
		}
		if len(repositioned[0].Models) != 1 || repositioned[0].Models[0].Id() != firstDeployed.Id() {
			t.Fatalf("REPOSITIONED models = %+v, want exactly [firstDeployed]", repositioned[0].Models)
		}
		if len(deployed) != 1 {
			t.Fatalf("DEPLOYED events = %v, want exactly 1", len(deployed))
		}

		refreshed, err := env.processorFor(first).GetById(firstDeployed.Id())
		if err != nil {
			t.Fatalf("GetById() unexpected err = %v", err)
		}
		if refreshed.Step() != 1 {
			t.Fatalf("first NPC's persisted Step() = %v, want 1 after reorganize", refreshed.Step())
		}
	})
}
