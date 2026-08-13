# Backend Audit — atlas-packet (task-195-foreign-disease-mobskill)

- **Scope:** single commit `780040742` ("fix(atlas-packet): foreign disease blocks carry the mob-skill key, in the client's read order"), diffed against `origin/main`.
- **Service Path:** `libs/atlas-packet` (shared library, not a `services/atlas-<svc>` domain service — see Phase 2 note below)
- **Guidelines Source:** backend-dev-guidelines skill
- **Date:** 2026-08-06
- **Build:** PASS
- **Tests:** all packages under `libs/atlas-packet` pass (`go test -race ./...`); `model` package specifically: all pre-existing + 7 new tests pass
- **Overall:** PASS

## Files changed

- `libs/atlas-packet/model/character_temporary_stat.go` (+173/-31)
- `libs/atlas-packet/model/character_temporary_stat_test.go` (+191, new tests only)
- `docs/tasks/task-195-foreign-disease-mobskill/investigation.md` (new)
- `docs/TODO.md` (+7, follow-up notes)

## Phase 1 — Build & Test Results

```
$ cd libs/atlas-packet && go build ./...
(clean, no output)

$ go vet ./...
(clean, no output)

$ go test -race ./...
ok for every package, including github.com/Chronicle20/atlas/libs/atlas-packet/model
```

Targeted re-run, non-cached, of the new/touched tests:

```
$ go test -race -v -run 'TestCTSForeign|TestForeignReadOrder' ./model/...
--- PASS: TestCTSForeignMultiStatRoundTrip (pre-existing, all 12 pt.Variants sub-tests)
--- PASS: TestCTSForeignEmptyV95ClaimsNothing (pre-existing)
--- PASS: TestCTSForeignDiseaseCarriesMobSkillKey
--- PASS: TestCTSForeignPoisonCarriesValueThenMobSkillKey
--- PASS: TestCTSForeignSlowIsMaskOnly
--- PASS: TestCTSForeignOrderMatchesClientReadOrder
--- PASS: TestCTSForeignMultiDiseaseRoundTrip (all 12 pt.Variants sub-tests, incl. legacy GMS v28/v48)
--- PASS: TestForeignReadOrderCoversEveryValueCarryingStat (all 12 pt.Variants sub-tests)
--- PASS: TestForeignReadOrderNamesOnlyRealStats
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/model	1.044s
```

Also ran repo-root guards relevant to this diff (not strictly required by the audit brief, run for completeness since CLAUDE.md gates "done" claims on them):

```
$ bash tools/goroutine-guard.sh        # exit 0, clean
$ bash tools/skill-job-id-guard.sh     # "skill-job-id-guard: clean (14 divergent const(s) checked)"
```

## Phase 2 — Domain Discovery note

`libs/atlas-packet/model` is a shared packet-codec library package, not a `services/atlas-<svc>/.../internal/<domain>` DDD package. It has no `resource.go`, `processor.go`, `entity.go`, `builder.go`, `administrator.go`, or REST/JSON:API surface, and is not itself an external-HTTP-client package. Per the task instruction, the DOM-01..20, FILE-*, SUB-*, EXT-*, and SCAFFOLD-* checklists are **not applicable** — this file is pure wire-codec logic (`Encode`/`Decode`/`EncodeForeign`/`DecodeForeign` over `response.Writer`/`request.Reader`). Only the checklist items that map onto codec correctness are run below (DOM-21 shared-types, plus the ad hoc symmetry/ordering/test-quality items called out in the audit brief).

## Phase 3 — Mechanical/Correctness Checks

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-21 | No duplication of atlas-constants types | PASS | No new `type`/`const` declarations. All stat identifiers still route through the existing `character.TemporaryStatType` from `github.com/Chronicle20/atlas/libs/atlas-constants/character` (`character_temporary_stat.go:12`, used throughout `foreignReadOrder` at `character_temporary_stat.go:991-1034`). The 4 new funcs (`MobSkillReasonForeignValueWriter/Reader`, `ValueMobSkillReasonForeignValueWriter/Reader`) are typed via the pre-existing `ForeignValueWriter`/`ForeignValueReader` aliases (`character_temporary_stat.go:274`, `:344`) — no new codec type family introduced. |
| ENC/DEC-SYM-1 | Every new writer has an exact mirroring reader | PASS | `MobSkillReasonForeignValueWriter` writes `Int16(SourceId) + Int16(Level)` (`character_temporary_stat.go:324-329`); `MobSkillReasonForeignValueReader` reads `Int16→sourceId, Int16→level` in the same order (`:368-372`). `ValueMobSkillReasonForeignValueWriter` writes `Int16(Value) + Int16(SourceId) + Int16(Level)` (`:336-342`); `ValueMobSkillReasonForeignValueReader` reads the same three fields in the same order (`:374-379`). Byte-for-byte pinned by `TestCTSForeignDiseaseCarriesMobSkillKey` and `TestCTSForeignPoisonCarriesValueThenMobSkillKey` (`character_temporary_stat_test.go` new block) against real IDA-derived expected values (`0x7b,0x00,0x03,0x00` = mobSkill 123, level 3), not encoder-against-itself. |
| ENC/DEC-SYM-2 | Old `ValueSourceLevelForeignValueWriter/Reader` cleanly removed | PASS | `git show 780040742 -- libs/atlas-packet/model/character_temporary_stat.go` shows both old funcs deleted (was `:305-320`/`:327-333` pre-diff); repo-wide grep for `ValueSourceLevelForeignValue` after the change returns zero hits — no dangling references in `services/` callers (`services/atlas-channel/.../socket/writer/character_buff_give.go`, `character_buff_cancel.go`, `character_spawn.go` only touch `CharacterTemporaryStat` at the `Encode`/`EncodeForeign` method level, never the per-writer funcs directly). |
| ENC/DEC-SYM-3 | `EncodeForeign`/`DecodeForeign` iterate in the same order, incl. `legacyGmsMask` and `baseStatNames` branches | PASS | Both functions build their present-stat key set, apply `sortForeign(keys)` (`:1086` encode, `:1285` decode), and only then branch on `legacyGmsMask(t)` for the trailer shape (`:1092-1098` encode, `:1291-1294` decode) — the branch is strictly after ordering, so it cannot desync the two sides. Both skip `baseStatNames[...]` entries via the same package-level map before sorting (`:1081-1085` encode, `:1280-1284` decode) — same map, same predicate. `sortForeign`'s tie-break is `foreignRank` then `Shift()` (`:1046-1054`); since `Shift()` is unique per stat (assigned by a monotonically-incrementing counter in `buildCharacterTemporaryStatRegistry`, `:64/:74`), there is no possible tie and thus no `sort.Slice`-instability hazard. Set equality between encode's `m.stats`-derived keys and decode's `reg.inOrder`-filtered-by-decoded-mask keys rests on `EncodeMask`/`activeMask` claiming exactly one bit per present stat (task-190, unchanged by this diff) — verified pre-existing invariant, not re-litigated here. |
| ORDER-FALLBACK | `foreignRank` fallback (unnamed stats sort last, by shift) is safe | PASS | Safe *because* it is unreachable for any byte-carrying stat: `TestForeignReadOrderCoversEveryValueCarryingStat` (`character_temporary_stat_test.go`, new) iterates `pt.Variants` (12 variants: GMS v28/48/61/72/79/83/84/86/87/92/95 + JMS v185 — a superset of the 9 canonical matrix versions `gms_v48/61/72/79/83/84/87/95/jms_v185` listed in `docs/packets/PROCESS.md:38-39`) and asserts every registry stat not named in `foreignReadOrder` and not in `baseStatNames` writes **zero** bytes for that version's own registry-assigned writer. Confirmed passing (see Phase 1 run). The fallback tail can therefore only ever hold 0-byte NoOp entries, for which position is irrelevant. |
| ORDER-COVERAGE | Guard test's `pt.Variants` loop actually covers the versions that matter | PASS, with a scope note | `pt.Variants` (`libs/atlas-packet/test/context.go:18-41`) is a superset of the 9-version coverage matrix (`docs/packets/PROCESS.md:38-39`); it additionally covers GMS v28 and v48 (both `< 61`, i.e. both exercise the `legacyGmsMask` 8-byte-mask path) and GMS v86/v92 (extra, not in the official matrix). `template_gms_12_1.json` exists under `services/atlas-configurations/seed-data/templates/` but v12 is **not** in `matrix.VersionKeys` / the 9-version set and not in `pt.Variants` — its absence is a pre-existing gap in the shared test fixture list, not something this commit introduced or was asked to fix, and v12 is not part of the officially-tracked coverage matrix this codec is graded against. |
| INIT-HAZARD | `foreignReadOrderIndex` init-order / mutability | PASS | `var foreignReadOrderIndex = func() map[...]int {...}()` (`character_temporary_stat.go:1065-1071`) references `foreignReadOrder` (`:991`). Go initializes package-level vars in dependency order (spec: a var initializes after every var its initializer expression depends on), so declaration order in the source is irrelevant and there is no read-before-write hazard regardless of physical ordering. Repo-wide grep for `foreignReadOrderIndex` shows exactly one write site (the init expression) and reads only in `foreignRank` (`:1059`) and the test file (read-only) — no mutation after init, so concurrent reads from multiple goroutines (e.g. concurrent `EncodeForeign` calls for different characters) are safe. |
| EXPORT-CONVENTION | New writer/reader funcs match file's existing export convention | PASS | Every sibling `*ForeignValueWriter`/`*ForeignValueReader` func in the file is exported (`ValueAsByteForeignValueWriter`, `ByteForeignValueReader`, `LevelSourceForeignValueWriter`, `LevelSourceForeignValueReader`, etc., `:276-379`) because they're referenced from `NewCharacterTemporaryStatType` call sites across the whole `buildCharacterTemporaryStatRegistry` body. The 4 new funcs (`:324`, `:336`, `:368`, `:374`) are exported the same way, for the same reason. The new *helper* funcs `sortForeign`/`foreignRank` are unexported (`:1046`, `:1058`), consistent with the file's other unexported version-logic helpers (`legacyGmsMask`, `writeMask`, `isGmsV61`, `movementAffectingStatNames`). |
| TEST-QUALITY-1 | New tests pin real wire bytes, not just encoder-vs-itself | PASS | `TestCTSForeignDiseaseCarriesMobSkillKey` asserts the literal bytes `{0x7b,0x00,0x03,0x00}` for mob skill 123 level 3, and separately reconstructs the 32-bit LE composite and checks it equals `123 \| (3<<16)` — the exact value the client's `Decode4` sees per `CUser::ShowAffectedSkillAni`. `TestCTSForeignPoisonCarriesValueThenMobSkillKey` and `TestCTSForeignOrderMatchesClientReadOrder` are likewise literal-byte assertions, not round-trips. These are genuine regression pins, not tautologies. |
| TEST-QUALITY-2 | Round-trip tests' scope and honest limits | PASS, with a caveat noted for the record | `TestCTSForeignMultiDiseaseRoundTrip` and `pt.RoundTrip` (`libs/atlas-packet/test/roundtrip.go:22-34`) only prove `EncodeForeign`/`DecodeForeign` stay symmetric with each other and fully consume the buffer — they cannot, by construction, catch a case where both sides agree with each other but disagree with the real client (the documented "round-trip ≠ client-validated" failure mode). That gap is closed here by the literal-byte tests above, which *are* client-derived, so the two test families are complementary rather than redundant. `investigation.md` §5 states explicitly this task did not perform a live two-client verification and calls that out as the one open item — an honest, not silent, gap. |
| BEHAVIOR-SCOPE | Non-disease stats: was any behavior silently changed beyond what the commit describes? | PASS (disclosed, not silent) | `sortForeign` is applied unconditionally to **every** foreign-value-carrying stat present in a CTS body, not just diseases (`:1079-1086` builds `keys` from all non-base `m.stats`, disease or not, before calling `sortForeign`). This means any multi-stat foreign body mixing e.g. `Speed`+`Frozen`+`FinalCut` also gets reordered by this commit, not only disease combinations. The commit message's framing — "any two value-carrying stats present at once swapped payloads" — is written broadly enough to cover this (the disease examples given are illustrative, not exhaustive), and `investigation.md` §3's per-version verification table explicitly walks the *full* `DecodeForRemote` sequence (Speed, Combo, WeaponCharge, ShadowPartner, Morph, Barrier, DefenseAttack, …), not a disease-only subset. Not a hidden scope expansion — it is the correct, and disclosed, blast radius of "fix the order the client actually reads." |
| SCOPE-DEFERRAL | Pre-existing non-disease mismatches found during the sweep | PASS (correctly deferred, not silently dropped) | `investigation.md` §6 lists 4 independent pre-existing defects (ShadowPartner 16-bit truncation of a player skill id on v87+; BanMap writing 4 unread bytes on gms_v48; v95 Mechanic/DarkAura/BlueAura/YellowAura registry `NoOp` vs. client `Decode4` — currently latent/unreachable since atlas never sets those bits; gms_v61 having no `ReverseInput`/Confuse branch at all). All 4 are reproduced verbatim in `docs/TODO.md` (new `### atlas-packet` section, diff above) with the same evidence citations. None of these are prerequisites of the #1196 disease-render fix — they are unrelated latent defects turned up incidentally by the same code sweep — and none was silently swept under the rug: each has a checkbox and an evidentiary pointer back to `investigation.md §6`. This satisfies the project's "no deferring producible work" bar for *this* task's actual scope (foreign disease rendering), since fixing them is not required to close #1196 and each is independently actionable later. |
| REGISTRY-CONSISTENCY | Disease-writer reassignment covers exactly the 9 documented diseases, nothing more/less | PASS | Diff shows STUN (`:98`), SEAL (`:100`), DARKNESS (`:101`) → `MobSkillReasonForeignValueWriter/Reader`; POISON (`:99`) → `ValueMobSkillReasonForeignValueWriter/Reader`; WEAKEN (`:116`), CURSE (`:117`) → `MobSkillReasonForeignValueWriter/Reader`; SEDUCE (`:133`), CONFUSE (`:145`) → `MobSkillReasonForeignValueWriter/Reader`; SLOW (`:126`) explicitly left as `NoOpForeignValueWriter/Reader` with a 7-line justifying comment (`:118-125`). That's all 9 `Disease()==true` stat types in the registry (grep for `newAndIncDiseased` confirms exactly: Stun, Poison, Seal, Darkness, Weaken, Curse, Slow, Seduce, Confuse, plus `Undead` at `:259` which stays `NoOp` and is not claimed as a disease-render target anywhere in the investigation — consistent, since `Undead` is a base/two-state stat per `baseStatNames` at `:1118` and never reaches the foreign per-stat writer path at all). |

## Security Review

Not applicable — `libs/atlas-packet` is a wire-codec library with no authentication, authorization, or token-handling surface.

## Summary

### Blocking (must fix)

None.

### Non-Blocking (should fix / worth tracking)

- The 4 pre-existing non-disease foreign-codec mismatches disclosed in `investigation.md` §6 and `docs/TODO.md` (`ShadowPartner` skill-id truncation, `BanMap` extra bytes on gms_v48, v95 `Mechanic`/`DarkAura`/`BlueAura`/`YellowAura` NoOp-vs-Decode4 mismatch, gms_v61 missing `ReverseInput`/Confuse branch) are correctly out of scope for this fix but remain live defects for a future task.
- `template_gms_12_1.json` exists in `services/atlas-configurations/seed-data/templates/` but GMS v12 is absent from both the 9-version coverage matrix (`docs/packets/PROCESS.md`) and `pt.Variants` (`libs/atlas-packet/test/context.go`) — pre-existing gap, unrelated to this commit, noted for completeness.
- The two-client live-verification step named in the original issue's repro instructions was not performed (`investigation.md` §5, explicitly disclosed as the one outstanding check) — the literal-byte and IDA-cross-referenced tests substitute for it but are not a full live-client confirmation.
