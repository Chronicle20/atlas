package dragon

import (
	"testing"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/constants"
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func mustTenant(t *testing.T, region string, major, minor uint16) tenant.Model {
	t.Helper()
	ten, err := tenant.Create(uuid.New(), region, major, minor)
	if err != nil {
		t.Fatal(err)
	}
	return ten
}

func TestHasDragonCoversEveryEvanGrowthStage(t *testing.T) {
	ten := mustTenant(t, "GMS", 95, 1)
	stages := []job.Id{2200, 2210, 2211, 2212, 2213, 2214, 2215, 2216, 2217, 2218}
	for _, id := range stages {
		if !HasDragon(ten, id) {
			t.Errorf("job %d must be dragon-bearing", id)
		}
	}
}

func TestHasDragonExcludesEvanBeginnerAndOtherJobs(t *testing.T) {
	ten := mustTenant(t, "GMS", 95, 1)
	for _, id := range []job.Id{2001, 2000, 2100, 2112, 100, 0, 910} {
		if HasDragon(ten, id) {
			t.Errorf("job %d must NOT be dragon-bearing", id)
		}
	}
}

// v83 has no Evan entry in its job table, so Resolve fails and the predicate
// returns false with no version special-case anywhere in the lifecycle code.
func TestHasDragonIsFalseForEveryJobOnV83(t *testing.T) {
	ten := mustTenant(t, "GMS", 83, 1)
	for _, id := range []job.Id{2200, 2214, 2218} {
		if HasDragon(ten, id) {
			t.Errorf("v83 tenant must have no dragon-bearing job, got %d", id)
		}
	}
}

func TestHasDragonOnJms185(t *testing.T) {
	ten := mustTenant(t, "JMS", 185, 1)
	// JMS185 either binds the Evan stages or does not; assert the predicate is
	// consistent with the version table rather than assuming an answer.
	got := HasDragon(ten, 2214)
	want := jms185BindsEvan(t)
	if got != want {
		t.Fatalf("HasDragon(JMS185, 2214) = %v, version table says %v", got, want)
	}
}

func jms185BindsEvan(t *testing.T) bool {
	t.Helper()
	id, ok := constants.For("JMS", 185, 1).Job.Resolve(job.Id(2214))
	return ok && id == job.EvanStage6
}
