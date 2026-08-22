package pending_change

import (
	"atlas-character/character"
	"context"
	"errors"
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
		parcelPending: func(_ logrus.FieldLogger, _ context.Context, _ uint32) (bool, error) {
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

// TestEligibilityRemoteGates covers gates 6-11: each overrides exactly one
// gateDeps field to trip (or, for the guild-member case, to deliberately NOT
// trip) while every other gate stays passing. Table-driven because every
// case shares the identical shape — mutate one dependency, evaluate, assert
// (ok, reason) — and only the mutation and expectation differ.
func TestEligibilityRemoteGates(t *testing.T) {
	tests := []struct {
		name       string
		mutate     func(deps *gateDeps)
		character  string
		wantOk     bool
		wantReason string
	}{
		{
			name: "Gate6Banned",
			mutate: func(deps *gateDeps) {
				deps.banned = func(_ logrus.FieldLogger, _ context.Context, _ uint32) (bool, error) {
					return true, nil
				}
			},
			character:  "Hotel",
			wantReason: "banned",
		},
		{
			name: "Gate7IsGuildMaster",
			mutate: func(deps *gateDeps) {
				deps.guildTitle = func(_ logrus.FieldLogger, _ context.Context, _ uint32) (byte, bool, error) {
					return guildMasterTitle, true, nil
				}
			},
			character:  "India",
			wantReason: "is_guild_master",
		},
		{
			// A non-master guild member is severed by the saga, not blocked
			// (design §3.6) — every other gate stays passing, so the request
			// is eligible.
			name: "GuildMemberIsNotBlocked",
			mutate: func(deps *gateDeps) {
				deps.guildTitle = func(_ logrus.FieldLogger, _ context.Context, _ uint32) (byte, bool, error) {
					return 3, true, nil
				}
			},
			character: "Juliett",
			wantOk:    true,
		},
		{
			name: "Gate8InFamily",
			mutate: func(deps *gateDeps) {
				deps.inFamily = func(_ logrus.FieldLogger, _ context.Context, _ uint32) (bool, error) {
					return true, nil
				}
			},
			character:  "Kilo",
			wantReason: "in_family",
		},
		{
			name: "Gate9TradeOpen",
			mutate: func(deps *gateDeps) {
				deps.tradeOpen = func(_ logrus.FieldLogger, _ context.Context, _ uint32) (bool, error) {
					return true, nil
				}
			},
			character:  "Lima",
			wantReason: "trade_open",
		},
		{
			name: "Gate10MerchantOpen",
			mutate: func(deps *gateDeps) {
				deps.merchantOpen = func(_ logrus.FieldLogger, _ context.Context, _ uint32) (bool, error) {
					return true, nil
				}
			},
			character:  "Mike",
			wantReason: "merchant_open",
		},
		{
			name: "Gate11MtsListingsOpen",
			mutate: func(deps *gateDeps) {
				deps.mtsHolding = func(_ logrus.FieldLogger, _ context.Context, _ uint32) (bool, error) {
					return true, nil
				}
			},
			character:  "November",
			wantReason: "mts_listings_open",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newProcessorTestDB(t)
			deps := passingGateDeps()
			tt.mutate(&deps)
			p := NewProcessor(testLogger(t), testContext(t), db).(*ProcessorImpl).
				withTransferEligibilityGates(deps).(*ProcessorImpl)

			c := buildCharacter(1, 1000, world.Id(0), tt.character, 0)
			reason, ok := p.evaluateTransferEligibility(c, world.Id(1))
			if ok != tt.wantOk || reason != tt.wantReason {
				t.Fatalf("got ok=%v reason=%s, want ok=%v reason=%s", ok, reason, tt.wantOk, tt.wantReason)
			}
		})
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
		parcelPending: func(logrus.FieldLogger, context.Context, uint32) (bool, error) {
			panic("gate 12 (parcelPending) must not be called when gate 1 already rejected")
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

// TestEligibilityGateErrorsReportCheckUnavailable is the error-injection
// counterpart to TestEligibilityRemoteGates: for every remote dependency in
// gateDeps, an error from the dependency must NOT surface as the gate's
// affirmative reason (e.g. "banned", "in_family") — that would report a mere
// outage as a finding. The gate still fails CLOSED (ok is always false) but
// the reported reason is the distinct "check_unavailable" (design §6,
// bug-world-transfer-eligibility-reasons.md §2b). The underlying dependency
// error is still logged (WithError) by the production code; this test only
// asserts the reason the caller sees.
func TestEligibilityGateErrorsReportCheckUnavailable(t *testing.T) {
	depErr := errors.New("dependency unavailable")
	tests := []struct {
		name   string
		mutate func(deps *gateDeps)
	}{
		{
			name: "Gate3WorldStatus",
			mutate: func(deps *gateDeps) {
				deps.worldStatus = func(_ logrus.FieldLogger, _ context.Context, _ world.Id) (bool, bool, error) {
					return false, false, depErr
				}
			},
		},
		{
			name: "Gate4AccountSlots",
			mutate: func(deps *gateDeps) {
				deps.accountSlots = func(_ logrus.FieldLogger, _ context.Context, _ uint32) (int16, error) {
					return 0, depErr
				}
			},
		},
		{
			name: "Gate6Banned",
			mutate: func(deps *gateDeps) {
				deps.banned = func(_ logrus.FieldLogger, _ context.Context, _ uint32) (bool, error) {
					return false, depErr
				}
			},
		},
		{
			name: "Gate7GuildTitle",
			mutate: func(deps *gateDeps) {
				deps.guildTitle = func(_ logrus.FieldLogger, _ context.Context, _ uint32) (byte, bool, error) {
					return 0, false, depErr
				}
			},
		},
		{
			name: "Gate8InFamily",
			mutate: func(deps *gateDeps) {
				deps.inFamily = func(_ logrus.FieldLogger, _ context.Context, _ uint32) (bool, error) {
					return false, depErr
				}
			},
		},
		{
			name: "Gate9TradeOpen",
			mutate: func(deps *gateDeps) {
				deps.tradeOpen = func(_ logrus.FieldLogger, _ context.Context, _ uint32) (bool, error) {
					return false, depErr
				}
			},
		},
		{
			name: "Gate10MerchantOpen",
			mutate: func(deps *gateDeps) {
				deps.merchantOpen = func(_ logrus.FieldLogger, _ context.Context, _ uint32) (bool, error) {
					return false, depErr
				}
			},
		},
		{
			name: "Gate11MtsHolding",
			mutate: func(deps *gateDeps) {
				deps.mtsHolding = func(_ logrus.FieldLogger, _ context.Context, _ uint32) (bool, error) {
					return false, depErr
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newProcessorTestDB(t)
			deps := passingGateDeps()
			tt.mutate(&deps)
			p := NewProcessor(testLogger(t), testContext(t), db).(*ProcessorImpl).
				withTransferEligibilityGates(deps).(*ProcessorImpl)

			c := buildCharacter(1, 1000, world.Id(0), tt.name, 0)
			reason, ok := p.evaluateTransferEligibility(c, world.Id(1))
			if ok || reason != "check_unavailable" {
				t.Fatalf("got ok=%v reason=%s, want ok=false reason=check_unavailable", ok, reason)
			}
		})
	}
}

// TestEligibilityGate4ExistingCharacterCountErrorReportsCheckUnavailable
// covers gate 4's second dependency: the LOCAL existing-character count
// (character.GetForAccountInWorld) is not part of the gateDeps seam, so the
// only way to trip its error path is a real DB failure — closing the pool
// underneath the query, exactly as resource_test.go's
// DatabaseFailureIsNotA200ZeroCharacter does.
func TestEligibilityGate4ExistingCharacterCountErrorReportsCheckUnavailable(t *testing.T) {
	db := newProcessorTestDB(t)
	p := NewProcessor(testLogger(t), testContext(t), db).(*ProcessorImpl).
		withTransferEligibilityGates(passingGateDeps()).(*ProcessorImpl)

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB(): %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("sqlDB.Close(): %v", err)
	}

	c := buildCharacter(1, 1000, world.Id(0), "Romeo", 0)
	reason, ok := p.evaluateTransferEligibility(c, world.Id(1))
	if ok || reason != "check_unavailable" {
		t.Fatalf("got ok=%v reason=%s, want ok=false reason=check_unavailable", ok, reason)
	}
}

// TestEligibilityGate5NameTakenCheckErrorReportsCheckUnavailable covers gate
// 5's dependency (character.GetForName), likewise a local DB call outside the
// gateDeps seam.
func TestEligibilityGate5NameTakenCheckErrorReportsCheckUnavailable(t *testing.T) {
	db := newProcessorTestDB(t)
	l, ctx := testLogger(t), testContext(t)
	characterId := seedCharacter(t, db, "Sierra", world.Id(0))

	p := NewProcessor(l, ctx, db).(*ProcessorImpl).
		withTransferEligibilityGates(passingGateDeps()).(*ProcessorImpl)

	c, err := character.NewProcessor(l, ctx, db).GetById()(characterId)
	if err != nil {
		t.Fatalf("GetById: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB(): %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("sqlDB.Close(): %v", err)
	}

	reason, ok := p.evaluateTransferEligibility(c, world.Id(1))
	if ok || reason != "check_unavailable" {
		t.Fatalf("got ok=%v reason=%s, want ok=false reason=check_unavailable", ok, reason)
	}
}

// TestEligibilityGate12ParcelPending covers gate 12 (parcel_pending) on both
// entry points: it is destination-INDEPENDENT, so it must reject/pass
// identically whether reached via CheckTransferEligibility (BUY time) or
// CheckTransferEligibilityIndependent (CHECK time) — the symmetry
// evaluateTransferEligibility and evaluateTransferEligibilityIndependent
// document as load-bearing for FR-28. A dependency error must resolve to
// "check_unavailable", never to the affirmative "parcel_pending" reason
// (design §6): the server failed closed without asserting a reason it does
// not actually know to hold.
func TestEligibilityGate12ParcelPending(t *testing.T) {
	depErr := errors.New("dependency unavailable")

	newProcessor := func(t *testing.T, name string, deps gateDeps) (Processor, uint32) {
		db := newProcessorTestDB(t)
		l, ctx := testLogger(t), testContext(t)
		characterId := seedCharacter(t, db, name, world.Id(0))
		p := NewProcessor(l, ctx, db).withTransferEligibilityGates(deps)
		return p, characterId
	}

	t.Run("blocks buy-time", func(t *testing.T) {
		deps := passingGateDeps()
		deps.parcelPending = func(_ logrus.FieldLogger, _ context.Context, _ uint32) (bool, error) {
			return true, nil
		}
		p, characterId := newProcessor(t, "ParcelBuy1", deps)
		ok, reason, err := p.CheckTransferEligibility(characterId, world.Id(1))
		if err != nil {
			t.Fatalf("CheckTransferEligibility: %v", err)
		}
		if ok || reason != "parcel_pending" {
			t.Fatalf("got ok=%v reason=%s, want ok=false reason=parcel_pending", ok, reason)
		}
	})

	t.Run("blocks check-time", func(t *testing.T) {
		deps := passingGateDeps()
		deps.parcelPending = func(_ logrus.FieldLogger, _ context.Context, _ uint32) (bool, error) {
			return true, nil
		}
		p, characterId := newProcessor(t, "ParcelChk1", deps)
		ok, reason, err := p.CheckTransferEligibilityIndependent(characterId)
		if err != nil {
			t.Fatalf("CheckTransferEligibilityIndependent: %v", err)
		}
		if ok || reason != "parcel_pending" {
			t.Fatalf("got ok=%v reason=%s, want ok=false reason=parcel_pending", ok, reason)
		}
	})

	t.Run("passes buy-time", func(t *testing.T) {
		p, characterId := newProcessor(t, "ParcelBuy2", passingGateDeps())
		ok, reason, err := p.CheckTransferEligibility(characterId, world.Id(1))
		if err != nil {
			t.Fatalf("CheckTransferEligibility: %v", err)
		}
		if !ok || reason != "" {
			t.Fatalf("got ok=%v reason=%s, want eligible", ok, reason)
		}
	})

	t.Run("passes check-time", func(t *testing.T) {
		p, characterId := newProcessor(t, "ParcelChk2", passingGateDeps())
		ok, reason, err := p.CheckTransferEligibilityIndependent(characterId)
		if err != nil {
			t.Fatalf("CheckTransferEligibilityIndependent: %v", err)
		}
		if !ok || reason != "" {
			t.Fatalf("got ok=%v reason=%s, want eligible", ok, reason)
		}
	})

	t.Run("dependency error buy-time", func(t *testing.T) {
		deps := passingGateDeps()
		deps.parcelPending = func(_ logrus.FieldLogger, _ context.Context, _ uint32) (bool, error) {
			return false, depErr
		}
		p, characterId := newProcessor(t, "ParcelBuy3", deps)
		ok, reason, err := p.CheckTransferEligibility(characterId, world.Id(1))
		if err != nil {
			t.Fatalf("CheckTransferEligibility: %v", err)
		}
		if ok || reason != "check_unavailable" {
			t.Fatalf("got ok=%v reason=%s, want ok=false reason=check_unavailable", ok, reason)
		}
	})

	t.Run("dependency error check-time", func(t *testing.T) {
		deps := passingGateDeps()
		deps.parcelPending = func(_ logrus.FieldLogger, _ context.Context, _ uint32) (bool, error) {
			return false, depErr
		}
		p, characterId := newProcessor(t, "ParcelChk3", deps)
		ok, reason, err := p.CheckTransferEligibilityIndependent(characterId)
		if err != nil {
			t.Fatalf("CheckTransferEligibilityIndependent: %v", err)
		}
		if ok || reason != "check_unavailable" {
			t.Fatalf("got ok=%v reason=%s, want ok=false reason=check_unavailable", ok, reason)
		}
	})

	t.Run("runs after mts", func(t *testing.T) {
		deps := passingGateDeps()
		deps.mtsHolding = func(_ logrus.FieldLogger, _ context.Context, _ uint32) (bool, error) {
			return true, nil
		}
		deps.parcelPending = func(_ logrus.FieldLogger, _ context.Context, _ uint32) (bool, error) {
			return true, nil
		}

		buyDb := newProcessorTestDB(t)
		buyId := seedCharacter(t, buyDb, "ParcelMts1", world.Id(0))
		buyP := NewProcessor(testLogger(t), testContext(t), buyDb).withTransferEligibilityGates(deps)
		ok, reason, err := buyP.CheckTransferEligibility(buyId, world.Id(1))
		if err != nil {
			t.Fatalf("CheckTransferEligibility: %v", err)
		}
		if ok || reason != "mts_listings_open" {
			t.Fatalf("got ok=%v reason=%s, want ok=false reason=mts_listings_open (gate 11 precedes gate 12)", ok, reason)
		}

		checkDb := newProcessorTestDB(t)
		checkId := seedCharacter(t, checkDb, "ParcelMts2", world.Id(0))
		checkP := NewProcessor(testLogger(t), testContext(t), checkDb).withTransferEligibilityGates(deps)
		ok, reason, err = checkP.CheckTransferEligibilityIndependent(checkId)
		if err != nil {
			t.Fatalf("CheckTransferEligibilityIndependent: %v", err)
		}
		if ok || reason != "mts_listings_open" {
			t.Fatalf("got ok=%v reason=%s, want ok=false reason=mts_listings_open (gate 11 precedes gate 12)", ok, reason)
		}
	})
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

// TestCheckTransferEligibilityIndependentEvaluatesOnlyTheIndependentHalf is
// the unit-level pin for the OQ-7 split (design §6.1,
// bug-world-transfer-eligibility-reasons.md, "The better fix for 2c"): the
// destination-free entry point rejects on a destination-independent gate
// (in_family) and, when every independent gate passes, reports eligible —
// even for a character whose only "problem" would be a destination-DEPENDENT
// gate (world_same), which this entry point takes no destinationWorldId to
// evaluate at all.
func TestCheckTransferEligibilityIndependentEvaluatesOnlyTheIndependentHalf(t *testing.T) {
	db := newProcessorTestDB(t)
	l, ctx := testLogger(t), testContext(t)
	characterId := seedCharacter(t, db, "Romeo", world.Id(0))

	inFamilyGates := passingGateDeps()
	inFamilyGates.inFamily = func(_ logrus.FieldLogger, _ context.Context, _ uint32) (bool, error) {
		return true, nil
	}
	p := NewProcessor(l, ctx, db).withTransferEligibilityGates(inFamilyGates)

	ok, reason, err := p.CheckTransferEligibilityIndependent(characterId)
	if err != nil {
		t.Fatalf("CheckTransferEligibilityIndependent: %v", err)
	}
	if ok || reason != "in_family" {
		t.Fatalf("got ok=%v reason=%s, want in_family", ok, reason)
	}

	p2 := NewProcessor(l, ctx, db).withTransferEligibilityGates(passingGateDeps())
	ok, reason, err = p2.CheckTransferEligibilityIndependent(characterId)
	if err != nil {
		t.Fatalf("CheckTransferEligibilityIndependent: %v", err)
	}
	if !ok || reason != "" {
		t.Fatalf("got ok=%v reason=%s, want eligible (destination-dependent gates must not run here)", ok, reason)
	}
}
