# Guild Dispatcher Family — Design

Task: task-103-guild-dispatcher-family
Status: Draft for review
Created: 2026-06-18
Companion to: `prd.md` (approved), `docs/packets/DISPATCHER_FAMILY.md` (governing pattern)

---

## 1. Problem & Current State (grounded)

The guild family is a mode-prefix dispatcher: one clientbound opcode
(`GUILD_OPERATION` → `CWvsContext::OnGuildResult`) whose leading byte selects one of
N arms, plus a BBS sub-dispatcher (`GUILD_BBS_PACKET` → `CUIGuildBBS::OnGuildBBSPacket`),
the foreign name/emblem broadcasts (`CUserRemote::OnGuild{Name,Mark}Changed`), and the
serverbound `GUILD_OPERATION` / BBS operations.

It is ~60% migrated. The codecs largely exist and v95/jms are byte-verified, but the
family still carries the task-096 footguns that keep it on
`docs/packets/dispatcher-lint-baseline.yaml` alongside `party` and `buddy`. Verified
findings from the current tree:

| Concern | Evidence (file:line) | Invariant |
|---|---|---|
| `GuildErrorBody(code string)` / `GuildErrorBody2(code, target string)` — caller-supplied operation selector | `libs/atlas-packet/guild/operation_body.go:64,70` | AP-4 / INV-3 |
| `ErrorMessage` struct fronts ~14 mode-only error arms through one `struct{mode byte}` | `libs/atlas-packet/guild/clientbound/operation.go:59-84`; run.go `#ErrorMessage` `cmd/run.go:1384` | AP-1 / AP-7 catch-all |
| `ErrorMessageWithTarget` fronts the 3 target-bearing error arms through one struct | `operation.go:89-117`; run.go `#ErrorMessageWithTarget` `cmd/run.go:1388` | AP-1 catch-all |
| Bare `CWvsContext::OnGuildResult` returns `RequestAgreement` as a **phantom representative** | `tools/packet-audit/cmd/run.go:1369-1372` ("OP-FAMILY-guild-clientbound: deferred to _pending.md") | AP-5 / INV-4 |
| `CWvsContext::OnGuildBBSPacket` + `CUIFadeYesNo::OnButtonClicked` top-level catch-all roots, deferred to `_pending.md` | `cmd/run.go:1462-1464,1480-1482` | AP-5 / INV-4 |
| Serverbound `AgreementResponse` wire mismatch (extra `Encode4 unk`) vs `CField::SendCreateGuildAgreeMsg` | `cmd/run.go:1487-1489`; `guild/serverbound/operation_agreement_response.go` | codec bug (FR-16) |
| `GuildInfoBody` bypasses `WithResolvedCode` and hard-codes mode 0x1A inside `Info.Encode` | `operation_body.go:146`; `clientbound/info.go:70` | AP-2 (latent) |
| BBS `BBSThreadList`/`BBSThread` hard-code the mode byte (0x06/0x07) inside `Encode` | `clientbound/bbs.go:54,167` | AP-2 (latent) |
| v84 ❌ across guild serverbound + BBS rows (task-100 reshift carryover) | `docs/packets/audits/STATUS.md` guild rows | FR-14 / coverage |

The operations key consts already exist (35 keys, `operation_body.go:13-47`) and the
per-version `operations` table is already populated for guild in the seed templates
(e.g. `template_gms_83_1.json:1579-1614`). There is **no**
`docs/packets/dispatchers/guild.yaml` source-of-truth file yet.

**Governing rule (from `DISPATCHER_FAMILY.md`):** `matrix ✅` means codec byte-correct,
nothing more. Discrete-per-mode shape, config-driven resolution, footgun-free APIs,
usability, live-config wiring, and honest grounding are separate requirements proven by
the gates in §8, not by a green cell.

## 2. Goal / Definition of Done

Migrate guild to the canonical discrete-per-mode pattern, drive every supported arm to
✅ across all five versions (`gms_v83`, `gms_v84`, `gms_v87`, `gms_v95`, `jms_v185`),
remove `CWvsContext::OnGuildResult` from the dispatcher-lint baseline, and patch live
tenant config so the completed family is usable. All gates in §8 exit 0.

## 3. Architecture — the canonical pattern applied to guild

We copy the migrated exemplars `libs/atlas-packet/field/clientbound/mts_operation.go`
(discrete per-mode structs in one consolidated file) +
`field/mts_operation_body.go` (per-mode body functions) and the `field_effect` family.
The shape for guild:

```
libs/atlas-packet/guild/
├── operation_body.go            # one fixed-key body func per supported clientbound arm; NO selector params
├── bbs_body.go                  # per-mode BBS body funcs (mode resolved, not literal)   [new or folded]
├── clientbound/
│   ├── operation.go             # ONE discrete struct per supported OnGuildResult arm (incl. each error arm)
│   ├── bbs.go                   # discrete BBSThreadList / BBSThread; mode resolved via body func
│   ├── info.go                  # Info arm routed through a body func (mode resolved, not 0x1A literal)
│   ├── name_changed_foreign.go  # standalone (unchanged)
│   └── emblem_changed_foreign.go# standalone (unchanged)
└── serverbound/
    ├── operation*.go            # AgreementResponse wire fix; missing v83/v87 sub-mode codecs added
    └── bbs_*.go                 # unchanged unless a version gap surfaces
docs/packets/dispatchers/
├── guild.yaml                   # NEW — per-version mode table, source of truth for OnGuildResult keys
└── guild_bbs.yaml               # NEW — per-version mode table for the BBS sub-dispatcher
tools/packet-audit/cmd/run.go    # one #-entry per supported mode; phantom roots + catch-alls removed
```

Data flow per clientbound arm (unchanged from the exemplar):

```
feature/handler → Guild<Arm>Body(arm data)                       (operation_body.go)
   → WithResolvedCode("operations", <FIXED_KEY const>,           resolves mode byte
        func(mode byte) → clientbound.New<Arm>(mode, arm data))   from tenant operations table
   → struct.Encode writes mode byte + full arm body              (clientbound/operation.go)
```

## 4. Key Design Decisions (alternatives + tradeoffs)

### D1 — Split the `ErrorMessage` / `ErrorMessageWithTarget` catch-alls into one discrete struct per mode

The current `ErrorMessage` struct serves ~14 modes and `ErrorMessageWithTarget` serves 3
(`operation.go:59-117`). `DISPATCHER_FAMILY.md` is explicit: "Bodyless notice/error arms
are still their own discrete struct (`struct { mode byte }`) — discrete means *discrete*,
even when two arms share a wire shape" (AP-1, AP-8).

- **Chosen:** one discrete struct per supported error/notice arm, all in the single
  consolidated `clientbound/operation.go`. Mode-only arms become
  `type Guild<Name> struct { mode byte }`; target-bearing arms become
  `struct { mode byte; target string }`. The exact set is the IDA-enumerated
  `OnGuildResult` switch (§4.7), not the full 35-key operations table — keys present in
  the table but absent from the switch are flagged, not invented into structs.
- **Rejected — keep shared-shape structs, add per-mode body funcs only:** body funcs would
  be fixed-key (passing INV-3), but the struct is still mapped by >1 `#`-entry → INV-1
  failure, and it's the exact AP-1 the doc bans. Also defeats per-mode grading.
- **Rejected — code-generate the bodyless structs:** no codegen precedent in the packet lib;
  `mts_operation.go` hand-writes many structs in one file. Hand-write for reviewability and
  decompile-citation per struct. Tradeoff accepted: `operation.go` grows, but stays one file
  (AP-8 satisfied).

Naming: each struct/body name derives from the operation-key const semantics
(e.g. `GuildJoinErrorMaxMembers` for `THE_GUILD_..._MAX_NUMBER_OF_USERS`), not the raw mode
byte, so the `#`-entry, struct, body func, and operations key line up.

### D2 — Replace `GuildErrorBody` / `GuildErrorBody2` with per-error-mode body functions

`GuildErrorBody(code string)` lets the caller pick the mode (AP-4) and is the live INV-3
violation. Remove both. For each error arm, add a fixed-key body func:

```go
func GuildJoinErrorMaxMembersBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
    return atlas_packet.WithResolvedCode("operations", GuildOperationJoinErrorMaxMembers, func(mode byte) packet.Encoder {
        return clientbound.NewGuildJoinErrorMaxMembers(mode)
    })
}
```

Target-bearing arms take only the arm data (`target string`), never an op/code/key
selector. `RequestGuildNameBody` / `RequestGuildEmblemBody` (currently delegating through
`GuildErrorBody`, `operation_body.go:50-55`) are rewritten to their own fixed-key bodies.
Also fold `GuildInfoBody` and the two BBS bodies into the same `WithResolvedCode` pattern
so no arm hard-codes its mode (clears the D1-adjacent AP-2 latents in `info.go:70`,
`bbs.go:54,167`).

### D3 — Remove the phantom representatives; one `#`-entry per supported mode

`candidatesFromFName` currently returns `RequestAgreement` as a stand-in for the bare
`CWvsContext::OnGuildResult` root (run.go:1369-1372) and leaves `OnGuildBBSPacket` /
`CUIFadeYesNo::OnButtonClicked` as deferred roots. Per FR-11/INV-4:

- The bare dispatcher-root cases return **no representative** (or are removed if the tool
  no longer requires a root entry once every arm has a `#`-entry — confirm against the
  exemplar `OnFieldEffect` root handling in run.go during execution).
- Add one `case "CWvsContext::OnGuildResult#<Mode>":` per supported arm → its discrete
  struct. Same for `CUIGuildBBS::OnGuildBBSPacket#<Mode>`.
- Resolve the INV-1 `RequestAgreement` shared-struct concern: confirm exactly which
  `#`-entries point at `RequestAgreement` today (`#RequestAgreement` at run.go:1373; the
  PRD also names `#AgreementResponse`). If two entries share it, give each its own struct.

### D4 — Fix the serverbound `AgreementResponse` wire mismatch

run.go:1488 flags `AgreementResponse` as `❌ wire mismatch — extra Encode4 unk`: the IDA
read in `CField::SendCreateGuildAgreeMsg` is `Encode1(op) + Encode1(agreed)`, but Atlas
writes `WriteInt(unk) + WriteBool(agreed)`. Re-derive the exact read order from IDA
(per-version), correct `operation_agreement_response.go` Encode/Decode to match, update
the round-trip test, and freshen the run.go comment to the verified verdict. This is a
real codec correctness fix, not a comment edit (FR-12 forbids relaying a stale `❌` as a
live finding).

### D5 — BBS sub-family treated as a peer dispatcher

`CUIGuildBBS::OnGuildBBSPacket` dispatches on `(Decode1 - 6)` (run.go:1461). Give the BBS
clientbound arms (`BBSThreadList` mode 0x06, `BBSThread` mode 0x07, plus any not-found arm
in the switch) discrete structs with mode resolved from a `guild_bbs.yaml` operations
table, body funcs in `bbs_body.go`, and `#`-entries. Serverbound BBS codecs already exist
and are verified on v83/v87/v95/jms; v84 needs the reshift fold (D7).

### D6 — `guild.yaml` / `guild_bbs.yaml` as the per-version source of truth

Create `docs/packets/dispatchers/guild.yaml` (and `guild_bbs.yaml`) mirroring the
`mts_operation.yaml` format: `writer`, `fname`, `op`, `direction`, and an `operations:`
list of `{ key, modes: { gms_v83, gms_v84, gms_v87, gms_v95, jms_v185 } }`. The mode
values are **IDA-verified per version**, not copied from a sibling. The seed-template
`operations` tables and the live-config patch (D9) are reconciled against this file via
`packet-audit operations --check`. v84 is `== v83` unless IDA proves a divergence (FR-14).

### D7 — v84 reshift carryover

v84 shows ❌ across guild serverbound + BBS in STATUS.md. Per task-100's known pattern,
the v84 ❌ is most likely a stale **opcode/operations** carryover, not a structural
divergence (v84 ≡ v83 structurally per task-083). The work: confirm per row whether v84
needs (a) only the `operations` mode table populated, (b) the handler/writer **opcode**
registry rows reshifted, or (c) both — verified against the gms_84 template and the
registry, **not assumed**. Whichever applies is folded into this task (PRD §2), with
per-arm v84 byte fixtures.

### D8 — Grounding workflow (IDA per version)

Per FR-1..FR-3 and project rules: resolve the IDB by `list_instances`/`select_instance`
(v83/v87/v95/jms ports), confirm version match, decompile the actual `OnGuildResult` /
`OnGuildBBSPacket` switch, and cite function+address in each struct/test comment. v84 has
no IDB → treated as v83 unless the registry/template proves a shift (D7). An unresolved
fname is a **stop-and-ask**, never an auto-re-export or faked hash (FR-3).

### D9 — Live config patch + restart (FR-22/FR-23)

Seed templates apply only at tenant creation. After the codec work lands, produce and
**execute** a runbook patching each live tenant's per-version `operations` table, guild
handler/writer opcode tables, and serverbound handler validators, then restart the
affected channels — verifying no "unhandled message op 0xXX" for guild in channel logs
post-patch. The exact live tenant/version set is determined at execution time (PRD open
question 4) via the k8s/Grafana MCP tooling.

## 5. Component-by-component scope

- **`libs/atlas-packet/guild/clientbound/operation.go`** — split catch-alls into discrete
  structs; every Encode writes mode + full body, decompile-cited.
- **`libs/atlas-packet/guild/clientbound/{bbs,info}.go`** — route mode through body funcs,
  drop literal mode bytes.
- **`libs/atlas-packet/guild/operation_body.go` + `bbs_body.go`** — per-mode fixed-key body
  funcs; `GuildErrorBody`/`GuildErrorBody2` removed.
- **`libs/atlas-packet/guild/serverbound/operation_agreement_response.go`** — wire fix; add
  any missing v83/v87 sub-mode codecs surfaced by enumeration.
- **`libs/atlas-packet/guild/.../​*_test.go`** — per-arm byte fixtures with
  `// packet-audit:verify` markers + IDA citations; round-trip tests for serverbound.
- **`tools/packet-audit/cmd/run.go`** — one `#`-entry per supported mode; phantom roots and
  catch-alls removed; comments freshened to current verdicts.
- **`docs/packets/dispatchers/guild.yaml` + `guild_bbs.yaml`** — new per-version mode tables.
- **`docs/packets/dispatcher-lint-baseline.yaml`** — remove `CWvsContext::OnGuildResult`
  (keep `party`, `buddy`).
- **`docs/packets/audits/STATUS.md` + `status.json`** — regenerated (toolSha stamp).
- **`services/atlas-channel`** — `socket/handler/guild_operation.go`, `guild_bbs.go`,
  `guild_invite_reject.go`, guild writers, `guild/producer.go`: update call sites from the
  removed catch-all to the new per-mode bodies; verify every serverbound handler has a
  validator (FR-20).
- **`services/atlas-guilds`** — producers updated only as needed to emit a newly-split
  packet; no domain-model/REST change beyond that.
- **Seed templates** (`gms_83/84/87/95/jms`) — per-version operations/opcode/validator
  entries reconciled with the dispatcher yamls.
- **Live config runbook** — documented and executed.

## 6. Testing strategy

- **Per-arm byte fixtures** (clientbound): hand-computed expected bytes from the IDA read
  order, `// packet-audit:verify` marker, IDA citation in the comment; one per supported
  version (v84 ⬜ only if genuinely version-absent, else ✅).
- **Round-trip tests** (serverbound): Decode∘Encode identity where bidirectional; explicit
  coverage of the corrected `AgreementResponse`.
- **Regression guard:** v95/jms fixtures must not change (NFR backward-compat) — existing
  fixtures stay green.
- **Tooling gates:** `dispatcher-lint`, `matrix --check`, `fname-doc --check`,
  `operations --check` all exit 0.
- **Build/test:** `go build/vet/test -race ./...` in every changed module;
  `docker buildx bake atlas-channel` (+ `atlas-guilds` if its go.mod is touched) from the
  worktree root; `tools/redis-key-guard.sh` if Redis touched (not expected).

## 7. Execution phasing (for the plan phase)

1. **Enumerate** — decompile `OnGuildResult` + `OnGuildBBSPacket` switches per version;
   author `guild.yaml`/`guild_bbs.yaml`; reconcile against the existing operations keys.
   Resolve the §4.7 supported-arm set and the §9 open questions.
2. **Clientbound split** — discrete structs + per-mode body funcs; drop literal modes;
   remove `GuildErrorBody`/`GuildErrorBody2`.
3. **Serverbound fix** — `AgreementResponse` wire correction + any missing v83/v87 codecs.
4. **run.go rewire** — per-mode `#`-entries; remove phantoms/catch-alls; freshen comments.
5. **Fixtures + matrix** — per-arm byte fixtures all five versions; regenerate STATUS.
6. **v84 fold** — operations/opcode reshift per D7; v84 fixtures to ✅.
7. **Wiring** — atlas-channel/atlas-guilds call sites → per-mode bodies; validator audit.
8. **De-baseline + gates** — remove guild from baseline; all four packet-audit checks +
   build/vet/test/bake green.
9. **Live config** — patch + restart runbook, executed and verified in logs.
10. **Review + PR** — modular reviewer agents, then PR mirroring the family-complete
    checklist; CI green on PR HEAD.

## 8. Acceptance gates (from PRD §10, must all hold)

Discrete-per-mode shape · footguns gone (zero `mode: 0x` literal, zero `func(_ byte)`,
no caller-supplied selector) · mapping honest (one `#`-entry per mode, no phantom) ·
coverage ✅ all five versions (v84 cleared) · usability + live wiring + validators ·
`dispatcher-lint`/`matrix --check`/`fname-doc --check`/`operations --check` exit 0,
guild de-baselined (baseline only shrinks) · `go build/vet/test -race` clean +
`docker buildx bake` per touched service · code review before PR · CI green on PR HEAD.

## 9. Open questions (resolved during execution, not now)

- **Exact supported-arm count** behind the catch-alls — set by the IDA enumeration of the
  `OnGuildResult` switch (drives the struct count; scope is already "all IDA-enumerated
  arms").
- **Version-absent vs unimplemented per arm** — resolved per-arm against each IDB (absent
  → ⬜; unimplemented → must reach ✅).
- **v84 fix scope** — operations table only vs registry opcode reshift vs both — confirmed
  against the gms_84 template + registry (D7).
- **Live tenant/version set** — determined at execution time for the FR-22 patch+restart.
- **`RequestAgreement` sharing** — confirm whether `#RequestAgreement` and (per PRD)
  `#AgreementResponse` both map to one struct; split if so (D3).

## 10. Out of scope (from PRD §2 non-goals)

Migrating `party`/`buddy` off the baseline; new guild gameplay features; alliance-specific
dispatcher arms routed to a separate `alliance` family; `atlas-guilds` domain-model/REST
changes beyond emitting a newly-split packet; introducing a new tenant version (no new LB
socket ports).
