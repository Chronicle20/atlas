# Guild Dispatcher Family — Complete Implementation & De-baseline — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-06-18
---

## 1. Overview

The guild packet family (`CWvsContext::OnGuildResult` / `GUILD_OPERATION` clientbound,
`GUILD_OPERATION` serverbound, `CUserRemote::OnGuild{Name,Mark}Changed`, and the BBS
sub-dispatcher `CUIGuildBBS::OnGuildBBSPacket`) is a mode-prefix dispatcher: one opcode
whose leading byte selects one of N arms, each with its own body. It is ~60% implemented
in `libs/atlas-packet/guild/` — most `GuildOperation` clientbound modes, the serverbound
`Operation` sub-modes, BBS, and name/emblem-changed structs exist, and real consumers
(`atlas-guilds` service, `atlas-channel/.../guild/`, `socket/handler/guild_operation.go`,
guild writers) already emit them. On the coverage matrix, `gms_v95` and `jms_v185` are
fully verified; `gms_v83`/`gms_v87` carry a cluster of unverified sub-modes; `gms_v84` is
mostly stale from the task-100 opcode-reshift carryover.

Despite the matrix being mostly green, guild is **not "complete"** by the dispatcher-family
standard (`docs/packets/DISPATCHER_FAMILY.md`). It still carries the task-096 footguns that
keep it on `docs/packets/dispatcher-lint-baseline.yaml` (alongside `party` and `buddy`):

- `GuildErrorBody(code string)` / `GuildErrorBody2(code string, target string)` are
  **caller-specified operation selectors** (AP-4 / INV-3) — the `code` parameter flows into
  the `WithResolvedCode("operations", …)` key, letting a caller send the wrong mode.
- Those bodies front an **Error/notice catch-all** (`#ErrorMessage`,
  `#ErrorMessageWithTarget`) that fans ~10–15 distinct sub-ops through one struct + one
  string-keyed body.
- The **`RequestAgreement` struct is shared by two `#`-entries** (`#RequestAgreement` and
  `#AgreementResponse`) — INV-1.
- Top-level catch-alls (`OnGuildResult`, `OnGuildBBSPacket`, `CUIFadeYesNo::OnButtonClicked`)
  return a **phantom representative** with `deferred to _pending.md` — AP-5 / INV-4.
- A flagged **codec bug**: serverbound `AgreementResponse` is marked
  `❌ wire mismatch — extra Encode4 unk` vs `CField::SendCreateGuildAgreeMsg` in `run.go`.

This task migrates guild to the canonical discrete-per-mode pattern, drives every supported
arm to ✅ across all five versions (including the v84 reshift cleanup), removes guild from
the dispatcher-lint baseline, and patches live tenant config so the completed family is
actually usable in running environments.

The governing rule (from `DISPATCHER_FAMILY.md`): **`matrix ✅` means codec byte-correct,
nothing more.** Discrete-per-mode shape, config-driven mode resolution, footgun-free APIs,
feature usability, live-config wiring, and honest grounding are separate requirements proven
by the gates in §10, not by a green cell.

## 2. Goals

Primary goals:
- Enumerate the full `GUILD_OPERATION` (`CWvsContext::OnGuildResult`) clientbound switch and
  the BBS (`CUIGuildBBS::OnGuildBBSPacket`) switch from IDA, and give **every supported
  sub-op its own discrete struct** (one consolidated clientbound file per family), including
  every error/notice arm currently hidden behind the catch-all.
- Eliminate the AP-4 / INV-3 footguns: replace `GuildErrorBody(code)` / `GuildErrorBody2(code,
  target)` and any string-keyed body with **per-mode body functions** that fix the operation
  key as a const and pass the **resolved** mode through `WithResolvedCode("operations",
  FIXED_KEY, func(mode byte)…)`.
- Resolve the INV-1 shared-struct violation (`RequestAgreement` mapped by two `#`-entries)
  and remove every phantom/dangling `#`-entry (INV-4 / AP-5).
- Fix the serverbound `AgreementResponse` wire mismatch (extra `Encode4 unk`) against the
  IDA read order.
- Drive **all five versions** (`gms_v83`, `gms_v84`, `gms_v87`, `gms_v95`, `jms_v185`) to ✅
  for every supported guild arm — folding the task-100 v84 reshift carryover for guild into
  this task — with per-mode byte fixtures carrying `// packet-audit:verify` markers and IDA
  citations.
- Include the **guild BBS sub-family** in the same migration.
- Ensure every verified codec is **usable**: a per-mode body function exists, and a real
  consumer/handler emits it (or the body-function layer is the documented API).
- **Patch live tenant config** (per-version `operations` mode tables, opcode tables, handler
  validators) and restart channels so the completed family works in running tenants.
- Remove `CWvsContext::OnGuildResult` from `dispatcher-lint-baseline.yaml`; keep `party` and
  `buddy` as-is.

Non-goals:
- Migrating `party` or `buddy` off the baseline (separate tasks).
- New guild gameplay features beyond what the existing packets/handlers support (e.g. no new
  alliance mechanics, no new BBS capabilities) — this is packet-family completion, not
  feature expansion.
- Alliance-specific dispatcher modes that are routed to a separate `alliance` family (only
  guild arms of shared opcodes are in scope; alliance arms stay out unless an arm is part of
  the guild switch itself).
- Changing the `atlas-guilds` service domain model or REST surface beyond what is needed to
  emit a newly-split packet.

## 3. User Stories

- As a packet-platform maintainer, I want guild migrated to the discrete-per-mode pattern so
  `dispatcher-lint` enforces the invariants on guild and the baseline shrinks by one family.
- As a gameplay engineer, I want a fixed-key per-mode body function for every guild sub-op so
  I can emit any guild result packet without being able to pass the wrong mode byte.
- As a player on a v83/v84/v87 tenant, I want guild operations (invite, join, kick, withdraw,
  notice, titles, emblem, errors, BBS) to render correctly so the guild UI is not crashing or
  silently dropping packets.
- As a release engineer, I want the live tenant `operations`/opcode/validator tables patched
  per version so the completed handlers and writers actually fire after deploy.
- As a reviewer, I want every byte traced to a decompile line or checked-in export entry so I
  can trust the family is grounded, not inferred from a sibling version.

## 4. Functional Requirements

### 4.1 Grounding & evidence
- FR-1. Every byte, opcode, field, and mode value MUST trace to a decompile line (function +
  address) or a checked-in export entry, cited in the struct/test comment. No values from
  general MapleStory knowledge or memory.
- FR-2. The IDA instance MUST be resolved by loaded IDB (`list_instances` / `select_instance`)
  and confirmed to match the target version before reading. Read order is taken from the
  client's actual read, never assumed from a sibling version.
- FR-3. If the correct IDB is not loaded and the export lacks the function, STOP and escalate
  — never substitute or guess. An unresolved packet-audit fname is a stop-and-ask, never an
  auto-re-export or faked hash.

### 4.2 Mode enumeration & discrete structs
- FR-4. Decompile the full `CWvsContext::OnGuildResult` switch and the
  `CUIGuildBBS::OnGuildBBSPacket` switch; record the complete set of supported sub-ops
  (mode byte + body) per version.
- FR-5. Each supported mode MUST have **one discrete struct** in the family's single
  consolidated clientbound file (no `*_result_<shape>_modes.go` sprawl — AP-8). Bodyless
  notice/error arms still get their own `struct { mode byte }` (discrete means discrete, even
  when two arms share a wire shape).
- FR-6. Each struct's `Encode` MUST write the mode byte then the **full arm body** — no
  mode-byte-only stub for an arm that has a body (AP-7, the "passes on 1 byte" false pass).
- FR-7. No struct may serve >1 mode (AP-1 / INV-1), including any catch-all `#`-entry that
  fronts many sub-ops through one struct — the `#ErrorMessage` / `#ErrorMessageWithTarget`
  catch-all and the shared `RequestAgreement`/`AgreementResponse` mapping MUST be split.

### 4.3 Config-driven mode resolution (footgun removal)
- FR-8. Mode bytes MUST be resolved from the tenant `operations` table, never hard-coded — no
  `mode: 0x..` literal (AP-2 / INV-2a), no `func(_ byte)` discarding the resolved mode
  (AP-3 / INV-2b).
- FR-9. Each per-mode body function MUST fix the operation key as a const and pass the
  resolved mode through: `WithResolvedCode("operations", FIXED_KEY, func(mode byte) …)`. The
  constructor takes `mode byte` first.
- FR-10. No body function may take a caller-supplied operation selector of any name
  (`op`/`code`/`mode`/`key`/`errorCode`/`reason`/…) — INV-3 by-name AND semantic. Replace
  `GuildErrorBody(code)` / `GuildErrorBody2(code, target)` with one fixed-key body per error
  sub-op.

### 4.4 run.go candidate mapping
- FR-11. `candidatesFromFName` MUST have one `#`-entry per supported mode mapping to that
  mode's discrete struct (`{name, pkg: "guild", dir}`). No dangling `#`-entry, no phantom
  representative, no top-level catch-all returning a stand-in (AP-5 / INV-4).
- FR-12. Every `#`-entry comment MUST reflect the current code (verified against the struct's
  Encode/Decode + the per-version audit verdict). Stale `❌ MISSING` / `WriteInt`-class
  comments MUST be freshened as part of the work — never relayed as a live finding.

### 4.5 Codec correctness (per version)
- FR-13. Encode writes exactly what the client reads, in order, at the right widths (watch
  byte-vs-int — the `CapacityChange` class of bug).
- FR-14. Version divergence MUST be handled explicitly and gated correctly (`>=87`, not `>83`;
  `v84..v86 == v83` unless proven otherwise — the `MajorVersion` off-by-one). Version-absent
  arms MUST be genuinely absent (⬜), not silently emitting a wrong shape.
- FR-15. Decode mirrors Encode (round-trip) where the codec is bidirectional (serverbound
  guild operations).
- FR-16. Fix the serverbound `AgreementResponse` wire mismatch (drop or correct the extra
  `Encode4 unk`) per `CField::SendCreateGuildAgreeMsg`.
- FR-17. Each supported arm MUST have a byte-fixture test with a `// packet-audit:verify`
  marker and the IDA citation in the comment.

### 4.6 Usability & wiring
- FR-18. A body function / writer layer MUST exist for every supported arm (no orphaned codec
  — AP-6 / INV-5).
- FR-19. A real consumer/handler MUST emit it, or the body-function layer MUST be the
  documented API a future feature calls. Existing `atlas-channel` guild handlers/writers and
  `atlas-guilds` producers MUST be updated to call the new per-mode bodies where they
  currently call the catch-all.
- FR-20. Serverbound: every handler MUST be registered with a validator — a missing/empty
  validator means `BuildHandlerMap` silently drops it.

### 4.7 Live config & per-version tables
- FR-21. Seed templates MUST carry the new/changed handler & writer opcodes, the per-version
  `operations` mode table entries for every guild sub-op, and validators — for every
  supported version (`gms_83`, `gms_84`, `gms_87`, `gms_95`, `jms`).
- FR-22. Live tenant config MUST be patched (per-version `operations`/opcode/validator tables)
  and channels restarted so existing tenants pick up the completed family (seed templates
  apply only at tenant creation). The runbook is documented AND executed.
- FR-23. Per-version operations/opcode/validator tables MUST be populated for every supported
  version — the v87/v95/jms "operations table missing → ResolveCode 99 → client crash" trap.

### 4.8 Honesty / no deferral
- FR-24. No `// TODO`, stubbed handler, or 501 in landed commits. Bounded work is finished,
  not split to a follow-up to dodge it. Genuine external blockers or ambiguous design
  decisions are surfaced explicitly (stop-and-ask), not buried.

## 5. API Surface

This is a packet-library + service-wiring task; the externally observable surface is the wire
protocol and the body-function API in `libs/atlas-packet/guild/`.

- **New/changed body functions** (`libs/atlas-packet/guild/operation_body.go`,
  `bbs_body.go` or equivalent): one exported `Guild<Mode>Body(<arm data, NO selector>)` per
  supported clientbound mode, each fixing its operation-key const. `GuildErrorBody` and
  `GuildErrorBody2` are **removed** and replaced by per-error-mode bodies.
- **New/changed clientbound structs** (`libs/atlas-packet/guild/clientbound/operation.go`,
  `bbs.go`): one discrete `type Guild<Mode> struct` + `New<Mode>(mode byte, …)` constructor
  per supported mode.
- **Serverbound** (`libs/atlas-packet/guild/serverbound/…`): `AgreementResponse` Decode/Encode
  corrected; any missing v83/v87 sub-mode codecs added.
- **`run.go` `candidatesFromFName`**: one `#`-entry per supported mode; catch-alls removed.
- **atlas-channel writers/handlers** updated to call per-mode bodies.
- No new REST endpoints or JSON:API resources are expected.

## 6. Data Model

No relational schema changes are anticipated. The "data model" here is configuration:

- **Tenant `operations` table** (per version): each guild sub-op key → its mode byte. New
  entries for every split error/notice/BBS arm, populated for all five versions.
- **Tenant opcode tables** (handler/writer): guild opcodes per version (the v84 reshift
  carryover correction lives here + in the registry rows).
- **Tenant validator map**: each guild serverbound handler entry → a validator
  (`LoggedInValidator` per version convention).

Seed templates (`gms_83`, `gms_84`, `gms_87`, `gms_95`, `jms`) are the source for new tenants;
live config patches cover existing tenants.

## 7. Service Impact

- **libs/atlas-packet** — primary: discrete structs, per-mode body funcs, fixtures, codec fix.
- **tools/packet-audit** — `run.go` `#`-entry rewiring; regenerate `STATUS.md` / `status.json`;
  remove guild from `dispatcher-lint-baseline.yaml`.
- **services/atlas-channel** — `socket/handler/guild_operation.go`, `guild_bbs.go`,
  `guild_invite_reject.go`, guild writers, and `guild/producer.go` updated to call per-mode
  bodies; validator coverage verified.
- **services/atlas-guilds** — producers updated only as needed to emit newly-split packets;
  no domain-model change beyond that.
- **Seed templates + live tenant config** — per-version `operations`/opcode/validator tables.
- **docs/packets** — evidence records, fixtures, regenerated matrix, baseline edit.

## 8. Non-Functional Requirements

- **Grounding/honesty:** every value decompile-cited; stale comments freshened; no invented
  values (project "Verification Over Memory" + "Grounding & Honesty" rules).
- **Multi-tenancy:** all config is per-tenant, per-version; no cross-tenant leakage; mode
  resolution always via the tenant `operations` table.
- **Observability:** dropped/unhandled guild ops should surface in channel logs at the
  existing level; verify no "unhandled message op 0xXX" after the live patch.
- **Backward compatibility:** v95/jms already verified — their wire output MUST NOT change
  (regression-guard with existing fixtures); changes are additive/correctness for v83/v84/v87.
- **CWD/path robustness:** any path logic verified from the repo root, not relying on worktree
  nesting (the `../../` INV-4 bug that passed in a 2-deep worktree and failed in CI).

## 9. Open Questions

- Exact count of distinct error/notice sub-ops behind the catch-all — to be enumerated from
  IDA in the design phase (scope was chosen as "all IDA-enumerated sub-ops", so the count
  sets the struct count, not the scope decision).
- Whether any guild arm is genuinely version-absent on v83/v84 (→ ⬜) vs merely unimplemented
  (→ must reach ✅) — resolved per-arm against each IDB during design/execution.
- Whether the v84 guild opcode rows need registry reshift correction (task-100 carryover) in
  addition to the operations table, or only the latter — confirm against the gms_84 template
  and registry during execution.
- Which running tenants/versions are live at execution time (determines the exact live-config
  patch + restart set for FR-22).

## 10. Acceptance Criteria

A. **Discrete-per-mode shape**
- [ ] One discrete struct per supported guild + BBS mode, in one consolidated clientbound
      file each; bodyless arms have their own `struct { mode byte }`.
- [ ] Each struct's `Encode` writes the full arm body (not just the mode byte), every field
      decompile-cited.
- [ ] No struct serves >1 mode; `RequestAgreement`/`AgreementResponse` split; no shared error
      catch-all.

B. **Config-driven resolution / footguns gone**
- [ ] Every constructor takes `mode byte`; every body func resolves it via
      `WithResolvedCode("operations", FIXED_KEY, func(mode byte)…)`. Zero `mode: 0x` literals,
      zero `func(_ byte)`.
- [ ] `GuildErrorBody` / `GuildErrorBody2` removed; no body func takes a caller-supplied
      op/code/mode/key/errorCode/reason selector (INV-3 by-name AND semantic).

C. **Mapping & honesty**
- [ ] `candidatesFromFName`: one `#`-entry per supported mode → its discrete struct; no
      dangling `#`-entry, phantom representative, or top-level catch-all stand-in.
- [ ] Every `#`-entry comment reflects current code and the per-version verdict; stale
      comments freshened.
- [ ] Serverbound `AgreementResponse` wire mismatch fixed.

D. **Coverage**
- [ ] Every supported guild + BBS arm is ✅ on `gms_v83`, `gms_v84`, `gms_v87`, `gms_v95`,
      `jms_v185` (version-absent arms → ⬜). v84 reshift carryover for guild cleared.
- [ ] Each arm has a byte-fixture with `// packet-audit:verify` + IDA citation.

E. **Usability & live wiring**
- [ ] No orphaned codec — every struct constructed by a body function.
- [ ] atlas-channel handlers/writers + atlas-guilds producers call the per-mode bodies; every
      serverbound handler has a validator.
- [ ] Seed templates carry per-version operations/opcode/validator entries for all guild ops.
- [ ] Live tenant config patched (per-version) and channels restarted; verified no unhandled
      guild op in logs post-patch.

F. **Tooling gates (all exit 0)**
- [ ] `packet-audit dispatcher-lint` clean; `CWvsContext::OnGuildResult` removed from
      `dispatcher-lint-baseline.yaml` (baseline only shrinks).
- [ ] `packet-audit matrix --check` — no orphan/dangling/stale/drift, no conflict-count
      increase; `STATUS.md` / `status.json` regenerated & committed (toolSha stamp).
- [ ] `packet-audit fname-doc --check` clean.
- [ ] `packet-audit operations --check` clean.

G. **Build & test gates (every changed module)**
- [ ] `go build ./...` clean.
- [ ] `go vet ./...` clean.
- [ ] `go test -race ./...` clean — including new/updated tests (call sites updated in the
      same commit).
- [ ] `docker buildx bake atlas-<svc>` for every service whose `go.mod` was touched.
- [ ] `tools/redis-key-guard.sh` clean (if Redis touched).
- [ ] Path logic verified from repo root (no worktree-nesting dependence).

H. **Final**
- [ ] Code review run before PR (modular reviewer agents).
- [ ] PR description mirrors the dispatcher "family complete" checklist.
- [ ] CI green on the actual PR HEAD (the check job specifically), not just locally.

---

## Appendix A — Source completion checklist (verbatim, governing)

> The governing rule: matrix ✅ means codec byte-correct, nothing more. It is not "complete."
> Discrete-per-mode shape, config-driven resolution, footgun-free APIs, feature-usability,
> live-config wiring, and honest grounding are all separate requirements — proven by the gates
> and the items below, not by a green matrix cell.

**1. Grounding & evidence (no inventing)**
- [ ] Every byte, opcode, field, and mode value traces to a decompile line (function + address)
      or a checked-in export entry — cited in the test/struct comment. No values from
      MapleStory general knowledge or memory.
- [ ] IDA instance resolved by loaded IDB (`list_instances`/`select_instance`), and confirmed
      it matches the target version before reading.
- [ ] Read order taken from the client's actual read, not assumed from a sibling version.
- [ ] If the right IDB isn't loaded and the export lacks the function → STOP and escalate,
      don't substitute or guess.
- [ ] An unresolved packet-audit fname is a stop-and-ask, never an auto-re-export/faked hash.

**2. Codec correctness (per version)**
- [ ] Encode writes exactly what the client reads, in order, at the right widths (watch
      byte-vs-int — the CapacityChange class of bug).
- [ ] Version divergence handled explicitly and gated correctly (>=87 not >83; v84–86 == v83
      unless proven otherwise — the MajorVersion off-by-one).
- [ ] Version-absent cases are genuinely absent (⬜), not silently emitting a wrong shape.
- [ ] Decode mirrors Encode (round-trip) where the codec is bidirectional.
- [ ] Byte-fixture test exists with a `// packet-audit:verify` marker and the IDA citation in
      the comment.

**3. Mode-prefix dispatcher families** (see `docs/packets/DISPATCHER_FAMILY.md`; enforced by
dispatcher-lint INV-1..INV-5 / anti-patterns AP-1..AP-8)
- [ ] One discrete struct per supported mode, in ONE consolidated clientbound file — no
      `*_result_<shape>_modes.go` sprawl (AP-8). Bodyless arms still get their own
      `struct { mode byte }`.
- [ ] Each struct's Encode writes the mode byte then the full arm body — no mode-byte-only stub
      for an arm that has a body (AP-7, the "passes on 1 byte" false pass).
- [ ] No struct serves >1 mode (AP-1 / INV-1) — including a catch-all #-entry that fronts many
      sub-ops through one struct (the #Error/#ErrorMessage blind spot). Split it.
- [ ] Mode byte resolved from config, never hard-coded — no `mode: 0x..` literal (AP-2 /
      INV-2a), no `func(_ byte)` discarding the resolved mode (AP-3 / INV-2b).
- [ ] Body function fixes the operation key and passes the resolved mode through:
      `WithResolvedCode("operations", FIXED_KEY, func(mode byte) …)`. The constructor takes
      `mode byte` first.
- [ ] No caller-specified operation selector (AP-4 / INV-3) — the key is a const, not a
      parameter (code/errorCode/reason/anything). Per-mode body funcs, not one string-keyed
      body func.
- [ ] `run.go` candidatesFromFName: one #-entry per supported mode → its discrete struct; no
      dangling #-entry or phantom representative (AP-5 / INV-4).
- [ ] Op-row reaches ✅ only when every supported arm is verified (FIELD_EFFECT model); family
      removed from families.yaml once complete.

**4. Usability & wiring (a verified codec nobody can send is incomplete)**
- [ ] A body function / writer layer exists that a feature can call — no orphaned codec
      (AP-6 / INV-5).
- [ ] A real consumer/handler emits it, or the body-function layer is the documented API a
      future feature calls.
- [ ] Serverbound: a handler is registered with a validator — a missing/empty validator means
      BuildHandlerMap silently drops it.
- [ ] Live tenant config carries the new handler/writer opcode, the dispatcher operations mode
      table (per-version!), and the validator — seed templates apply only at tenant creation;
      existing tenants need a config patch + channel restart.
- [ ] Per-version operations/opcode/validator tables populated for every supported version (the
      v87/v95/jms "operations table missing → ResolveCode 99 → client crash" trap).
- [ ] New LB socket ports (new version) added in both k8s base yaml and login/channel service
      config. *(N/A — no new version introduced by this task; retained for completeness.)*

**5. Honesty: comments, docs, no deferral**
- [ ] run.go #-entry comments reflect current code — verified against the struct's
      Encode/Decode + the per-version audit verdict. Never relay a comment's ❌ "MISSING"
      verdict as a live finding (the stale-comment trap); freshen stale comments as part of the
      work.
- [ ] No `// TODO`, stubbed handler, or 501 in landed commits. Bounded work finished, not split
      to a follow-up to dodge it.
- [ ] Genuine external blockers / ambiguous design decisions surfaced explicitly
      (stop-and-ask), not buried.

**6. Tooling gates — all exit 0**
- [ ] `packet-audit dispatcher-lint` (if a dispatcher family) — clean, family de-baselined if
      migrated (baseline only shrinks).
- [ ] `packet-audit matrix --check` — no orphan/dangling/stale/drift, no conflict-count
      increase; STATUS.md/status.json regenerated & committed after any tools/packet-audit
      change (the toolSha stamp).
- [ ] `packet-audit fname-doc --check` — clean.
- [ ] `packet-audit operations --check` — clean.

**7. Build & test gates (every changed module)**
- [ ] `go build ./...` clean.
- [ ] `go vet ./...` clean.
- [ ] `go test -race ./...` clean — including new/updated tests (test files are part of the
      package; update call sites in the same commit).
- [ ] `docker buildx bake atlas-<svc>` for every service whose go.mod was touched (catches
      missing `COPY libs/...` the workspace build won't).
- [ ] `tools/redis-key-guard.sh` clean (if Redis touched).
- [ ] CWD/path robustness: any path logic verified from the repo root, not relying on worktree
      nesting (the `../../` INV-4 bug — passed locally in a 2-deep worktree, failed in CI).

**8. Final**
- [ ] Code review run before PR (the modular reviewer agents).
- [ ] PR description mirrors the dispatcher "family complete" checklist where relevant.
- [ ] CI green on the actual PR HEAD (not just locally) — the check job specifically.
