# Scoped re-review — final fix wave (23d12c455..e1f509056)

Scope: the fix diff only (`.superpowers/sdd/plan/review-23d12c455..e1f509056.diff`,
commit `e1f509056`). The four plan tasks were reviewed and approved at
`23d12c455` and are not re-reviewed here.

## Findings

### 1. DOM-01 / FILE-05 — builder move — ADDRESSED

`services/atlas-drops/atlas.com/drops/party/model.go` (lines removed) and the
new `services/atlas-drops/atlas.com/drops/party/builder.go` (lines added):
diffed byte-for-byte identical — `modelBuilder`, `memberBuilder`,
`NewBuilder`, `NewMemberBuilder`, all setters, and `Build()` bodies are
unchanged, only relocated. `model.go` retains `Model`/`MemberModel` and their
accessor methods (confirmed via `git show HEAD:.../party/model.go`). No other
file in the package references the builders by a changed path (same package,
so no import changes needed).

### 2. EXT-01 — SetToOneReferenceID stubs — ADDRESSED

`services/atlas-drops/atlas.com/drops/party/rest.go:67-69` (RestModel) and
`:156-158` (MemberRestModel) each add
`func (r *X) SetToOneReferenceID(_, _ string) error { return nil }`, matching
the required boilerplate in `libs/atlas-rest/CLAUDE.md:23-25` verbatim
(pointer receiver, blank params, nil return). Both types already had
`SetToManyReferenceIDs`, so the earlier finding (missing `SetToOneReferenceID`
only) is now fully closed.

### 3. EXT-02 — httptest/fixture test drives real unmarshal path — ADDRESSED

`services/atlas-drops/atlas.com/drops/party/rest_test.go:1325-1390`,
`TestExtract_JSONAPIFixture`. Verified field names against
`services/atlas-parties/atlas.com/parties/party/rest.go:95-103`
(`MemberRestModel` json tags `worldId`, `channelId`, `mapId`, `instance`,
`online`) — the fixture's `attributes` block uses exactly these keys, and the
outer `"type": "parties"` / member `"type": "members"` match
`RestModel.GetName()` (`rest.go:20-22`, returns `"parties"`) and
`MemberRestModel.GetName()` (returns `"members"`). Import is
`github.com/jtumidanski/api2go/jsonapi` (confirmed no `manyminds` import
remains anywhere in the package — `grep -n manyminds` on both `rest.go` and
`rest_test.go` returned nothing), the same fork `rest.go` itself imports.
Test asserts `Extract(rm)` produces a non-zero `Model` with the member's id,
online flag, and field values populated from the decoded relationship —
this only passes if both `SetToOneReferenceID` stubs exist and are wired
correctly, so it genuinely exercises the EXT-01 fix through the real
unmarshal path rather than around it.

### 4. DOM-28 — degrade.Observe in resolveMembers — ADDRESSED

`services/atlas-drops/atlas.com/drops/drop/processor.go:192-199`.
`resolveMembers` still returns `nil` on the `p.pp.GetByMemberId` error branch
with no new branch — only the log call became
`degrade.Observe(p.l, "drops.meso_split.party", characterId, err)`. Compared
to the reference shape in
`services/atlas-buffs/atlas.com/buffs/character/processor.go:445`
(`degrade.Observe(p.l, "buffs.periodic.character_hp", characterId, err)`),
the call is structurally identical: static low-cardinality component string,
entity id, error. `degrade.Observe`'s implementation
(`libs/atlas-rest/degrade/degrade.go:24-27`) logs at Warn and increments
`atlas_enrichment_degraded_total{component}` — pure observability, no control
flow. The test case `"party lookup error awards full amount to picker"`
(processor_test.go, ex-`TestProcessor_Reserve_PartyLookupError_...`, now a
case inside `TestProcessor_Reserve_MesoSplit`) still asserts exactly 1 award
to character 12345 for the full 100 amount with `Picker == true` — assertion
body byte-identical to the pre-fix version, confirming the outcome is
unchanged.

### 5. DOM-20 — table-driven conversions — ADDRESSED, all cases preserved

Compared each file's pre-fix `func Test...` list (at `23d12c455`) against the
post-fix case names (at `e1f509056`):

- `party/rest_test.go`: pre `TestExtract` (2-member body) + `TestExtract_NoMembers`
  → post `TestExtract` with cases `"extracts a party with members"` and
  `"no members"`. 2 cases, both assertion bodies unchanged. Count preserved.
- `drop/processor_test.go`: pre 6 funcs
  (`TestProcessor_Reserve_MesoDrop_SplitsAmongCoLocatedPartyMembers`,
  `..._ExcludesMembersNotCoLocated`, `..._ItemDrop_MakesNoPartyLookup`,
  `..._PartyLookupError_AwardsFullAmountToPicker`,
  `..._FailedReservation_EmitsNoAwards`,
  `..._ZeroShareSuppressesNonPickersOnly`) → post
  `TestProcessor_Reserve_MesoSplit` with 6 named cases, one per original func,
  bodies unchanged apart from indentation. `TestProcessor_Reserve_FailedReservation_BuffersFailureMessage`
  and `TestProcessor_Reserve_PartyMemberCanReserve` (untouched, outside this
  finding's scope) remain standalone funcs, confirmed not folded in or
  dropped. Count preserved (6/6).
- `character/meso_award_test.go`: pre 6 funcs → post `TestAwardPickedUpMeso`
  with 6 named cases, one per original func name (kebab-cased), bodies
  byte-identical. Count preserved (6/6).
- `kafka/consumer/drop/consumer_test.go`: pre 1 func
  (`TestHandleMesoAwarded_IgnoresNonMesoAwardedEvents`) → post
  `TestHandleMesoAwarded` with 1 case `"ignores non meso awarded events"`; the
  FR-15 guard comment is preserved (moved to sit above the case literal).
  Count preserved (1/1).

No case was dropped, renamed away from its meaning, or merged with another;
no assertion value changed.

### 6. Doc drift — domain.md — ADDRESSED

`services/atlas-character/docs/domain.md:124` now reads `AwardPickedUpMeso |
Credit a picked-up meso share to a character and, for the picker, emit the
pick-up completion command`. Verified the named function exists with that
exact signature at
`services/atlas-character/atlas.com/character/character/processor.go:941`:
`func (p *ProcessorImpl) AwardPickedUpMeso(transactionId uuid.UUID, f
field.Model, characterId uint32, dropId uint32, meso uint32, picker bool)
error`. `AttemptMesoPickUp` no longer appears in the table.

## Constraints check

- No `go.mod`/`go.sum` changes anywhere in the fix diff (`git diff
  23d12c455..e1f509056 -- '**/go.mod' '**/go.sum'` is empty).
- No new `*_testhelpers.go` files (`git diff --stat ... | grep testhelpers`
  empty).
- No cross-service imports introduced; `degrade` and the builder move are
  both intra-service/intra-package.
- No `// TODO` or stub added beyond the two EXT-01-mandated no-ops (searched
  the diff for `TODO` — none present).
- Immutable models / value receivers preserved: `Model`/`MemberModel`
  accessor methods untouched; only the *builder* types (already
  pointer-receiver, mutation-then-return-self) moved, which is the
  established pattern elsewhere in the repo.

## Not evaluable

None. The fix diff is fully self-contained (party package, drop processor,
character tests, one doc line) and every referenced contract
(`libs/atlas-rest/CLAUDE.md`, `atlas-parties`' `rest.go` wire shape,
`atlas-buffs`' `degrade.Observe` call shape, `atlas-character`'s
`processor.go` signature) was read directly to confirm the finding closure.

## Verdict

APPROVED. All six findings ADDRESSED, no new breakage found in the fix diff.
