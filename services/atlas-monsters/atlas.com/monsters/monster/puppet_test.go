package monster

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// idsProvider returns a model.Provider yielding the given character ids — the
// "characters in field" candidate pool the controller picker draws from.
func idsProvider(ids ...uint32) model.Provider[[]uint32] {
	return func() ([]uint32, error) {
		return ids, nil
	}
}

// TestAddPuppetBiasesController verifies that when a puppet owned by a character
// sits within vicinity (distanceSq < 177777) of a monster, the controller picker
// prefers that puppet's owner over the default least-controlled candidate.
func TestAddPuppetBiasesController(t *testing.T) {
	r := GetMonsterRegistry()
	pr := GetPuppetRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)
	pr.Clear(ctx)

	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()

	// Monster at (100,100).
	m := r.CreateMonster(ctx, ten, f, 9300018, 100, 100, 0, 5, 0, 100, 50)

	// Default candidate (least-controlled) would be char 1 (it controls
	// nothing yet, neither does char 2 — first iterated wins by tie).
	// Puppet owner is char 2; placed at (110,110): distanceSq=200 < 177777.
	puppetOwner := uint32(2)
	pr.Add(ctx, ten, f, puppetOwner, 110, 110)

	p := &ProcessorImpl{l: logrus.New(), ctx: ctx, t: ten}
	cid, err := p.getControllerCandidate(f, m.X(), m.Y(), idsProvider(1, 2))
	if err != nil {
		t.Fatalf("getControllerCandidate: %v", err)
	}
	if cid != puppetOwner {
		t.Fatalf("expected puppet owner [%d] to be preferred as controller, got [%d]", puppetOwner, cid)
	}
}

// TestRemovePuppetClearsBias verifies that after the puppet is removed the bias
// is gone and the picker falls back to the default least-controlled selection.
func TestRemovePuppetClearsBias(t *testing.T) {
	r := GetMonsterRegistry()
	pr := GetPuppetRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)
	pr.Clear(ctx)

	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := r.CreateMonster(ctx, ten, f, 9300018, 100, 100, 0, 5, 0, 100, 50)

	// Make char 2 already control a monster so the least-controlled fallback is
	// deterministically char 1 (map iteration over equal counts is unordered).
	other := r.CreateMonster(ctx, ten, f, 9300018, 500, 500, 0, 5, 0, 100, 50)
	_, _ = r.ControlMonster(ten, other.UniqueId(), 2)

	puppetOwner := uint32(2)
	pr.Add(ctx, ten, f, puppetOwner, 110, 110)
	pr.Remove(ctx, ten, f, puppetOwner)

	p := &ProcessorImpl{l: logrus.New(), ctx: ctx, t: ten}
	cid, err := p.getControllerCandidate(f, m.X(), m.Y(), idsProvider(1, 2))
	if err != nil {
		t.Fatalf("getControllerCandidate: %v", err)
	}
	if cid == puppetOwner {
		t.Fatalf("expected no puppet bias after removal; got puppet owner [%d]", cid)
	}
	if cid != 1 {
		t.Fatalf("expected default least-controlled candidate [1], got [%d]", cid)
	}
}

// TestPuppetOutOfVicinityNoBias verifies a puppet beyond the vicinity threshold
// does not bias the pick.
func TestPuppetOutOfVicinityNoBias(t *testing.T) {
	r := GetMonsterRegistry()
	pr := GetPuppetRegistry()
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := context.Background()
	r.Clear(ctx)
	pr.Clear(ctx)

	f := field.NewBuilder(world.Id(0), channel.Id(0), _map.Id(40000)).Build()
	m := r.CreateMonster(ctx, ten, f, 9300018, 0, 0, 0, 5, 0, 100, 50)

	// Make char 2 already control a monster so the least-controlled fallback is
	// deterministically char 1 (map iteration over equal counts is unordered).
	other := r.CreateMonster(ctx, ten, f, 9300018, 500, 500, 0, 5, 0, 100, 50)
	_, _ = r.ControlMonster(ten, other.UniqueId(), 2)

	// Puppet far away: (1000,0) -> distanceSq=1_000_000 > 177777.
	puppetOwner := uint32(2)
	pr.Add(ctx, ten, f, puppetOwner, 1000, 0)

	p := &ProcessorImpl{l: logrus.New(), ctx: ctx, t: ten}
	cid, err := p.getControllerCandidate(f, m.X(), m.Y(), idsProvider(1, 2))
	if err != nil {
		t.Fatalf("getControllerCandidate: %v", err)
	}
	if cid != 1 {
		t.Fatalf("expected default candidate [1] (puppet out of vicinity), got [%d]", cid)
	}
}
