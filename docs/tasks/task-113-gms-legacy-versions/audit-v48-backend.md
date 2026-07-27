# Backend Audit — task-113 GMS v48 pass (packet-codec + seed-template)

- **Branch:** task-113-gms-legacy-versions
- **Scope:** v48 pass only — commit range `36be000b65..HEAD` (parent of first v48 commit `b5f8d3767f` = v61 stage-F tip; HEAD `ced11d7037`). NOTE: the `452c1c9fb4..HEAD` range in the brief conflates the v61/v72/v79 passes; the correct v48-only base is `36be000b65`.
- **Date:** 2026-07-04
- **Build:** PASS (`go build ./...` exit 0)
- **Vet:** PASS (`go vet ./libs/atlas-packet/... ./tools/packet-audit/...` exit 0)
- **Tests:** PASS (`go test ./libs/atlas-packet/... ./tools/packet-audit/...` exit 0)
- **Overall:** PASS (no blocking findings)

## Notes on applicability
This is packet-codec + seed-template work. There is no domain package (no `model.go`/`processor.go`/`resource.go`), so the DOM-01..20 domain checklist and SUB-* / SEC-* auth checks do not apply. Relevant gates: the existing-version invariant, region-scoping, DOM-21 (constant reuse), and the socket-handler-validator seed rule.

## Critical invariant — existing-version (v61+) behavior

The v48-specific divergences are all region-scoped `GMS && < 61` (or `!MajorAtLeast(61)`) and are v48-only; v61/72/79/83/84/87/95/JMS output is preserved. Verified file:line:
- `model/avatar.go:78,84` — `>28` widened to `>=61`; new `else if >28` single-int arm is v48-only; v61+ keep the 3-int loop.
- `character/clientbound/spawn.go:80` `legacyV48 = GMS && <61`; `spawn.go:170` `<95` tightened to `>=61 && <95` — v48-only.
- `character/clientbound/status_message.go` `legacyV48 = GMS && <61` (IncEXP + meso arms) — v48-only.
- `model/attack_info.go:60` `legacyGmsNoRangedBulletCoords = GMS && <61` — the ONLY v48-pass attack change; the `<79`/`<72` gates are pre-existing (v72/v61 passes).
- `model/character_temporary_stat.go` `legacyGmsMask = GMS && <61` (8-byte int64 mask local+foreign) — region-scoped, v48-only.
- `party/clientbound/{disband,invite}.go`, `guild/clientbound/{info,operation}.go` — all `IsRegion("GMS") && <61`.
- `npc/serverbound/start_conversation.go:35` early-returns for non-GMS, then adds `|| ==48` to the existing `>=79 || ==61`; v72 stays excluded exactly as in base — no v61+/v72 change.
- `npc/clientbound/conversation.go` narrowed `GMS && !MajorAtLeast(83)` → `+ MajorAtLeast(61)` — only removes v48, v61/72/79 unchanged.
- `character/clientbound/list.go` `>=61` wrapper inside the existing `Region()=="GMS"` block; the `>87` (v95) gate is unchanged — v48-only exclusion.

### The invariant is NOT literally held — the v48 pass corrected THREE prior-pass false-passes (v61/v72). All IDA-verified + fixture-backed, none a regression, and all confined to branch-only versions (none of v48/61/72/79 exists on `main`):
1. **`messenger/clientbound/add.go:48` (flagged in brief)** — `GMS && <72` → `GMS && <=28`. v61 now correctly writes `channelId+pad`. IDA @0x6d144e; the prior gate keyed off a CMiniRoomBaseDlg arm (sub_5BF5AE), not the messenger dispatcher. Fixture `v61_test.go` explicitly re-pins ("CORRECTS the prior v61 fixture"). Correct.
2. **`buddy/serverbound/operation_add.go:47,58`** — base wrote `m.group` unconditionally; now `MajorVersion() > 61`, dropping the spurious group field for v61. IDA v48@0x4c6452 / v61@0x4e9c03 send name only; v72+@0x515575 append group. Fixture `operation_add_test.go:20` gates `hasGroup := v.MajorVersion > 61`. Correct — NOT in the brief.
3. **`reactor/serverbound/hit.go`** — base wrote `isSkill` + `skillId` unconditionally; now `isSkill` gated `MajorAtLeast(72)` and `skillId` gated `MajorAtLeast(79)`. This drops isSkill+skillId for v61 AND drops skillId for v72. IDA v48@0x5a5d1a / v61@0x633ac7 (direct oid→dwHitOption), v72@0x6928bc (isSkill), v79@0x6b8077 (skillId). Fixtures updated. Correct — NOT in the brief.

**Invariant verdict:** intent preserved (no accidental v61+ breakage; nothing shipped is affected), but the literal "no existing-version behavior changed" is false — the v48 pass deliberately re-cut the v61/v72 wire for messenger, buddy, and reactor. Two of these three (buddy, reactor) were not called out in the brief; each is IDA-cited and fixture-backed, so I grade them PASS-with-note, not FAIL. They warrant a second reviewer glance because they retroactively invalidate the v61/v72 passes' original verification of those three packets.

## Minor findings (non-blocking)
- **Region-less gate (cosmetic, memory M1):** `buddy/serverbound/operation_add.go:47,58` use bare `t.MajorVersion() > 61` with no `Region()` guard. Correct only because JMS=185 (>61, wants group) and GMS-legacy ≤61; a hypothetical GMS v62–71 would misroute. Consistent with the M1 "region-less = cosmetic" disposition. All other v48 gates are region-scoped. `field/clientbound/effect_weather.go:43` relies on `else`-after-`JMS` to mean GMS — functionally correct but implicit.
- **v28 folding (documented controller decision):** the `< 61` boundaries also capture v28 (no IDB). Defensible: no v28 tenant exists and the seed set has none, so the folded-in legacy wire is never exercised. Risk is latent — if a v28 tenant is ever seeded, avatar single-pet (8-byte long vs 4-byte int), messenger `<=28`, and the `<61` bodies would need re-verification. Note, not a blocker.

## Other checks
- **DOM-21:** no new `type`/`const` reinventing atlas-constants (world/channel/map/job ids) in the v48 diff. PASS.
- **Seed template** `template_gms_48_1.json`: valid JSON; region GMS / major 48 / minor 1 / usesPin false; 75 handlers, 54 writers; 0 handlers with missing/empty validator (satisfies the silent-drop seed rule). PASS.
- **packet-audit** (`internal/matrix/model.go`, `cmd/fnamedoc.go`, `cmd/run.go`): `gms_v48` added to VersionKeys/shortLabels/fnamedocOrder; guard `version_lists_test.go` passes (green). PASS.
- **SEC:** N/A — pure wire codecs, no JWT/token/redirect/secret surface. Login-flow "serverbound" changes are packet decoders, no auth logic.
