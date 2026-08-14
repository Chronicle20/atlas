package pending_change

import (
	"atlas-character/character"
	"context"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// passingGateDeps is every remote gate stubbed to "not blocking" — the
// baseline every single-gate test in this file overrides exactly one field
// of, so a test that wants to prove gate N rejects never has to worry about
// gates 3..11 it does not care about tripping first.
func passingGateDeps() gateDeps {
	return gateDeps{
		worldStatus: func(_ logrus.FieldLogger, _ context.Context, _ world.Id) (bool, bool, error) {
			return true, false, nil
		},
		accountSlots: func(_ logrus.FieldLogger, _ context.Context, _ uint32) (int16, error) {
			return 99, nil
		},
		banned: func(_ logrus.FieldLogger, _ context.Context, _ uint32) (bool, error) {
			return false, nil
		},
		guildTitle: func(_ logrus.FieldLogger, _ context.Context, _ uint32) (byte, bool, error) {
			return 0, false, nil
		},
		inFamily: func(_ logrus.FieldLogger, _ context.Context, _ uint32) (bool, error) {
			return false, nil
		},
		tradeOpen: func(_ logrus.FieldLogger, _ context.Context, _ uint32) (bool, error) {
			return false, nil
		},
		merchantOpen: func(_ logrus.FieldLogger, _ context.Context, _ uint32) (bool, error) {
			return false, nil
		},
		mtsHolding: func(_ logrus.FieldLogger, _ context.Context, _ uint32) (bool, error) {
			return false, nil
		},
	}
}

// buildCharacter constructs an in-memory character.Model without touching the
// database — evaluateTransferEligibility's local gates (1, 2) read only its
// getters, so a builder-only fixture is enough for those two.
func buildCharacter(id uint32, accountId uint32, worldId world.Id, name string, gm int) character.Model {
	return character.NewModelBuilder().
		SetId(id).
		SetAccountId(accountId).
		SetWorldId(worldId).
		SetName(name).
		SetGm(gm).
		Build()
}

func TestEligibilityGate1WorldSame(t *testing.T) {
	db := newProcessorTestDB(t)
	p := NewProcessor(testLogger(t), testContext(t), db).(*ProcessorImpl).
		withTransferEligibilityGates(passingGateDeps()).(*ProcessorImpl)

	c := buildCharacter(1, 1000, world.Id(0), "Alfa", 0)
	reason, ok := p.evaluateTransferEligibility(c, world.Id(0))
	if ok || reason != "world_same" {
		t.Fatalf("got ok=%v reason=%s, want world_same", ok, reason)
	}
}

func TestEligibilityGate2IsGm(t *testing.T) {
	db := newProcessorTestDB(t)
	p := NewProcessor(testLogger(t), testContext(t), db).(*ProcessorImpl).
		withTransferEligibilityGates(passingGateDeps()).(*ProcessorImpl)

	c := buildCharacter(1, 1000, world.Id(0), "Bravo", 1)
	reason, ok := p.evaluateTransferEligibility(c, world.Id(1))
	if ok || reason != "is_gm" {
		t.Fatalf("got ok=%v reason=%s, want is_gm", ok, reason)
	}
}

func TestEligibilityGate3WorldUnknown(t *testing.T) {
	db := newProcessorTestDB(t)
	deps := passingGateDeps()
	deps.worldStatus = func(_ logrus.FieldLogger, _ context.Context, _ world.Id) (bool, bool, error) {
		return false, false, nil
	}
	p := NewProcessor(testLogger(t), testContext(t), db).(*ProcessorImpl).
		withTransferEligibilityGates(deps).(*ProcessorImpl)

	c := buildCharacter(1, 1000, world.Id(0), "Charlie", 0)
	reason, ok := p.evaluateTransferEligibility(c, world.Id(9))
	if ok || reason != "world_unknown" {
		t.Fatalf("got ok=%v reason=%s, want world_unknown", ok, reason)
	}
}

func TestEligibilityGate3WorldFull(t *testing.T) {
	db := newProcessorTestDB(t)
	deps := passingGateDeps()
	deps.worldStatus = func(_ logrus.FieldLogger, _ context.Context, _ world.Id) (bool, bool, error) {
		return true, true, nil
	}
	p := NewProcessor(testLogger(t), testContext(t), db).(*ProcessorImpl).
		withTransferEligibilityGates(deps).(*ProcessorImpl)

	c := buildCharacter(1, 1000, world.Id(0), "Delta", 0)
	reason, ok := p.evaluateTransferEligibility(c, world.Id(1))
	if ok || reason != "world_full" {
		t.Fatalf("got ok=%v reason=%s, want world_full", ok, reason)
	}
}

func TestEligibilityGate4NoCharacterSlot(t *testing.T) {
	db := newProcessorTestDB(t)
	l, ctx := testLogger(t), testContext(t)
	// The account already holds one character in the destination world; the
	// remote cap (stubbed to 1) leaves no room for a second.
	seedCharacter(t, db, "Existing", world.Id(1))
	// seedCharacter always uses accountId 1000 (refund_idempotency_test.go),
	// which is exactly the accountId this test's fixture character carries.

	deps := passingGateDeps()
	deps.accountSlots = func(_ logrus.FieldLogger, _ context.Context, _ uint32) (int16, error) {
		return 1, nil
	}
	p := NewProcessor(l, ctx, db).(*ProcessorImpl).withTransferEligibilityGates(deps).(*ProcessorImpl)

	c := buildCharacter(1, 1000, world.Id(0), "Echo", 0)
	reason, ok := p.evaluateTransferEligibility(c, world.Id(1))
	if ok || reason != "no_character_slot" {
		t.Fatalf("got ok=%v reason=%s, want no_character_slot", ok, reason)
	}
}

func TestEligibilityGate5NameTaken(t *testing.T) {
	db := newProcessorTestDB(t)
	l, ctx := testLogger(t), testContext(t)
	characterId := seedCharacter(t, db, "Foxtrot", world.Id(0))
	seedCharacter(t, db, "Golf", world.Id(1))

	p := NewProcessor(l, ctx, db).(*ProcessorImpl).
		withTransferEligibilityGates(passingGateDeps()).(*ProcessorImpl)

	c, err := character.NewProcessor(l, ctx, db).GetById()(characterId)
	if err != nil {
		t.Fatalf("GetById: %v", err)
	}
	// The fixture character is "Foxtrot"; rename its in-memory copy to
	// collide with the already-seeded "Golf" so gate 5 trips.
	c = character.CloneModel(c).SetName("Golf").Build()

	reason, ok := p.evaluateTransferEligibility(c, world.Id(1))
	if ok || reason != "name_taken" {
		t.Fatalf("got ok=%v reason=%s, want name_taken", ok, reason)
	}
}

func TestEligibilityGate6Banned(t *testing.T) {
	db := newProcessorTestDB(t)
	deps := passingGateDeps()
	deps.banned = func(_ logrus.FieldLogger, _ context.Context, _ uint32) (bool, error) {
		return true, nil
	}
	p := NewProcessor(testLogger(t), testContext(t), db).(*ProcessorImpl).
		withTransferEligibilityGates(deps).(*ProcessorImpl)

	c := buildCharacter(1, 1000, world.Id(0), "Hotel", 0)
	reason, ok := p.evaluateTransferEligibility(c, world.Id(1))
	if ok || reason != "banned" {
		t.Fatalf("got ok=%v reason=%s, want banned", ok, reason)
	}
}

func TestEligibilityGate7IsGuildMaster(t *testing.T) {
	db := newProcessorTestDB(t)
	deps := passingGateDeps()
	deps.guildTitle = func(_ logrus.FieldLogger, _ context.Context, _ uint32) (byte, bool, error) {
		return guildMasterTitle, true, nil
	}
	p := NewProcessor(testLogger(t), testContext(t), db).(*ProcessorImpl).
		withTransferEligibilityGates(deps).(*ProcessorImpl)

	c := buildCharacter(1, 1000, world.Id(0), "India", 0)
	reason, ok := p.evaluateTransferEligibility(c, world.Id(1))
	if ok || reason != "is_guild_master" {
		t.Fatalf("got ok=%v reason=%s, want is_guild_master", ok, reason)
	}
}

// A non-master guild member is severed by the saga, not blocked (design
// §3.6) — every other gate stays passing, so the request is eligible.
func TestEligibilityGuildMemberIsNotBlocked(t *testing.T) {
	db := newProcessorTestDB(t)
	deps := passingGateDeps()
	deps.guildTitle = func(_ logrus.FieldLogger, _ context.Context, _ uint32) (byte, bool, error) {
		return 3, true, nil
	}
	p := NewProcessor(testLogger(t), testContext(t), db).(*ProcessorImpl).
		withTransferEligibilityGates(deps).(*ProcessorImpl)

	c := buildCharacter(1, 1000, world.Id(0), "Juliett", 0)
	reason, ok := p.evaluateTransferEligibility(c, world.Id(1))
	if !ok || reason != "" {
		t.Fatalf("expected a rank-3 guild member to be eligible, got ok=%v reason=%s", ok, reason)
	}
}

func TestEligibilityGate8InFamily(t *testing.T) {
	db := newProcessorTestDB(t)
	deps := passingGateDeps()
	deps.inFamily = func(_ logrus.FieldLogger, _ context.Context, _ uint32) (bool, error) {
		return true, nil
	}
	p := NewProcessor(testLogger(t), testContext(t), db).(*ProcessorImpl).
		withTransferEligibilityGates(deps).(*ProcessorImpl)

	c := buildCharacter(1, 1000, world.Id(0), "Kilo", 0)
	reason, ok := p.evaluateTransferEligibility(c, world.Id(1))
	if ok || reason != "in_family" {
		t.Fatalf("got ok=%v reason=%s, want in_family", ok, reason)
	}
}

func TestEligibilityGate9TradeOpen(t *testing.T) {
	db := newProcessorTestDB(t)
	deps := passingGateDeps()
	deps.tradeOpen = func(_ logrus.FieldLogger, _ context.Context, _ uint32) (bool, error) {
		return true, nil
	}
	p := NewProcessor(testLogger(t), testContext(t), db).(*ProcessorImpl).
		withTransferEligibilityGates(deps).(*ProcessorImpl)

	c := buildCharacter(1, 1000, world.Id(0), "Lima", 0)
	reason, ok := p.evaluateTransferEligibility(c, world.Id(1))
	if ok || reason != "trade_open" {
		t.Fatalf("got ok=%v reason=%s, want trade_open", ok, reason)
	}
}

func TestEligibilityGate10MerchantOpen(t *testing.T) {
	db := newProcessorTestDB(t)
	deps := passingGateDeps()
	deps.merchantOpen = func(_ logrus.FieldLogger, _ context.Context, _ uint32) (bool, error) {
		return true, nil
	}
	p := NewProcessor(testLogger(t), testContext(t), db).(*ProcessorImpl).
		withTransferEligibilityGates(deps).(*ProcessorImpl)

	c := buildCharacter(1, 1000, world.Id(0), "Mike", 0)
	reason, ok := p.evaluateTransferEligibility(c, world.Id(1))
	if ok || reason != "merchant_open" {
		t.Fatalf("got ok=%v reason=%s, want merchant_open", ok, reason)
	}
}

func TestEligibilityGate11MtsListingsOpen(t *testing.T) {
	db := newProcessorTestDB(t)
	deps := passingGateDeps()
	deps.mtsHolding = func(_ logrus.FieldLogger, _ context.Context, _ uint32) (bool, error) {
		return true, nil
	}
	p := NewProcessor(testLogger(t), testContext(t), db).(*ProcessorImpl).
		withTransferEligibilityGates(deps).(*ProcessorImpl)

	c := buildCharacter(1, 1000, world.Id(0), "November", 0)
	reason, ok := p.evaluateTransferEligibility(c, world.Id(1))
	if ok || reason != "mts_listings_open" {
		t.Fatalf("got ok=%v reason=%s, want mts_listings_open", ok, reason)
	}
}

func TestEligibilityAllGatesPassingIsEligible(t *testing.T) {
	db := newProcessorTestDB(t)
	p := NewProcessor(testLogger(t), testContext(t), db).(*ProcessorImpl).
		withTransferEligibilityGates(passingGateDeps()).(*ProcessorImpl)

	c := buildCharacter(1, 1000, world.Id(0), "Oscar", 0)
	reason, ok := p.evaluateTransferEligibility(c, world.Id(1))
	if !ok || reason != "" {
		t.Fatalf("got ok=%v reason=%s, want eligible", ok, reason)
	}
}

// Ordering is load-bearing: a request failing gate 1 (local) must never fan
// out to any remote service. Every remote gate function here panics if
// invoked; the test passing at all IS the assertion.
func TestEligibilityOrderingShortCircuitsBeforeAnyRemoteCall(t *testing.T) {
	db := newProcessorTestDB(t)
	panicking := gateDeps{
		worldStatus: func(logrus.FieldLogger, context.Context, world.Id) (bool, bool, error) {
			panic("gate 3 (worldStatus) must not be called when gate 1 already rejected")
		},
		accountSlots: func(logrus.FieldLogger, context.Context, uint32) (int16, error) {
			panic("gate 4 (accountSlots) must not be called when gate 1 already rejected")
		},
		banned: func(logrus.FieldLogger, context.Context, uint32) (bool, error) {
			panic("gate 6 (banned) must not be called when gate 1 already rejected")
		},
		guildTitle: func(logrus.FieldLogger, context.Context, uint32) (byte, bool, error) {
			panic("gate 7 (guildTitle) must not be called when gate 1 already rejected")
		},
		inFamily: func(logrus.FieldLogger, context.Context, uint32) (bool, error) {
			panic("gate 8 (inFamily) must not be called when gate 1 already rejected")
		},
		tradeOpen: func(logrus.FieldLogger, context.Context, uint32) (bool, error) {
			panic("gate 9 (tradeOpen) must not be called when gate 1 already rejected")
		},
		merchantOpen: func(logrus.FieldLogger, context.Context, uint32) (bool, error) {
			panic("gate 10 (merchantOpen) must not be called when gate 1 already rejected")
		},
		mtsHolding: func(logrus.FieldLogger, context.Context, uint32) (bool, error) {
			panic("gate 11 (mtsHolding) must not be called when gate 1 already rejected")
		},
	}
	p := NewProcessor(testLogger(t), testContext(t), db).(*ProcessorImpl).
		withTransferEligibilityGates(panicking).(*ProcessorImpl)

	c := buildCharacter(1, 1000, world.Id(3), "Papa", 0)
	reason, ok := p.evaluateTransferEligibility(c, world.Id(3))
	if ok || reason != "world_same" {
		t.Fatalf("got ok=%v reason=%s, want world_same", ok, reason)
	}
}

// CheckTransferEligibility is the read-side entry point the REST endpoint
// uses: it resolves the character by id (rather than requiring the caller to
// already have a character.Model) and runs the same gate table.
func TestCheckTransferEligibilityResolvesTheCharacterAndEvaluates(t *testing.T) {
	db := newProcessorTestDB(t)
	l, ctx := testLogger(t), testContext(t)
	characterId := seedCharacter(t, db, "Quebec", world.Id(0))

	p := NewProcessor(l, ctx, db).withTransferEligibilityGates(passingGateDeps())

	ok, reason, err := p.CheckTransferEligibility(characterId, world.Id(0))
	if err != nil {
		t.Fatalf("CheckTransferEligibility: %v", err)
	}
	if ok || reason != "world_same" {
		t.Fatalf("got ok=%v reason=%s, want world_same", ok, reason)
	}

	ok, reason, err = p.CheckTransferEligibility(characterId, world.Id(1))
	if err != nil {
		t.Fatalf("CheckTransferEligibility: %v", err)
	}
	if !ok || reason != "" {
		t.Fatalf("got ok=%v reason=%s, want eligible", ok, reason)
	}
}
