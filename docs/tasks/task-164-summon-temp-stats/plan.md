# Summon Temporary Stats (PUPPET/SUMMON) Server-Only Classification — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Classify `PUPPET`/`SUMMON` as server-only temporary stats in `libs/atlas-packet` so they skip silently (DEBUG) in `AddStat` instead of logging ERROR, with byte fixtures — across **every supported tenant version** — proving every other stat's wire encoding is unchanged.

> **Plan v2 (2026-08-07).** Rebased onto main at `e0f5bd01d`. Changes from v1:
> line references re-verified (`AddStat` moved 531 → 580, registry builder 61 → 63,
> `character_buff_cancel.go` 21,36 → 22,37); every new test now loops `pt.Variants`
> (all 12 supported versions) instead of two anchors; the byte-offset-skipping
> comparison is replaced by zero-expiry saturated durations so full byte slices can
> be compared on every version (design §5.1); `tools/goroutine-guard.sh` and
> `tools/lint.sh --check` added to the verification gate.

**Architecture:** A package-level `serverOnlyStatNames` set (mirroring the existing `baseStatNames` idiom) consulted at the top of `CharacterTemporaryStat.AddStat`, before the registry lookup. `AddStat` is the single chokepoint for all four atlas-channel CTS encode paths (local give, foreign give, cancel, character-spawn), so the entire production change is one file in the lib — no service, registry, or config changes.

**Tech Stack:** Go, logrus (`hooks/test` for log assertions), existing `libs/atlas-packet/test` helpers (`pt.CreateContext`, `tenant.Create`), byte-fixture assertions.

## Global Constraints

- **Only `libs/atlas-packet` changes.** `services/atlas-channel`, `services/atlas-data`, `services/atlas-buffs`, `services/atlas-summons` must have zero diff (PRD §7, acceptance criterion 5).
- **`buildCharacterTemporaryStatRegistry` must NOT be touched** — it is a shift-ordered enumeration; any entry change moves mask bits (FR-8).
- **The unknown-name ERROR path in `AddStat` stays byte-for-byte intact**: `l.WithError(err).Errorf("Attempting to add buff [%s], but cannot find it.", name)` (FR-3, acceptance (d)).
- **Server-only skip logs at DEBUG on every occurrence** (design §2, open question resolved: plain `Debugf`, no first-add-only state).
- **`AddStat`'s signature is unchanged** — no caller churn (design §4).
- **The skip goes on `CharacterTemporaryStat` only.** `MonsterTemporaryStat` has its own `AddStat` (4 call sites: `monster_spawn.go:20`, `kafka/consumer/monster/consumer.go:475,509,543`) and is explicitly out of scope (PRD §2 non-goals). Do not touch it.
- **Every new test loops `pt.Variants`** — all 12 supported versions (FR-7/FR-7.1), matching the existing suite idiom. Two-anchor coverage is insufficient: it misses the legacy pre-v61 8-byte-mask class.
- **No hard-coded byte offsets in the new tests.** The mask is 8 bytes on GMS `< 61` and 16 bytes elsewhere, so any offset arithmetic breaks on the legacy class. Use zero expiry (`time.Time{}`) for determinism and compare complete byte slices (design §5.1).
- Verification bar: `go test -race ./...`, `go vet ./...`, `go build ./...` clean in `libs/atlas-packet`; `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh`, and `tools/lint.sh --check` clean from the worktree root. No `docker buildx bake` needed for a lib-only change (no service `go.mod` touched).
- Committed files use repo-relative paths only — never literal home/absolute paths.
- All commands below run from the task worktree root (the repo checkout on branch `task-164-summon-temp-stats`) unless a `cd` is shown.

---

### Task 1: Server-only name set + silent skip in `AddStat`

**Files:**
- Modify: `libs/atlas-packet/model/character_temporary_stat.go` (add `serverOnlyStatNames` var immediately above `AddStat`, currently line ~580; add skip inside `AddStat`)
- Test: `libs/atlas-packet/model/character_temporary_stat_test.go` (append new tests)

**Interfaces:**
- Consumes: `character.TemporaryStatTypePuppet` / `character.TemporaryStatTypeSummon` (`libs/atlas-constants/character/temporary_stat.go:123-124`, string values `"PUPPET"`/`"SUMMON"`); existing `AddStat(l)(t)(n, sourceId, amount, level, expiresAt)`.
- Produces: package-level `var serverOnlyStatNames map[character.TemporaryStatType]bool` and the skip behavior Task 2's tests rely on: `AddStat` with a server-only name leaves `m.stats` untouched and logs only at DEBUG.

- [ ] **Step 1: Write the failing test (server-only skip) and the characterization test (unknown name still errors)**

Append to `libs/atlas-packet/model/character_temporary_stat_test.go`. Two new imports are needed: `"github.com/sirupsen/logrus"` and `testlog "github.com/sirupsen/logrus/hooks/test"` (both already dependencies of this module — see `libs/atlas-packet/test/roundtrip.go`).

Note on loggers: existing tests pass `nil` to `AddStat` because the happy path never logs. The server-only skip calls `l.Debugf`, so these tests MUST pass a real logger (a nil `logrus.FieldLogger` interface would panic).

```go
// TestCTSServerOnlyStatsSkippedSilently proves PUPPET/SUMMON never reach the
// wire and never log at ERROR (task-164 FR-1/FR-3, acceptance (a)). Both the
// self and foreign encodes of a CTS holding only server-only stats must be
// byte-identical to a freshly-constructed empty CTS, on EVERY supported tenant
// version (FR-7/FR-7.1 — all seven registry classes, including the legacy
// pre-v61 8-byte-mask class). The two AddStat calls must each log exactly one
// DEBUG entry (skip trace) and nothing at ERROR.
//
// Zero expiry (time.Time{}) is used throughout: it saturates to a constant
// duration on both the modern and legacy writers, so byte comparisons are
// deterministic and need no offset arithmetic (design §5.1).
func TestCTSServerOnlyStatsSkippedSilently(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			tn, _ := tenant.Create([16]byte{}, v.Region, v.MajorVersion, v.MinorVersion)
			l, hook := testlog.NewNullLogger()
			l.SetLevel(logrus.DebugLevel)

			// sourceId/amount/level are arbitrary; only wire disposition matters.
			input := NewCharacterTemporaryStat()
			input.AddStat(l)(tn)(string(character.TemporaryStatTypePuppet), 1, 1, 1, time.Time{})
			input.AddStat(l)(tn)(string(character.TemporaryStatTypeSummon), 2, 1, 1, time.Time{})

			for _, e := range hook.AllEntries() {
				if e.Level <= logrus.ErrorLevel {
					t.Errorf("server-only stat add logged at %s: %q", e.Level, e.Message)
				}
			}
			if got := len(hook.AllEntries()); got != 2 {
				t.Errorf("expected exactly 2 DEBUG skip entries, got %d", got)
			}
			for _, e := range hook.AllEntries() {
				if e.Level != logrus.DebugLevel {
					t.Errorf("skip entry logged at %s, want DEBUG: %q", e.Level, e.Message)
				}
			}

			empty := NewCharacterTemporaryStat()
			if got, want := input.Encode(nil, ctx)(nil), empty.Encode(nil, ctx)(nil); !bytes.Equal(got, want) {
				t.Errorf("Encode with server-only stats differs from empty CTS:\ngot  % x\nwant % x", got, want)
			}
			if got, want := input.EncodeForeign(nil, ctx)(nil), empty.EncodeForeign(nil, ctx)(nil); !bytes.Equal(got, want) {
				t.Errorf("EncodeForeign with server-only stats differs from empty CTS:\ngot  % x\nwant % x", got, want)
			}
		})
	}
}

// TestCTSAddStatUnknownNameStillErrors pins the existing behavior for
// genuinely unregistered stat names (task-164 acceptance (d)): the stat is
// dropped AND the ERROR log fires. Guards against the server-only skip
// accidentally widening into a general silent-drop. Looped over every
// supported version because the error path must NOT become version-dependent.
func TestCTSAddStatUnknownNameStillErrors(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			tn, _ := tenant.Create([16]byte{}, v.Region, v.MajorVersion, v.MinorVersion)
			l, hook := testlog.NewNullLogger()

			input := NewCharacterTemporaryStat()
			input.AddStat(l)(tn)("BOGUS", 1, 1, 1, time.Time{})

			errorEntries := 0
			for _, e := range hook.AllEntries() {
				if e.Level == logrus.ErrorLevel {
					errorEntries++
					if e.Message != "Attempting to add buff [BOGUS], but cannot find it." {
						t.Errorf("unexpected error message: %q", e.Message)
					}
				}
			}
			if errorEntries != 1 {
				t.Errorf("expected exactly 1 ERROR entry for unknown stat, got %d", errorEntries)
			}

			empty := NewCharacterTemporaryStat()
			if got, want := input.Encode(nil, ctx)(nil), empty.Encode(nil, ctx)(nil); !bytes.Equal(got, want) {
				t.Errorf("unknown stat leaked into encode:\ngot  % x\nwant % x", got, want)
			}
		})
	}
}
```

- [ ] **Step 2: Run the new tests to verify the skip test fails and the characterization test passes**

```bash
cd libs/atlas-packet && go test ./model/ -run 'TestCTSServerOnlyStatsSkippedSilently|TestCTSAddStatUnknownNameStillErrors' -v
```

Expected: `TestCTSServerOnlyStatsSkippedSilently` FAILs on the log assertions, on **every** version subtest — current code logs `Attempting to add buff [PUPPET], but cannot find it.` at ERROR (entries are ErrorLevel, not DebugLevel). The byte-equality assertions pass even today (lookup failure already drops the stat); the log assertions are the failing signal. `TestCTSAddStatUnknownNameStillErrors` PASSes on every version subtest (characterization of existing behavior).

- [ ] **Step 3: Implement the server-only set and the skip**

In `libs/atlas-packet/model/character_temporary_stat.go`, insert immediately above `func (m *CharacterTemporaryStat) AddStat` (currently line 580):

```go
// serverOnlyStatNames are temporary stats that exist only for server-side
// lifecycle bookkeeping (Odin lineage). No supported client has a
// SecondaryStat bit for them — IDA-verified across every version Atlas holds
// a binary for (GMS v48/v61/v72/v79/v83/v84/v87/v92/v95, JMS v185), see
// docs/tasks/task-164-summon-temp-stats/prd.md §1.1 — so they are never
// encoded into any CTS mask or payload, on any tenant version. Summon
// visibility for observers is carried by the summon object packets
// (task-088/106), not by a buff. Adding a name here requires the same
// IDA evidence trail.
var serverOnlyStatNames = map[character.TemporaryStatType]bool{
	character.TemporaryStatTypePuppet: true,
	character.TemporaryStatTypeSummon: true,
}
```

Then modify the body of `AddStat` — the existing code is:

```go
			name := character.TemporaryStatType(n)
			st, err := CharacterTemporaryStatTypeByName(t)(name)
			if err != nil {
				l.WithError(err).Errorf("Attempting to add buff [%s], but cannot find it.", name)
				return
			}
```

Change it to (only the two lines between `name := ...` and `st, err := ...` are new; the lookup and error path are untouched):

```go
			name := character.TemporaryStatType(n)
			if serverOnlyStatNames[name] {
				l.Debugf("Skipping server-only temporary stat [%s]; it has no client wire representation.", name)
				return
			}
			st, err := CharacterTemporaryStatTypeByName(t)(name)
			if err != nil {
				l.WithError(err).Errorf("Attempting to add buff [%s], but cannot find it.", name)
				return
			}
```

- [ ] **Step 4: Run the new tests to verify they pass, then the whole package for registry invariance**

```bash
cd libs/atlas-packet && go test ./model/ -run 'TestCTSServerOnlyStatsSkippedSilently|TestCTSAddStatUnknownNameStillErrors' -v
```

Expected: both PASS, on all 12 version subtests each.

```bash
cd libs/atlas-packet && go test -race ./...
```

Expected: PASS with zero failures — in particular the pre-existing byte fixtures pass unmodified, proving no mask bit or per-stat shape moved (FR-8) in any registry class:

| Class | Fixtures that must stay green |
|---|---|
| Legacy / v61 | `TestCTSHomingBeaconV61PopulatedBlock`, `TestCTSHomingBeaconV61RoundTrip`, `TestCTSHomingBeaconLegacyVersionsHaveNoTrailer`, `TestLegacyDurationUnitsNoExpirySaturates` |
| Mid GMS (v72–v83) | `TestCTSEncodeSlowDiseasePerStatLayout`, `TestCTSEncodeBuffPerStatLayout`, `TestCTSMonsterRidingV83MaskAndNoDoubleEncode`, `TestCTSHomingBeaconPre95PopulatedBlock`, `TestNoExpiryStatEncodesSaturatedDuration` |
| v95 | `TestCTSMonsterRidingV95MaskAndLayout`, `TestCTSHomingBeaconV95MaskAndBlock`, `TestCTSPartyBoosterV95Block`, `TestCTSEmptyV95ClaimsNothing`, `TestCTSForeignEmptyV95ClaimsNothing`, `TestCTSHomingBeaconV95RoundTrip`, `TestCTSPartyBoosterV95RoundTrip` |
| All variants | `TestCTSForeignEmptyRoundTrip`, `TestCTSForeignSingleStatRoundTrip`, `TestCTSForeignMultiStatRoundTrip`, `TestCTSAbsentTwoStateStatsEmitNoTrailer`, `TestCTSTwoStateGroupShape`, `TestMaskContainsOnlyActiveStats`, `TestMaskEmptyForEmptyCTS`, `TestMovementAffectingMaskMembership` |

Any fixture diff is a hard failure: revert and re-check the change touched only `AddStat` + the new var.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-packet/model/character_temporary_stat.go libs/atlas-packet/model/character_temporary_stat_test.go
git commit -m "feat(atlas-packet): classify PUPPET/SUMMON as server-only CTS stats (task-164)"
```

---

### Task 2: Byte-equality invariance fixtures (mixed buff, pure server-only buff)

**Files:**
- Test: `libs/atlas-packet/model/character_temporary_stat_test.go` (append two tests)

**Interfaces:**
- Consumes: `serverOnlyStatNames`-driven skip in `AddStat` from Task 1 (a server-only name leaves `m.stats` untouched); existing `Encode`/`EncodeForeign`.
- Produces: regression fixtures only — no new production symbols.

These are **regression pins, not failing-first TDD tests**: after Task 1 they pass immediately (and their byte-equality halves pass even against pre-task-164 code, since the failed lookup already dropped the stats). Their purpose is to hard-fail any future change that gives PUPPET/SUMMON a registry entry or otherwise leaks them into the wire (design §7 risk 1 — e.g. someone "helpfully" registering them would flip mixed-buff byte-equality).

- [ ] **Step 1: Write the mixed-buff and pure-server-only fixtures**

Append to `libs/atlas-packet/model/character_temporary_stat_test.go`:

```go
// TestCTSMixedBuffServerOnlyByteInvariance proves a buff carrying both a wire
// stat and server-only stats encodes byte-identically to the same buff without
// the server-only stats (task-164 acceptance (b)), on EVERY supported tenant
// version.
//
// Determinism: the self Encode path writes each per-stat duration as a function
// of expiresAt evaluated at encode time, which is not stable across two Encode
// calls. Passing the zero time saturates that field to a constant on both the
// modern and legacy writers (pinned by TestNoExpiryStatEncodesSaturatedDuration
// and TestLegacyDurationUnitsNoExpirySaturates), so the FULL byte slices compare
// equal with no offset arithmetic — which also keeps this test correct on the
// legacy pre-v61 class, whose mask is 8 bytes rather than 16 (design §5.1).
//
// Booster is the wire-stat probe because it is registered unconditionally
// (shift 11, before any version gate), so it exists in every registry class and
// sits inside the legacy mask's bits 0-46.
func TestCTSMixedBuffServerOnlyByteInvariance(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			tn, _ := tenant.Create([16]byte{}, v.Region, v.MajorVersion, v.MinorVersion)
			l, _ := testlog.NewNullLogger()

			// Booster + the two server-only stats. sourceId/amount arbitrary;
			// only wire disposition is under test.
			mixed := NewCharacterTemporaryStat()
			mixed.AddStat(l)(tn)(string(character.TemporaryStatTypeBooster), 1001, -2, 1, time.Time{})
			mixed.AddStat(l)(tn)(string(character.TemporaryStatTypePuppet), 1002, 1, 1, time.Time{})
			mixed.AddStat(l)(tn)(string(character.TemporaryStatTypeSummon), 1003, 1, 1, time.Time{})

			plain := NewCharacterTemporaryStat()
			plain.AddStat(l)(tn)(string(character.TemporaryStatTypeBooster), 1001, -2, 1, time.Time{})

			if got, want := mixed.Encode(nil, ctx)(nil), plain.Encode(nil, ctx)(nil); !bytes.Equal(got, want) {
				t.Errorf("Encode differs:\ngot  % x\nwant % x", got, want)
			}
			if got, want := mixed.EncodeForeign(nil, ctx)(nil), plain.EncodeForeign(nil, ctx)(nil); !bytes.Equal(got, want) {
				t.Errorf("EncodeForeign differs:\ngot  % x\nwant % x", got, want)
			}
		})
	}
}

// TestCTSPureServerOnlyBuffEncodesAsEmpty proves a buff whose changes are ALL
// server-only yields exactly the empty-CTS body (task-164 acceptance (c),
// FR-5/FR-6): mask claims nothing, no per-stat blocks, standard trailer. The
// buff writers emit unconditionally, so these bytes are what an emitted
// empty-mask GIVE_BUFF / cancel-reset carries — on every supported version,
// including the legacy class where that mask is 8 zero bytes.
func TestCTSPureServerOnlyBuffEncodesAsEmpty(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			tn, _ := tenant.Create([16]byte{}, v.Region, v.MajorVersion, v.MinorVersion)
			l, _ := testlog.NewNullLogger()

			pure := NewCharacterTemporaryStat()
			pure.AddStat(l)(tn)(string(character.TemporaryStatTypePuppet), 1, 1, 1, time.Time{})

			empty := NewCharacterTemporaryStat()
			if got, want := pure.Encode(nil, ctx)(nil), empty.Encode(nil, ctx)(nil); !bytes.Equal(got, want) {
				t.Errorf("pure server-only Encode differs from empty CTS:\ngot  % x\nwant % x", got, want)
			}
			if got, want := pure.EncodeForeign(nil, ctx)(nil), empty.EncodeForeign(nil, ctx)(nil); !bytes.Equal(got, want) {
				t.Errorf("pure server-only EncodeForeign differs from empty CTS:\ngot  % x\nwant % x", got, want)
			}
		})
	}
}
```

- [ ] **Step 2: Run the new tests**

```bash
cd libs/atlas-packet && go test ./model/ -run 'TestCTSMixedBuffServerOnlyByteInvariance|TestCTSPureServerOnlyBuffEncodesAsEmpty' -v
```

Expected: PASS — both tests, on all 12 version subtests each.

- [ ] **Step 3: Run the full package with the race detector**

```bash
cd libs/atlas-packet && go test -race ./...
```

Expected: PASS, zero failures.

- [ ] **Step 4: Commit**

```bash
git add libs/atlas-packet/model/character_temporary_stat_test.go
git commit -m "test(atlas-packet): pin PUPPET/SUMMON server-only byte invariance (task-164)"
```

---

### Task 3: Module verification and scope audit

**Files:**
- No file changes expected. This task is the verification gate (PRD §8, acceptance criteria 3–5).

**Interfaces:**
- Consumes: the committed state from Tasks 1–2.
- Produces: verified branch ready for code review.

- [ ] **Step 1: Full verification suite in the changed module**

```bash
cd libs/atlas-packet && go test -race ./... && go vet ./... && go build ./...
```

Expected: all three clean (no output from vet/build, `ok` lines from test).

- [ ] **Step 2: Repo-root guards**

```bash
tools/redis-key-guard.sh && tools/goroutine-guard.sh && tools/lint.sh --check
```

Expected: all clean, exit 0. This change adds no Redis usage and no goroutines; the guards are part of the standard bar. Run from the repo/worktree root, without a `GOWORK=off` prefix.

Notes:
- `tools/lint.sh --check` needs Node available for the atlas-ui half — source `nvm` (Node 22) first or it false-fails; it can also false-fail on golangci-lint lock contention if another worktree is linting concurrently. If it reports formatting diffs, run `tools/lint.sh` (no flags) to fix in place, then re-check and amend.
- The template guards (`template-opcode-order-guard.sh`, `template-movement-types-guard.sh`), `service-registration-guard.sh`, `skill-job-id-guard.sh`, and `buff-duration-guard.sh` do **not** apply — this branch touches no template, no services.json/deploy file, no job/skill id comparison, and no Kafka buff-command `duration` field.

- [ ] **Step 3: Scope audit — prove only libs/atlas-packet (+ task docs) changed**

```bash
git diff --stat main...HEAD -- . ':!docs/tasks'
```

Expected: exactly two files — `libs/atlas-packet/model/character_temporary_stat.go` and `libs/atlas-packet/model/character_temporary_stat_test.go`. If anything under `services/` appears, STOP: acceptance criterion 5 forbids changes in atlas-data/atlas-buffs/atlas-summons statup or lifecycle code, and the design expects zero atlas-channel diff. No service `go.mod` is touched, so `docker buildx bake` is not required.

- [ ] **Step 4: Confirm branch and worktree are correct**

```bash
git branch --show-current
```

Expected: `task-164-summon-temp-stats`.

---

## Post-merge runtime acceptance (not part of this plan's execution)

Acceptance criterion 1 (no `cannot find it` ERROR for PUPPET/SUMMON in live atlas-channel logs across cast, observer spawn-in, expiry, summon death) is validated after deployment by casting a puppet skill and a summon skill on a v83 tenant and grepping atlas-channel logs. Recorded here so the reviewer knows it is deliberately out of the automated plan, per design §6.
