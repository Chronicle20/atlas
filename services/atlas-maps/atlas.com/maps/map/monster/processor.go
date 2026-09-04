// Package monster implements spawn point management with cooldown tracking.
//
// Spawn Point Cooldown Mechanism:
// This package provides a spawn point registry system that prevents over-spawning
// by enforcing cooldown periods on individual spawn points. The key features include:
//
// - Redis-backed registry scoped by MapKey (tenant/world/channel/map)
// - MobTime-based cooldown enforcement per spawn point (default 5s for normal monsters)
// - Lazy initialization from REST provider
// - Lua scripts for atomic eligibility checks and cooldown updates
// - Maintains existing spawn rate calculations
//
// Architecture:
// - CooldownSpawnPoint: Extends SpawnPoint with NextSpawnAt timestamp
// - ProcessorImpl: Implements spawn logic using Redis-backed SpawnPointRegistry
// - Thread safety: Redis atomicity via Lua scripts
// - Multi-tenant: Separate Redis hashes per MapKey
package monster

import (
	monster2 "atlas-maps/data/map/monster"
	"atlas-maps/map/character"
	"atlas-maps/monster"
	"context"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	routine "github.com/Chronicle20/atlas/libs/atlas-routine"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// defaultSpawnCooldown is the cooldown applied to a spawn point when its
// MobTime is not positive. Matches the historical default.
const defaultSpawnCooldown = 5 * time.Second

type Processor interface {
	SpawnMonsters(transactionId uuid.UUID, field field.Model) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	t   tenant.Model

	dp monster2.Processor
	cp character.Processor
	mp monster.Processor
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		t:   tenant.MustFromContext(ctx),
		dp:  monster2.NewProcessor(l, ctx),
		cp:  character.NewProcessor(l, ctx),
		mp:  monster.NewProcessor(l, ctx),
	}
}

var _ Processor = (*ProcessorImpl)(nil)

// SpawnMonsters implements the core spawn logic with cooldown enforcement.
//
//  1. Initialize the spawn point registry for this field (lazy, one HTTP drain)
//  2. Return early if the field has no characters
//  3. Atomically claim and fire the field's one-time batch, uncapped (FR-2.1..2.4)
//  4. Compute the recurring deficit from the RECURRING spawn-point count only
//  5. Atomically reserve up to that many eligible recurring points and spawn them
func (p *ProcessorImpl) SpawnMonsters(transactionId uuid.UUID, f field.Model) error {
	p.l.Debugf("Executing spawn mechanism for Tenant [%s] Field [%s].", p.t.String(), f.Id())

	mapKey := character.MapKey{
		Tenant: p.t,
		Field:  f,
	}

	registry := GetRegistry()
	if err := registry.InitializeForMap(p.ctx, mapKey, p.dp, p.l); err != nil {
		p.l.WithError(err).Errorf("Failed to initialize spawn points for field [%s].", f.Id())
		return err
	}

	cs, err := p.cp.GetCharactersInMap(transactionId, f)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to retrieve characters in map. Aborting spawning for field [%s].", f.Id())
		return err
	}

	c := len(cs)
	if c <= 0 {
		return nil
	}

	// Claim and fire the field's one-time batch before anything else consults a
	// count. The claim is a single atomic HSETNX inside Lua, so exactly one of
	// the concurrent triggers (character-enter, and the 10s NewRespawn task)
	// can win it; the loser sees an empty batch (FR-2.1..2.4).
	fired, err := registry.ClaimOneTimeSpawnPoints(p.ctx, mapKey)
	if err != nil {
		p.l.WithError(err).Errorf("Failed to claim one-time spawn points for field [%s].", f.Id())
		return err
	}
	if len(fired) > 0 {
		p.l.Debugf("Firing [%d] one-time spawn points for field [%s].", len(fired), f.Id())
		for _, csp := range fired {
			sp := csp.SpawnPoint
			routine.Go(p.l, p.ctx, func(_ context.Context) {
				p.mp.CreateMonster(transactionId, f, sp.Template, sp.X, sp.Y, sp.Fh, sp.Team)
			})
		}
	}

	// Recurring points only: a one-time batch must not inflate the recurring
	// population target (FR-2.5).
	recurringCount, err := registry.Count(p.ctx, mapKey)
	if err != nil {
		p.l.WithError(err).Errorf("Failed to count spawn points for field [%s].", f.Id())
		return err
	}
	if recurringCount == 0 {
		oneTimeCount, cerr := registry.CountOneTime(p.ctx, mapKey)
		if cerr != nil {
			p.l.WithError(cerr).Errorf("Failed to count one-time spawn points for field [%s].", f.Id())
			return cerr
		}
		if oneTimeCount == 0 {
			p.l.Debugf("Field [%s] has no spawn points: one-time [0] recurring [0].", f.Id())
		}
		return nil
	}

	// monstersInMap counts EVERY monster in the field, including the one-time
	// ones. On a mixed map, once the one-time batch is alive it will usually
	// exceed monstersMax, so recurring spawns are suppressed until enough
	// one-time monsters are killed. This is deliberate, matches the reference
	// implementation's map-wide respawn deficit, and is not a regression: no
	// mixed map spawns anything from its one-time set today. Attributing the
	// count per-origin would require an atlas-monsters contract change, which
	// is a PRD non-goal. Do not "fix" this.
	monstersInMap, err := p.mp.CountInMap(transactionId, f)
	if err != nil {
		// Skip this pass rather than assuming zero monsters: a transient count
		// failure treated as zero would spawn the full deficit and over-populate
		// the map. Under-spawning for one tick is the safe direction.
		p.l.WithError(err).Errorf("Unable to count monsters in map; skipping spawn for field [%s] to avoid over-spawn.", f.Id())
		return err
	}

	monstersMax := p.getMonsterMax(c, recurringCount)
	toSpawn := monstersMax - monstersInMap
	if toSpawn <= 0 {
		return nil
	}

	// Atomically select and reserve up to toSpawn eligible spawn points. Folding
	// selection and cooldown reservation into one Redis operation prevents
	// concurrent spawn passes (character-enter + periodic task) from reserving
	// the same points and over-spawning beyond the spawn-point count.
	// Mask the seed into Lua's exact-integer range (< 2^31) so the in-script LCG
	// shuffle stays uniform; Lua numbers are float64 and lose precision above 2^53.
	seed := time.Now().UnixNano() & 0x7fffffff
	reserved, err := registry.ReserveEligibleSpawnPoints(p.ctx, mapKey, toSpawn, defaultSpawnCooldown, seed)
	if err != nil {
		p.l.WithError(err).Errorf("Failed to reserve spawn points for field [%s].", f.Id())
		return err
	}
	if len(reserved) == 0 {
		p.l.Debugf("No eligible spawn points available (all on cooldown) for field [%s].", f.Id())
		return nil
	}

	for _, csp := range reserved {
		sp := csp.SpawnPoint
		p.l.Debugf("Spawning monster at spawn point [%d] with template [%d] at position (%d, %d)", sp.Id, sp.Template, sp.X, sp.Y)
		routine.Go(p.l, p.ctx, func(_ context.Context) {
			p.mp.CreateMonster(transactionId, f, sp.Template, sp.X, sp.Y, sp.Fh, sp.Team)
		})
	}

	p.l.Debugf("Spawned %d monsters out of %d needed for field [%s].", len(reserved), toSpawn, f.Id())
	return nil
}

func (p *ProcessorImpl) getMonsterMax(characterCount int, spawnPointCount int) int {
	spawnRate := 0.70 + (0.05 * math.Min(6, float64(characterCount)))
	return int(math.Ceil(spawnRate * float64(spawnPointCount)))
}
