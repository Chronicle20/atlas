# Zombify Healing Consequences — Design

Version: v1
Status: Draft
Created: 2026-08-21
Input: `docs/tasks/task-256-zombify-healing-consequences/prd.md` (approved)

---

## 0. Resolved open questions

### OQ-1 — Cleric Heal `hpCon` across provisioned versions: **0 everywhere. No suppression code.**

Queried the live `atlas-data` skill resource in `atlas-main` for skill `2301002`,
once per provisioned main-environment tenant, extracting every `HPConsume` value
across all 30 effect levels:

```
v48    count=30 distinct=["HPConsume":0 ]
v61    count=30 distinct=["HPConsume":0 ]
v72    count=30 distinct=["HPConsume":0 ]
v79    count=30 distinct=["HPConsume":0 ]
v83    count=30 distinct=["HPConsume":0 ]
v84    count=30 distinct=["HPConsume":0 ]
v87    count=30 distinct=["HPConsume":0 ]
v92    count=30 distinct=["HPConsume":0 ]
v95    count=30 distinct=["HPConsume":0 ]
jms185 count=30 distinct=["HPConsume":0 ]
```

Ten tenants (GMS v48/61/72/79/83/84/87/92/95 and JMS v185 — the full
`environment: main` set from `atlas-tenants`), 300 effect rows, one distinct
value: `0`. `MPConsume` is 12 at level 1, confirming the rows are real data and
not a zero-filled default.

**Decision:** FR-17's conditional is not triggered. `common.go:137` already gates
the cast-cost `ChangeHP` on `e.HPConsume() > 0`, so the reference's `hpCon = 0`
forcing is already a structural no-op on every provisioned version. **No code is
written for FR-17**, and no test asserts a suppression that does not exist. If a
future version ingests a non-zero Heal `hpCon`, the suppression becomes live work
at `skill/handler/common.go:137`; this finding is the record that it was checked,
not skipped.

### OQ-2 — `BUFFS` service root for `atlas-consumables`: **already available. No deployment change.**

`requests.RootUrlFor(ctx, "BUFFS")` (`libs/atlas-rest/requests/url.go:34-64`)
resolves in two steps: a per-domain `BUFFS_SERVICE_URL` override if present,
otherwise `BASE_SERVICE_URL` rewritten for the caller's environment namespace.

A repo-wide grep for `BUFFS_SERVICE_URL` under `deploy/` returns nothing — no
service anywhere sets the per-domain override; every `BUFFS` consumer
(`atlas-channel`, `atlas-monsters`, `atlas-rates`, `atlas-effective-stats`,
`atlas-query-aggregator`) reaches atlas-buffs through the shared ingress via
`BASE_SERVICE_URL`. The override name appears only in `t.Setenv` calls inside
tests.

`atlas-consumables` already takes `envFrom: configMapRef: atlas-env`
(`deploy/k8s/base/atlas-consumables.yaml:21-23`) — the same ConfigMap every other
service uses — and already resolves six domains through `RootUrlFor`
(`DATA`, `MAPS`, `PETS`, `SKILLS`, `MONSTERS`, `INVENTORY`).

**Decision:** adding a `BUFFS` read to `atlas-consumables` requires **zero**
deployment or ConfigMap change. It is the same base URL those six already use.

### OQ-3 — killer attribution: **accept `KillerTypeUnknown`.**

`ChangeHP` hard-codes `character2.KillerTypeUnknown`
(`services/atlas-character/atlas.com/character/character/processor.go:1404`).
Caster attribution would require a new `ChangeHP` variant carrying a killer id,
widening the `atlas-character` command surface for one edge case. The PRD's
proposed default stands: no `atlas-character` change.

### OQ-4 — client feedback: **no additional packet.**

The client renders whatever HP delta it is told about. A half-heal is already
visible as a smaller number; a negated Heal is already visible as an HP drop.
Nothing is added.

### OQ-5 — HP restoration outside `ApplyItemEffects`: **one path exists, and it is in scope.**

Grepping every non-test `ChangeHP` call site in `atlas-consumables` returns two:

| Site | Path |
|---|---|
| `consumable/processor.go:250` | `ApplyItemEffects` → `plan.hpChanges` |
| `consumable/morph_coupon.go:116` | `consumeMorphCoupon` → `plan.hp` |

The second is a genuine second HP-restoring path. Transformation **coupons**
(`ClassificationTransformationCoupon`, cash item class 0530) route to
`ConsumeMorphCoupon`, not `ConsumeStandard`, and never reach `ApplyItemEffects`
— `routesToMorphCoupon` (`morph_coupon.go:71-73`) is checked before
`usesStandardConsumer`, and the deliberate exclusion is documented at
`processor.go:292-293`. `computeMorphCouponPlan` reads `cash.SpecTypeHp` and
heals it flat (`morph_coupon.go:46-48`).

That is exactly the reference's non-skill flat `hp` branch — `hpchange /= 2`
under zombify. Halving it is a faithful port, not a divergence, and the
prerequisite (a buff read in `atlas-consumables`) is being built for the
consumable path anyway.

**Decision: the morph-coupon HP heal is IN SCOPE and is halved on the same
rule.** This is a deliberate extension beyond the PRD's §7 service-impact table,
recorded here so the plan phase carries it. Excluding it would ship a known,
one-line-away inconsistency where a zombified player gets a half-heal from a
potion and a full heal from a transformation coupon. `morphCouponDeps` already
carries a `buff buff.Processor` field (`morph_coupon.go:86`), so the read seam
costs nothing structurally.

Both `ApplyItemEffects` call sites (`processor.go:495` in `ConsumeStandard`,
`processor.go:1062` in the NPC/saga-initiated apply) go through
`computeEffectPlan`, so a single change inside `ApplyItemEffects` covers both.
Everything else falls through to `ConsumeBare`, which restores no HP.

---

## 1. Architecture

Three changes, two services, one shared idea:

```
atlas-consumables                              atlas-channel
─────────────────                              ─────────────
character/buff                                 character/buff
  + GetByCharacterId  ──┐                        GetByCharacterId (exists) ──┐
  + Model / RestModel   │                        + IsZombified(bs) ──────────┤
  + IsZombified(bs) ────┤                                                    │
                        │                                                    │
consumable/processor.go │                      skill/handler/heal            │
  ApplyItemEffects ─────┤                        Apply ─────────────────────┘
    zombified := ...    │                          zombified := casterZombifiedFunc(...)
    computeEffectPlan(c, ci, zombified)            perTarget stays the magnitude
                        │                          delta := healDelta(perTarget, r, zombified)
consumable/morph_coupon.go                         XP block skipped when zombified
  consumeMorphCoupon ───┘
    computeMorphCouponPlan(ci, zombified)
```

The shared idea is that **zombify is a predicate over a buff list**, resolved to
a plain `bool` at the top of each effect application, and every downstream
computation stays a pure function of that bool. No processor is threaded into
`computeEffectPlan`; no buff read happens per-recipient.

### D1 — The predicate lives in each service's `character/buff` package as `IsZombified([]Model) bool`

This is the established idiom for "ask a question of a fetched buff list", and
both services already have a precedent to copy:

- `atlas-channel`: `buff.IsMount(m Model) bool` (`character/buff/model.go:11-19`)
  — walks `m.changes` comparing `c.Type()` against a
  `charconst.TemporaryStatType` cast to string.
- `atlas-monsters`: `buff.HasActiveGmHide(ctx, bs []Model) bool`
  (`character/buff/model.go:36+`) — a slice-level predicate in the same package.

`IsZombified` takes the **slice**, not a single buff, because the caller already
holds the drained list and the question is "does any of these". Signature:

```go
// IsZombified reports whether bs contains an unexpired buff carrying an
// UNDEAD stat change — the ZOMBIFY disease. See task-256.
func IsZombified(bs []Model) bool
```

Note the type asymmetry, which the plan must not paper over:

- `atlas-channel`'s `stat.Model.Type()` returns a **`string`**
  (`character/buff/stat/model.go:7-9`), so the comparison is
  `c.Type() == string(charconst.TemporaryStatTypeUndead)` — identical to
  `IsMount`.
- `atlas-consumables`' `stat.Model` is a plain struct whose `Type` field is
  already a **`character.TemporaryStatType`**
  (`character/buff/stat/model.go:3-8`), so the comparison is
  `c.Type == character.TemporaryStatTypeUndead` with no cast.

Neither is "wrong"; unifying them is a refactor this task does not own.

**Alternatives rejected.** A shared `libs/` helper: rejected — the two `Model`
types are per-service by design and a shared helper would need a shared buff
model, which is a much larger refactor than this task warrants
(CLAUDE.md: don't re-export type aliases across a service boundary to avoid a
straightforward duplicate). Inlining the loop at each call site: rejected —
FR-5's user story explicitly asks for one named predicate per service so a
future healing path adopts the rule rather than re-deriving it.

**Expiry.** `IsZombified` must skip `m.Expired()` buffs (FR-1). `atlas-channel`'s
`Model.Expired()` already exists (`model.go:48-53`, honouring `noExpiry`);
`atlas-consumables`' new `Model` gets the same shape.

### D2 — `atlas-consumables` gains a read-side buff client modeled on `atlas-monsters`

`atlas-consumables/character/buff` is today write-only (Kafka producers for
`Apply`/`Cancel`/`CancelByTypes`). It needs a REST read. The closest template is
`atlas-monsters/atlas.com/monsters/character/buff` — a four-file package
(`requests.go`, `rest.go`, `model.go`, `processor.go`) whose `GetByCharacterId`
is a single `DrainProvider` call.

New/changed files in `atlas-consumables`:

| File | Change |
|---|---|
| `character/buff/requests.go` | **new** — `RootUrlFor(ctx, "BUFFS")` + `characters/%d/buffs`, bare URL (not a `requests.Request`) because the list is paginated and drained. Mirrors `atlas-channel/character/buff/requests.go:14-28` verbatim in shape. |
| `character/buff/rest.go` | **new** — `RestModel` with `sourceId/level/duration/changes/createdAt/expiresAt/noExpiry` + `Extract`. Field names copied from `atlas-channel/character/buff/rest.go` because both decode the same atlas-buffs payload. |
| `character/buff/stat/rest.go` | **new** — `RestModel{Type string; Amount int32}` + `Extract` into the existing `stat.Model`, converting `Type` to `character.TemporaryStatType`. |
| `character/buff/model.go` | **new** — read-side `Model` with `Changes()`, `Expired()`, plus `IsZombified`. |
| `character/buff/processor.go` | `Processor` interface gains `GetByCharacterId(characterId uint32) ([]Model, error)`; `ProcessorImpl` implements it via `requests.DrainProvider[RestModel, Model](l, ctx)(url, 250, Extract, model.Filters[Model]())()`. |
| `character/buff/mock/processor.go` | gains `GetByCharacterIdFunc` + method, default `return nil, nil`. |

**404 normalization.** `atlas-channel`'s provider maps `requests.ErrNotFound` to
the empty slice, because atlas-buffs materializes a character's registry entry
lazily and answers 404 for a buffless character
(`character/buff/processor.go:44-56`). That is the single most likely response
for a player drinking a potion — most players carry no buffs at all — so the
consumables client **must** carry the same normalization. Without it, FR-3's
fail-open would fire on the common case and every drink would log a `Warn`.
`atlas-monsters`' simpler client omits this; copy `atlas-channel`'s, not
`atlas-monsters`', on this specific point.

**Rejected: reuse the existing write-side `stat.Model` for the wire type.** The
existing `stat.Model` is the right *domain* type and is reused; what is added is
a separate `stat.RestModel` for decoding, matching the channel split.

### D3 — `computeEffectPlan` takes `zombified bool` as its third parameter

```go
func computeEffectPlan(l logrus.FieldLogger, c character.Model, ci consumable3.Model, zombified bool) effectPlan
```

The read happens once, at the top of `ApplyItemEffects`, before the plan is
computed and therefore before the cure dispatch — which preserves the task-051
D3 ordering (cure, then HP/MP, then statups) untouched, as FR-9 requires. The
existing `bp := buff.NewProcessor(l, ctx)` at `processor.go:233` is already in
scope, so no new construction is needed:

```go
bp := buff.NewProcessor(l, ctx)
cp := character.NewProcessor(l, ctx)

zombified := false
if bs, err := bp.GetByCharacterId(characterId); err != nil {
    l.WithError(err).Warnf("Unable to read buffs for character [%d]; treating as not zombified.", characterId)
} else {
    zombified = buff.IsZombified(bs)
}

plan := computeEffectPlan(l, c, ci, zombified)
```

Fail-open is the `false` initializer plus the `Warn` — FR-3 exactly. No `error`
is returned from `ApplyItemEffects` (it returns nothing today and keeps that).

**Alternatives rejected.** Threading `zombified` in from the two callers:
rejected — duplicates the read and the fail-open branch at two sites for no
gain. Passing the `buff.Processor` into `computeEffectPlan`: rejected — it would
destroy the purity that NFR "Determinism in tests" and FR-10 both require, and
the function's own doc comment promises.

**Halving lives inside `computeEffectPlan`,** not at the dispatch loop, so the
zero-drop (FR-7) falls out of the existing `val > 0` structure rather than
needing a second filter in `ApplyItemEffects`:

```go
if val, ok := ci.GetSpec(consumable3.SpecTypeHP); ok && val > 0 {
    if amt := halveIfZombified(int16(val), zombified); amt > 0 {
        plan.hpChanges = append(plan.hpChanges, amt)
    }
}
if val, ok := ci.GetSpec(consumable3.SpecTypeHPRecovery); ok && val > 0 {
    pct := float64(val) / float64(100)
    if amt := halveIfZombified(int16(math.Floor(float64(c.MaxHp())*pct)), zombified); amt > 0 {
        plan.hpChanges = append(plan.hpChanges, amt)
    }
}
```

with

```go
// halveIfZombified halves an HP restoration amount when the recipient is
// zombified, using Go integer division (truncation toward zero) to match the
// reference's `hpchange /= 2` on an int. Order matters for hpR: the reference
// computes (int)(maxHp * hpR) / 2 — cast first, then integer-divide — which is
// what passing the already-floored int16 here reproduces.
func halveIfZombified(amount int16, zombified bool) int16 {
    if !zombified {
        return amount
    }
    return amount / 2
}
```

`mp`, `mpR`, cure types, statups, and duration are untouched (FR-8) — the MP
branches do not call the helper.

### D4 — `computeMorphCouponPlan` takes `zombified bool` on the same rule

```go
func computeMorphCouponPlan(ci cash.Model, zombified bool) morphCouponPlan
```

with `plan.hp = halveIfZombified(int16(val), zombified)`. The existing
`if plan.hp > 0` guard at `morph_coupon.go:115` already implements the zero-drop.

`consumeMorphCoupon` reads the predicate through `d.buff.GetByCharacterId` —
`morphCouponDeps.buff` is already a `buff.Processor`, and D2 puts
`GetByCharacterId` on that interface, so the seam already exists and the
package's existing `buff/mock` covers it. The read joins the existing
`model.NewGroup` fan-in **before** `ConsumeItem`? **No** — deliberately not.

The morph-coupon ordering contract (`morph_coupon.go:94-99`) is "every *fallible*
read before the commit, so a data failure returns the paid cash item to the
player". The zombify read is *not* fallible in that sense: FR-3 makes its failure
mode a logged `false`, never an abort. Putting it in the pre-commit group would
either (a) add a new way to bounce a paid coupon, which is the exact failure that
ordering exists to prevent, or (b) sit in the group ignoring its own error, which
misrepresents the group's contract. It goes **after** the commit, immediately
before `computeMorphCouponPlan`, alongside the other post-commit effect work.

The `plan.hp == 0 && len(plan.statups) == 0` warning at `morph_coupon.go:129`
must not fire for a coupon whose `hp` truncated to 0 under zombify — that
message blames the tenant's cash WZ ingest, which would be a false diagnosis.
Gate it on the pre-halving spec presence, or equivalently skip the warning when
`zombified` is true.

### D5 — Heal: one function seam, one pure delta helper

`skill/handler/heal` has no deps struct today; `Apply` constructs its
collaborators inline. Introducing a full `healDeps` (the `healdispel` pattern)
would be a wholesale restructuring of a working handler for one new read — more
churn than the change earns, and it would put every existing collaborator behind
a new indirection this task has no reason to touch.

Instead, follow the **`dispel` precedent**: a single package-level replaceable
`var` (`dispel.selectPartyMembersFunc`, `dispel.propRollFunc`,
`dispel.go:43-50`).

```go
// casterZombifiedFunc is the zombify-state seam tests can replace. Production
// drains the caster's buffs from atlas-buffs and applies buff.IsZombified.
// Per task-256 FR-3 a failed read resolves to false (not zombified): a
// buff-service fault must never turn a Cleric's Heal into party-wide damage.
var casterZombifiedFunc = func(l logrus.FieldLogger, ctx context.Context, characterId uint32) bool {
    bs, err := buff.NewProcessor(l, ctx).GetByCharacterId(characterId)
    if err != nil {
        l.WithError(err).Warnf("Heal: buff read failed for caster [%d]; treating as not zombified.", characterId)
        return false
    }
    return buff.IsZombified(bs)
}
```

Called **once**, before the recipient loop, next to the caster's effective-stats
fetch — never inside the loop (NFR "the read must not be issued per-recipient",
FR-11). A test replacing the seam with a counting closure pins the
once-per-cast contract (acceptance criterion 9).

The per-recipient arithmetic goes in `formula.go` as a pure function next to
`appliedPerRecipient`, so it is testable with no seam at all:

```go
// healDelta returns the ChangeHP delta for one recipient of a Heal cast.
//
// Non-zombified: the existing headroom clamp — never push Hp past MaxHp.
// Zombified: the reference negates the heal (StatEffect.calcHPChange), so the
// delta is damage. It is clamped to the recipient's CURRENT Hp so a cast never
// removes more HP than the recipient has; landing exactly on 0 kills them,
// which is intended (atlas-character emits DIED at adjusted == 0).
func healDelta(perTarget int16, r recipient, zombified bool) int16 {
    if !zombified {
        return appliedPerRecipient(perTarget, r)
    }
    if r.Hp == 0 {
        return 0
    }
    magnitude := int32(perTarget)
    if magnitude < 0 {
        magnitude = 0
    }
    if magnitude > int32(r.Hp) {
        magnitude = int32(r.Hp)
    }
    if -magnitude < math.MinInt16 {
        return math.MinInt16
    }
    return int16(-magnitude)
}
```

`appliedPerRecipient` is **not** modified — FR-13's hazard is avoided by never
passing it a negative value, and the non-zombified path stays byte-for-byte
identical (acceptance criterion 1).

On the `math.MinInt16` saturation: with `perTarget` an `int16` its magnitude
cannot exceed `32767`, and `-32767 > math.MinInt16`, so the branch is not
reachable from today's inputs. It stays as a defensive guard against a future
widening of `perTarget`, and the plan should note it is unreachable rather than
write a test that cannot construct the input.

`perTarget` itself keeps its sign (positive magnitude) throughout; the negation
happens per-recipient inside `healDelta`. This matters for two reasons: the
`Debugf` at `heal.go:167` keeps reporting a meaningful magnitude, and `HealXp`
— which takes `perTarget` — is never handed a negative.

The call site becomes:

```go
for _, r := range recipients {
    delta := healDelta(perTarget, r, zombified)
    if delta == 0 {
        continue
    }
    if hpErr := cp.ChangeHP(f, r.Id, delta); hpErr != nil { ... }
}

// XP gate: skip when sole recipient AND no undead targets in this cast.
// Also skipped entirely on a negated cast — HealXp derives from the applied
// heal, and a zombified cast heals nobody (task-256 FR-15).
if !zombified && !(len(recipients) == 1 && len(info.AffectedMobIds()) == 0) {
    ...
}
```

The `delta == 0` continue already covers "recipient at 0 HP is skipped"
(acceptance criterion 4) via `healDelta`'s `r.Hp == 0` branch.

`AnnounceSkillUse` / `AnnounceForeignSkillUse` are below the XP block and outside
any new condition — unchanged (FR-16).

### D6 — Undead *mobs* are untouched

The reference Heal also damages undead monsters, and `info.AffectedMobIds()` is
already read by the XP gate. Atlas's Heal handler applies no mob damage today,
zombified or not. That gap is pre-existing and out of scope; this design does not
widen it and does not close it.

---

## 2. Observability

Per NFR "Observability":

- `Debug` on a zombify-modified effect, at each of the three sites, including the
  character id and the pre/post amounts. In `computeEffectPlan` the logger is
  already a parameter, so the flat/ratio halving can log there; in `heal` the
  existing `Debugf` at `heal.go:167` gains the zombified flag.
- `Warn` on a failed buff read, with the character id and the error — one per
  site, inside the fail-open branch (D3, D5).

The `Debug` in `computeEffectPlan` is the one impurity in an otherwise pure
function, and it is the same impurity the function already carries (it logs a
morph warning at `processor.go:216`). It does not affect the return value, so
determinism is preserved.

---

## 3. Testing strategy

All new tests are plain Go unit tests. No Kafka, no REST, no clock.

**`atlas-consumables` — `computeEffectPlan` (pure, table-driven):**

| Case | Assert |
|---|---|
| `hp=300`, not zombified | `hpChanges == [300]` |
| `hp=300`, zombified | `hpChanges == [150]` |
| `hpR=25`, maxHp 1000, not zombified | `hpChanges == [250]` |
| `hpR=25`, maxHp 1000, zombified | `hpChanges == [125]` |
| `hp` and `hpR` both set, zombified | two entries, flat first, both halved |
| `hp=1`, zombified | `hpChanges` empty (truncates to 0, FR-7) |
| any of the above | `mpChanges`/`cureTypes`/`statups`/`duration` identical across both zombify states |

Every existing `processor_test.go` / `morph_coupon_test.go` case passes
`zombified: false` and its **expected values do not move** (NFR "Backward
compatibility"). The signature change is mechanical.

**`atlas-consumables` — fail-open:** exercise `ApplyItemEffects` with a
`buff/mock.ProcessorMock` whose `GetByCharacterIdFunc` returns an error; assert
full-value `hpChanges` via the `character/mock` `ChangeHPFunc` recorder.

**`atlas-consumables` — `IsZombified`:** unexpired UNDEAD → true; expired UNDEAD
→ false; a non-UNDEAD stat → false; empty slice → false; a multi-change buff
where UNDEAD is not the first change → true.

**`atlas-channel` — `healDelta` (pure, in `formula_test.go`):**

| Case | Assert |
|---|---|
| not zombified | identical to `appliedPerRecipient` for the same inputs (headroom clamp intact) |
| zombified, `Hp` > magnitude | `-magnitude` |
| zombified, `Hp` < magnitude | `-Hp` exactly |
| zombified, `Hp == magnitude` | `-Hp` (lands on 0 → the kill case) |
| zombified, `Hp == 0` | `0` (skipped) |

**`atlas-channel` — handler-level, via the `casterZombifiedFunc` seam:** a
counting closure asserts exactly one invocation per cast regardless of recipient
count, and that the argument is the caster id and not a recipient id
(acceptance criteria 9 and 11). Whether the existing handler test scaffolding
can drive `Apply` end-to-end offline is a plan-phase question; if it cannot, the
seam's call count is asserted by whatever harness the sibling handlers use, and
the per-recipient arithmetic is fully covered by `healDelta`'s pure tests
regardless.

**`atlas-channel` — `IsZombified`:** same five cases as consumables, against the
`Type() string` model.

**Not tested here:** `atlas-character`'s `DIED` emission. Acceptance criterion 5
is asserted at the `ChangeHP` call boundary (a delta of exactly `-Hp` was
dispatched); re-testing `atlas-character` is explicitly out of scope.

---

## 4. Risks

| Risk | Mitigation |
|---|---|
| A buff-service outage turns every Cleric Heal into party-wide damage | FR-3 fail-open, implemented as a `false` initializer in both seams, tested. |
| The 404-on-buffless-character case floods `Warn` logs on every potion drink | D2 copies `atlas-channel`'s `ErrNotFound` → empty-slice normalization, not `atlas-monsters`'. Tested. |
| `appliedPerRecipient` silently mangles a negative amount | It is never handed one — `healDelta` branches before calling it. |
| The signature change to `computeEffectPlan` drifts an existing expected value | The plan touches only the call sites and adds a `false` argument; NFR "Backward compatibility" is a review checkpoint. |
| Morph-coupon "bad WZ" warning misfires on a zombify-truncated heal | D4 gates the warning on pre-halving spec presence. |
| Per-cast latency | One extra single REST call per cast/drink, on paths that already issue several. Never per-recipient. |

---

## 5. Scope delta from the PRD

The PRD's §7 table lists `atlas-consumables` (`computeEffectPlan` +
`ApplyItemEffects`) and `atlas-channel` (`heal.go`). This design adds one item:

- **`atlas-consumables/consumable/morph_coupon.go`** — `computeMorphCouponPlan`
  gains `zombified bool` and halves `plan.hp`; `consumeMorphCoupon` reads the
  predicate post-commit. Rationale in OQ-5. It is the same reference branch, on
  the same service, behind the same prerequisite.

No other service changes. `atlas-buffs`, `atlas-character`, `libs/atlas-constants`,
and `atlas-ui` are all unchanged, as the PRD states. No deployment or ConfigMap
change (OQ-2). No FR-17 suppression code (OQ-1).
