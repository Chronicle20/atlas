# Task-163 Context — Priest Dispel Party Debuff Cure

Companion to `plan.md`. Key files, decisions, and dependencies for implementers.

Revised 2026-08-07 — current with `main` @ `e0f5bd01d`.

## Worktree / Modules

- Worktree: `.worktrees/task-163-priest-dispel-party/`, branch
  `task-163-priest-dispel-party`. Never edit the main checkout.
- Primary changed module: `services/atlas-channel/atlas.com/channel` (module
  `atlas-channel`). Run all `go` commands from that directory.
- Conditionally changed module: `libs/atlas-packet` — **only** if the per-version
  decode audit (plan Task 3) finds a divergence.
- `services/atlas-buffs/` and `services/atlas-monsters/` must end the task with
  zero diffs.

## What Already Landed (do not rebuild)

The v1 docs were written before task-156 and task-187 merged. Both changed this
task's foundations:

| Thing | Where it is now |
|---|---|
| `CANCEL_BY_TYPES` command type + body | `atlas-channel/kafka/message/buff/kafka.go:17,64` |
| `CancelByTypesCommandProvider(f, characterId, types []string)` | `atlas-channel/character/buff/producer.go:89` |
| `Processor.CancelByTypes(f, characterId, types []string) error` | `atlas-channel/character/buff/processor.go:25,82` — **uncurried, `[]string`, returns `error`** |
| Wire-contract test for the above | `atlas-channel/character/buff/producer_test.go:19` |
| Identity-keyed skill handler registry | `atlas-channel/skill/handler/registry.go` — `map[skill2.Identity]Handler` |
| Wire→Identity resolution at dispatch | `skill/handler/common.go` `UseSkill` — `constants.For(t.Region(), t.MajorVersion(), t.MinorVersion()).Skill.Resolve(...)` then `Lookup(castId)` |
| Mob→character disease infliction | `atlas-monsters` `monster/disease.go` `applyDiseaseCommandProvider`, driven by `SkillTypeToDiseaseName` (`libs/atlas-constants/monster/skill.go:112`) at `monster/processor.go:1239-1252` |
| SuperGM Heal+Dispel (11-type purge) | `atlas-channel/skill/handler/healdispel/healdispel.go` (task-156) |

## Key Files

| File | Role |
|---|---|
| `services/atlas-channel/atlas.com/channel/skill/handler/dispel/` | New per-skill handler subpackage — the only new production code. |
| `services/atlas-channel/atlas.com/channel/skill/handler/registrations/registrations.go` | Blank-import list driving handler `init()`. Currently **seven** imports; the edit is additive (eighth). |
| `services/atlas-channel/atlas.com/channel/skill/handler/registry.go` | `Register(id skill2.Identity, h Handler)` / `Lookup(id skill2.Identity)`. Identity-keyed — a wire-keyed registration will not compile. |
| `services/atlas-channel/atlas.com/channel/skill/handler/recipients.go` | `SelectPartyMembersInMap` (line 188) — map-wide bitmap selector. Filters offline / other-map / no-session / dead; returns nil for `bitmap == 0 \|\| bitmap >= 128` (line 236); MSB-first slot decode documented at 257-259. Consumed, not modified. |
| `services/atlas-channel/atlas.com/channel/skill/handler/common.go` | UNTOUCHED. Mob half of Dispel (`applyToMobs`, `isCrashOrDispel`, `dispelSkillClass` — all Identity-keyed) + the per-skill dispatcher that invokes the new handler. Its unexported `propRollFunc` is the semantics the dispel package mirrors. |
| `services/atlas-channel/atlas.com/channel/skill/handler/healdispel/healdispel.go` | Closest precedent: dep-struct seams, `[]string` disease set at package level, per-recipient log-and-continue, `channelhandler.Register(skill2.SuperGmHealDispel, Apply)`. |
| `services/atlas-channel/atlas.com/channel/character/buff/processor.go` | `CancelByTypes` — consumed, NOT modified. |
| `services/atlas-channel/atlas.com/channel/socket/handler/character_skill_use.go` | `CharacterUseSkillHandleFunc` — decodes `SkillUsageInfo`, gates on skill-book/HP/cooldown, fetches the effect from `atlas-data`, calls `UseSkill`, then emits the self + foreign skill-use announces. A failed `GetEffect` aborts the cast **before** any handler runs. |
| `libs/atlas-packet/model/skill_usage_info.go` | The per-version risk. Three hard-coded, version-blind wire-id gate lists, each with `// TODO this is not all inclusive`. `2311001` is in **all three**. |
| `libs/atlas-constants/skill/identities_gen.go` | `PriestDispel Identity = 2311001` (line 177). |
| `libs/atlas-constants/skill/version_*_gen.go` | Per-version identity↔wire maps. `PriestDispel: 2311001` in all 11. |
| `libs/atlas-constants/constants/registry_gen.go` | The 11 provisioned `(region, major, minor)` version sets. |
| `deploy/k8s/base/versions.json` | The authoritative supported-version list (11). |
| `docs/packets/audits/STATUS.md` | Coverage matrix — only **9** columns (no gms_12, no gms_92). Serverbound `SPECIAL_MOVE` (`CUserLocal::SendSkillUseRequest`) is ❌ on every column. |
| `services/atlas-configurations/seed-data/templates/template_gms_12_1.json` | 24 handlers, no `CharacterUseSkillHandle` — evidence that gms_12 is unreachable. |

## Precedents to Follow

- **Handler shape + seams + tests:** `skill/handler/mysticdoor/` and
  `skill/handler/healdispel/` — package-level func-var seams, `t.Cleanup`
  restore, `Apply(l)(ctx)(wp, f, characterId, info, e)` signature.
- **Per-recipient error handling:** `healdispel.applyHealDispel` — log and
  continue, never abort the cast.
- **`[]string` stat set at package level:** `healdispel.diseaseTypes`.
- **Registry test:** `skill/handler/registry_test.go`.
- **Byte-layout regression test for a decode fix:** task-111 (Bishop
  Resurrection 2321006).

## Decisions Locked in Design (do not re-litigate)

1. **Approach A**: per-skill handler subpackage consuming the **existing**
   `CancelByTypes`. Not inline in `UseSkill`, not an atlas-buffs `DISPEL`
   command, and **not** a reshape of the shared processor signature (Approach D,
   rejected — `healdispel` and the Homing Beacon path depend on it).
2. **Cure set is exactly six**: CURSE, DARKNESS, POISON, SEAL, WEAKEN, SLOW.
   STUN / SEDUCE / CONFUSE / UNDEAD / STOP_PORTION / STOP_MOTION / FEAR excluded
   — `atlas-monsters` can inflict all of them, but they are cure-all semantics.
3. **Registration is by Identity** (`skill2.PriestDispel`), never by wire id.
4. **Map-wide recipients, no rectangle**: caster + `SelectPartyMembersInMap`.
   The WZ lt/rb rect governs only the mob half.
5. **Prop roll per recipient** (caster included), mirrored `propRollFunc` (the
   parent seam is unexported; healdispel/heal precedent). `e.Prop()` is
   pre-normalized 0.0–1.0 — no `/100`.
6. **Handler always returns nil** — failures logged, cast never aborts.
7. **Stat set stays handler-local**; the shared processor takes the set as a
   parameter, so `healdispel`'s wider set and Dispel's six coexist.
8. **The handler is version-blind.** No `constants`, no `tenant`, no
   `skill2.Is` on `info.SkillId()` inside the package. All version sensitivity
   is resolved upstream (registry, atlas-data, decoder).

## Supported Versions (11)

`gms_12_1, gms_48_1, gms_61_1, gms_72_1, gms_79_1, gms_83_1, gms_84_1,
gms_87_1, gms_92_1, gms_95_1, jms_185_1` — from `deploy/k8s/base/versions.json`,
mirrored in `constants/registry_gen.go` and the 11 seed templates.

- `PriestDispel → 2311001` on **all 11**; not in task-187's `divergences.csv`,
  so `tools/skill-job-id-guard.sh` does not fire on it.
- `CharacterUseSkillHandle` routed on **10 of 11** — `gms_12_1` has it on none,
  so Dispel (and every other per-skill handler) is unreachable there.
- The packet matrix covers only **9** of the 11 — `gms_12` and `gms_92` have no
  audit dir and no export, so gms_92 findings go in the task folder only.

## The Version Trap (read before implementing)

`SkillUsageInfo.Decode` reads the serverbound skill-use packet with three
hard-coded, **version-blind** raw-wire-id lists. `2311001` is in all three, so
the assumed layout is:

```
updateTime(4) skillId(4) slv(1) castX(2) castY(2) bitmap(1) delay(2) mobCount(1) mobIds(4×N) delay(2)
```

with `delay` read twice. The client writes each group **conditionally**, by the
skill's client-side category. A membership disagreement on any version shifts
every subsequent field: the bitmap decodes to 0, `SelectPartyMembersInMap`'s
`bitmap == 0` gate returns nil, and Dispel silently degrades to caster-only —
**no error, no log**. The lists and the slot-order decode both come from **v83
only** (`CUserLocal::SendSkillUseRequest` @0x96d399,
`is_antirepeat_buff_skill` @0x96d6ca, `CUserLocal::FindParty` @0x96db3f), and
`SPECIAL_MOVE` is ❌ on every matrix column — no version has pinned evidence.

**Field signature of the failure:** the FR-8 summary line reads
`bitmap=0, recipients_selected=1` on a version where the caster demonstrably had
a party. That is why the summary log ships before the audit.

Prior occurrences: task-111 (Bishop Resurrection 2321006), task-155 / PR#1136
(Buccaneer Time Leap 5121010).

## Dependencies / Coordination

- **atlas-buffs (existing, verified):** consumer `handleCancelByTypes`
  (`kafka/consumer/character/consumer.go:93`) → `CancelByStatTypes` → cancels
  intersecting buffs → `EXPIRED` status events → atlas-channel buff status
  consumer → client buff-cancel packets. Verified in acceptance, not
  reimplemented.
- **atlas-monsters (existing, verified):** the debuff source. Mob skill types
  120/121/122/124/125/126 map to SEAL / DARKNESS / WEAKEN / CURSE / POISON /
  SLOW — exactly Dispel's six, so every one is curable in live play. Types
  123/128/132/133/134/135/136 (STUN / SEDUCE / CONFUSE / UNDEAD /
  STOP_PORTION / STOP_MOTION / FEAR) are inflictable but **not** dispellable —
  use one of these for the negative acceptance case.
- **task-156 (gm-hide-heal-dispel): LANDED.** It owns the shared `CancelByTypes`
  producer and the 11-type `healdispel` purge set. Nothing left to coordinate.

## Verification Gates (CLAUDE.md + design §8)

1. `go test -race ./...`, `go vet ./...`, `go build ./...` clean in
   `services/atlas-channel/atlas.com/channel` — and in `libs/atlas-packet` if
   plan Task 4 ran.
2. `docker buildx bake atlas-channel` from the worktree root.
3. `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh`,
   `tools/skill-job-id-guard.sh`, `tools/lint.sh --check` clean from the repo
   root.
4. `git diff --stat main...HEAD -- services/atlas-buffs/ services/atlas-monsters/ .../skill/handler/common.go .../character/buff/ .../kafka/message/buff/` EMPTY.
5. `registrations.go` contains **8** handler imports (none dropped).
6. `version-findings.md` has a row for all 11 versions — evidence or an explicit
   "unverified — reason". No blanks, no omissions.
7. Code review (`superpowers:requesting-code-review`) before any PR.

## Gotchas

- Test files that override the unexported seams must be internal
  (`package dispel`), not `package dispel_test`.
- `channelhandler.Register` takes `skill2.Identity`. `skill2.PriestDispel` is
  the Identity; `skill2.PriestDispelId` is the wire id — they are both `2311001`
  numerically but are **different types**, and passing the wrong one is a
  compile error (rely on that).
- `SkillUsageInfo.AffectedPartyMemberBitmap()` has a pointer receiver; calling
  it on an addressable value is fine (heal/healdispel do the same).
- The bitmap is MSB-first by party slot (slot i → bit 5-i) — but the dispel
  handler never decodes it; it passes the raw byte to the selector.
- `effect.Model{}` zero value has Prop 0 → the real roll always fails; tests
  needing passes must build via `effect.Extract(effect.RestModel{Prop: 1.0})`.
- `CancelByTypes` returns `error` and is **not** curried — the v1 design's
  `model.Operator[uint32]` shape does not exist. Call it once per recipient.
- Do not add `*_testhelpers.go` files (project rule); the harness lives inside
  the `_test.go` file.
- `tools/lint.sh --check` false-fails without nvm22 on PATH (frontend leg) and
  under cross-worktree golangci-lint lock contention — run `tools/lint.sh` (fix
  mode) before committing.
- Do not cite prop values from memory. Re-read them per version from
  `atlas-data`; the v83 34→100 figures are a reference to verify, not a source.
