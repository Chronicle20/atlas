# Party + Buddy Dispatcher Family Migration — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-06-23
---

## 1. Overview

`CWvsContext::OnPartyResult` (`PARTY_OPERATION`) and `CWvsContext::OnFriendResult`
(`BUDDYLIST`) are the last two **mode-prefix dispatcher families** still baselined
in `docs/packets/dispatcher-lint-baseline.yaml`. A mode-prefix dispatcher is a
single opcode whose leading byte (`CInPacket::Decode1`) is a discriminator the
client `switch`es on, routing to N sub-handler "arms," each reading a different
body. The canonical Atlas pattern (`docs/packets/DISPATCHER_FAMILY.md`) requires
**one discrete struct per supported mode**, a per-mode body function that resolves
the per-version mode byte from the tenant `operations` table, a per-mode
`candidatesFromFName` `#`-entry, and per-mode byte-fixture verification.

Both families currently violate this. Each fronts a **single shared `Error`
struct** selected by a caller-supplied string code:

- `party/clientbound/operation_body.go` — `PartyErrorBody(code string, name string)`
  routes every error sub-op (`UNABLE_TO_FIND_THE_CHARACTER`,
  `UNABLE_TO_FIND_THE_REQUESTED_CHARACTER_IN_THIS_CHANNEL`,
  `HAVE_DENIED_REQUEST_TO_THE_PARTY`, plus a runtime `errorType`) through one
  `clientbound.NewError(mode, name)`. This is the AP-4 footgun (caller picks the
  mode → can send the wrong one → client crash).
- `buddy/operation_body.go` — `BuddyErrorBody(errorCode string)` routes ~10 error
  modes (`BUDDY_LIST_FULL`, `OTHER_BUDDY_LIST_FULL`, `ALREADY_BUDDY`,
  `CANNOT_BUDDY_GM`, `CHARACTER_NOT_FOUND`, `UNKNOWN_ERROR_1..4`, …) through one
  `clientbound.NewBuddyError(mode, hasExtra)`. Its selector is named `errorCode`,
  not `code`, which is precisely why it escaped the original by-name INV-3 check
  and only surfaced when task-101 hardened the linter to semantic matching.

This task migrates both families to the canonical discrete-per-mode pattern,
following the task-103 (guild) / task-104 (message) playbook exactly, and removes
both entries from the dispatcher-lint baseline. After this task, the baseline is
empty and the `dispatcher-family` campaign is complete.

This is a refactor of the packet codec/serialization layer plus its verification
artifacts. It changes **how** party/buddy error packets are constructed and
graded; it introduces **no** new gameplay behavior, no new error conditions, and
no protocol changes visible to the client (the bytes on the wire are unchanged —
the same mode byte + body for each error is emitted, just from a discrete struct
chosen by a fixed key instead of a shared struct chosen by a caller string).

## 2. Goals

Primary goals:

- Split the party `Error` catch-all into discrete per-mode structs — one struct
  per error case the `OnPartyResult` client switch handles.
- Split the buddy `Error` catch-all into discrete per-mode structs — one struct
  per error case the `OnFriendResult` client switch handles.
- Replace `PartyErrorBody(code, name)` and `BuddyErrorBody(errorCode)` with
  fixed-key per-mode body functions (no caller-supplied op/code/mode/key
  selector) — kill the AP-4 / INV-3 footgun.
- Author `docs/packets/dispatchers/party.yaml` and `buddy.yaml` mode tables
  (neither exists today), each the IDA-verified source of truth for the
  per-version mode byte, including the v95 non-uniform shift.
- Verify every supported arm with a per-mode `#`-entry, synthetic export entry,
  audit report, byte-fixture (`// packet-audit:verify` marker), and pinned
  evidence across **gms_v83, gms_v84, gms_v87, gms_v95, jms_v185** (version-absent
  arms marked ⬜).
- Migrate all `services/atlas-channel` call sites to the new fixed-key body
  functions.
- Populate any empty v87/v95/jms `operations` tables for these families from the
  new dispatcher yamls (the `bug_operations_mode_tables_missing_v87_v95_jms` gap
  guild hit in task-103).
- Remove `CWvsContext::OnPartyResult` and `CWvsContext::OnFriendResult` from
  `docs/packets/dispatcher-lint-baseline.yaml`.

Non-goals:

- The latent `pet` `PetDespawnBody(reason string)` AP-4 footgun
  (`libs/atlas-packet/pet/clientbound/activated_body.go`) — a separate unenrolled
  family (DISPATCHER_FAMILY.md "Known limitations"); tracked elsewhere.
- `CWvsContext::OnAllianceResult` (`ALLIANCE_OPERATION`) — unimplemented (❌ all
  versions, no Packet binding); not a baselined family.
- Any new party/buddy gameplay behavior, new error conditions, or new error
  messages.
- Changing the non-error party/buddy arms that are already discrete
  (`Created`/`Disband`/`Left`/`Join`/`Update`/`ChangeLeader`/`Invite`/`TownPortal`
  for party; `ListUpdate`/`BuddyUpdate`/`Invite`/`ChannelChange`/`CapacityUpdate`
  for buddy) beyond what the split mechanically requires.

## 3. User Stories

- As a backend engineer emitting a party/buddy error packet, I want a body
  function whose name fixes the error mode (`PartyUnableToFindCharacterBody(name)`,
  not `PartyErrorBody("...", name)`) so that I cannot accidentally pass a wrong or
  version-invalid mode byte that crashes the client.
- As a packet-audit maintainer, I want each error arm to have its own discrete
  struct, `#`-entry, and byte-fixture so that the coverage matrix grades each arm
  independently and `dispatcher-lint` can enforce the family with no baseline
  exemption.
- As a reviewer, I want `dispatcher-lint`, `matrix --check`, `fname-doc --check`,
  and `operations --check` to all exit 0 with an **empty** baseline so that I have
  machine-checked proof the dispatcher-family campaign is complete.
- As a tenant operator on v87/v95/jms, I want the party/buddy `operations` tables
  populated so that newly-split error modes resolve to the correct version byte
  instead of the `99` fallback that crashes the client.

## 4. Functional Requirements

### 4.1 Mode enumeration (IDA, all 5 versions)

- FR-1.1 Decompile `CWvsContext::OnPartyResult` in each IDB
  (gms_v83 :13342, gms_v84 :13337, gms_v87 :13341, gms_v95 :13340, jms_v185 :13339)
  and enumerate every error case in the client switch with its mode byte. Use
  `select_instance(port)` and confirm the function address before reading.
- FR-1.2 Decompile `CWvsContext::OnFriendResult` in each IDB and enumerate every
  error case with its mode byte.
- FR-1.3 For v95, verify the mode bytes against the **non-uniform shift** family
  (same as the opcode-table / guild bug) — do not fold from v83; read each arm's
  case value from the v95 switch and cross-check via the decrypted StringPool
  message where the arm shows one.
- FR-1.4 Record version-absent arms explicitly (an arm with no case in a given
  version's switch → that version's mode omitted; matrix cell ⬜). Mirror the
  guild `SET_SKILL_RESPONSE`-in-jms precedent.
- FR-1.5 Every mode byte in `party.yaml`/`buddy.yaml` is cited to the decompile
  line / case label of THAT version — never inferred, never copied from general
  MapleStory knowledge.

### 4.2 Discrete structs (full client-switch enumeration)

- FR-2.1 For **every** error case the `OnPartyResult` switch handles, create a
  discrete struct in the single consolidated `party/clientbound` error file
  (one family → one struct file; no `*_modes.go` sprawl, AP-8). Bodyless
  notice/error arms are still their own `struct { mode byte }` (discrete means
  discrete even when two arms share a wire shape).
- FR-2.2 Same for every `OnFriendResult` error case in `buddy/clientbound`.
- FR-2.3 Each struct's `Encode` writes the mode byte then the **full arm body**,
  every field cited to a decompile line. No mode-byte-only stub for an arm that
  has a body (AP-7). The party error arm currently writes `mode + AsciiString
  name`; the buddy error arm writes `mode` (+ extra int when `UNKNOWN_ERROR`) —
  preserve each arm's exact wire shape per the decompile.
- FR-2.4 Delete the shared `Error` struct from each package once all its modes
  have discrete replacements; no dangling `#`-entry or audit report may cite the
  removed struct (AP-5 / INV-4).

### 4.3 Body functions (fixed key, resolved mode)

- FR-3.1 Each error arm gets a body function in
  `party/clientbound/operation_body.go` / `buddy/operation_body.go` of the form
  `WithResolvedCode("operations", <FIXED_KEY_CONST>, func(mode byte) packet.Encoder
  { return clientbound.New<Arm>(mode, …) })`.
- FR-3.2 No body function takes a caller-supplied `op`/`code`/`mode`/`key`
  selector, nor any parameter (of any name) that flows into the `WithResolvedCode`
  key (INV-3 semantic). `PartyErrorBody(code, name)` and `BuddyErrorBody(errorCode)`
  are removed.
- FR-3.3 Every constructor takes `mode byte` as its first parameter; the body
  function passes the **resolved** mode through (`func(mode byte)`, never
  `func(_ byte)`, never a `mode: 0x..` literal — INV-2).
- FR-3.4 Every discrete struct is constructed by at least one body function (no
  orphaned codec — INV-5). Because the chosen breadth is full client-switch
  enumeration, arms Atlas does not currently emit still get a body function so the
  codec is a usable API (DISPATCHER_FAMILY.md step 6); these are the future-feature
  entry points.

### 4.4 Audit wiring + verification

- FR-4.1 `tools/packet-audit/cmd/run.go` `candidatesFromFName`: replace the single
  `CWvsContext::OnPartyResult#Error` / `CWvsContext::OnFriendResult#Error`
  catch-all entries with one `#<Mode>` entry per error arm, each returning that
  arm's discrete struct `{name, pkg, dir: clientbound}`.
- FR-4.2 For each arm: synthetic `#`-suffixed entry in the per-version export,
  audit report, byte-fixture with a `// packet-audit:verify` marker, and evidence
  where the grader requires it (per `VERIFYING_A_PACKET.md`).
- FR-4.3 The `PARTY_OPERATION` and `BUDDYLIST` op-rows aggregate worst-of all arms
  and reach ✅ for a version only when every supported arm for that version is
  verified (the FIELD_EFFECT model; the family is NOT added to `families.yaml`).

### 4.5 Operations tables

- FR-5.1 Author `docs/packets/dispatchers/party.yaml` and
  `docs/packets/dispatchers/buddy.yaml` in the guild.yaml format (writer, fname,
  op, direction, per-key per-version mode table with the IDA-verified addresses
  and the v95-shift note).
- FR-5.2 Run `packet-audit operations` (generate) to populate any empty
  v87/v95/jms `operations` maps for these families from the new yamls; the v83/v84
  templates already carry the existing arms' modes and must continue to match.
- FR-5.3 `operations --check` exits 0 for both families across all applicable
  versions after generation.

### 4.6 Call-site migration

- FR-6.1 Migrate every `PartyErrorBody(...)` call site
  (`socket/handler/party_operation.go:97,106`,
  `kafka/consumer/party/consumer.go:452`,
  `kafka/consumer/invite/consumer.go:171`) to the matching fixed-key body
  function. The `consumer/party` site passes a runtime `errorType`; it must map
  that runtime value to the specific body function via an explicit switch/lookup,
  not by passing the string into a generic body func.
- FR-6.2 Migrate every `BuddyErrorBody(...)` call site
  (`kafka/consumer/buddylist/consumer.go:238`) to the matching fixed-key body
  function, with the same runtime-value → specific-body mapping.
- FR-6.3 No call site retains a string error selector after migration.

### 4.7 De-baseline

- FR-7.1 Remove `CWvsContext::OnPartyResult` and `CWvsContext::OnFriendResult`
  from `docs/packets/dispatcher-lint-baseline.yaml`. The `exempt_families` list is
  then empty (the baseline only ever shrinks).
- FR-7.2 `dispatcher-lint` exits 0 with no suppressed-violation notes for either
  family.

## 5. API Surface

No REST/HTTP surface. The "API" here is the Go body-function layer in
`libs/atlas-packet`:

Removed:
- `party/clientbound.PartyErrorBody(code string, name string) → encoder`
- `buddy.BuddyErrorBody(errorCode string) → encoder`

Added (one per error arm; names final in design — illustrative):
- `party/clientbound.Party<Error>Body([arm fields]) → encoder` (e.g.
  `PartyUnableToFindCharacterBody(name string)`,
  `PartyRequestDeniedBody(name string)`, …).
- `buddy.Buddy<Error>Body([arm fields]) → encoder` (e.g.
  `BuddyListFullBody()`, `BuddyAlreadyBuddyBody()`,
  `BuddyUnknownErrorBody()` — the arm with the trailing extra int, …).

Each takes only that arm's body data (no mode/op/code selector); the mode byte is
resolved internally from the tenant `operations` table via the fixed key.

Error cases: a key absent from a version's `operations` table resolves to the
`99` fallback (existing behavior). FR-5.2 prevents this for supported arms by
populating the tables; version-absent arms (FR-1.4) are never emitted for that
version.

## 6. Data Model

No database entities. The "data model" is the tenant configuration `operations`
table (JSONB, served by atlas-tenants per the configuration system) for the
`PartyOperation` / `BuddyList` writers, and the two new dispatcher YAML source-of-
truth files. Migration notes:

- New repo files: `docs/packets/dispatchers/party.yaml`,
  `docs/packets/dispatchers/buddy.yaml`.
- Seed-template change: v87/v95/jms `operations` maps for these writers gain the
  per-arm keys (generated, not hand-edited) where currently empty. v83/v84
  unchanged (already populated). This is a seed-template edit; LIVE tenants are
  not retro-patched by this task (per the
  `bug_new_opcodes_not_in_live_tenant_config` note — live patching is operational,
  not a code change), but the PRD records that live v87/v95/jms tenants will need
  the config patch + channel restart to use newly-split arms.

## 7. Service Impact

- `libs/atlas-packet/party` — discrete error structs, body functions, removal of
  the shared `Error` struct + `PartyErrorBody`. Tests updated/added.
- `libs/atlas-packet/buddy` — same for buddy.
- `tools/packet-audit` — `run.go` `candidatesFromFName` rewrite; new audit
  reports, fixtures, evidence; regenerated matrix; updated dispatcher-lint
  baseline.
- `services/atlas-channel` — call-site migration in
  `socket/handler/party_operation.go` and
  `kafka/consumer/{party,invite,buddylist}/consumer.go`.
- Seed templates (the configuration seed data consumed by atlas-tenants) —
  populated party/buddy `operations` maps for v87/v95/jms.

No other service changes; no Kafka topic/contract changes; no new endpoints.

## 8. Non-Functional Requirements

- **Correctness/grounding:** every mode byte IDA-verified per version; v95 shift
  read from the v95 switch, not inferred; version-absent arms confirmed by the
  absence of a case (CLAUDE.md "Verification Over Memory", "No Inventing").
- **Determinism:** no behavior change on the wire — a regression-style byte
  comparison (old shared `Error` output vs new discrete-struct output) must be
  byte-identical for every error currently emitted.
- **CI gates:** `dispatcher-lint`, `matrix --check`, `fname-doc --check`,
  `operations --check`, `redis-key-guard.sh`, `go vet ./...`,
  `go test -race ./...`, `go build ./...`, and `docker buildx bake` for any
  service whose `go.mod` was touched — all clean (CLAUDE.md Build & Verification).
- **Multi-tenancy:** mode resolution stays per-tenant via the `operations` table;
  no hard-coded bytes.
- **Observability:** no new logs required; the existing "ResolveCode 99 fallback"
  path remains the misconfiguration signal.

## 9. Open Questions

- Final naming convention for the per-arm body functions and structs (resolved in
  design — follow the guild.go naming, e.g. `PartyOperation*` / `Buddy*`).
- Whether any party error arm carries a body beyond `mode + name` (e.g. an
  Apprenticeship/expedition error variant) — to be settled by the FR-1.1
  decompile; the PRD assumes the current `mode + AsciiString` shape for the
  emitted arms and will add fields if the switch reveals them.
- Exact buddy `UNKNOWN_ERROR` extra-field semantics (the current `hasExtra` bool
  gates one trailing field) — confirm which mode(s) carry it from the
  `OnFriendResult` decompile.

## 10. Acceptance Criteria

- [ ] `docs/packets/dispatchers/party.yaml` and `buddy.yaml` exist, each with the
      writer/fname/op/direction header, the 5 IDA-verified function addresses, the
      v95-shift note, and a per-key per-version mode table — every value cited to a
      decompile line.
- [ ] One discrete struct per error arm for each family, in one consolidated
      `clientbound` file; each `Encode` writes the full arm body (no mode-only
      stub); the shared `Error` struct is deleted from both packages.
- [ ] `PartyErrorBody`/`BuddyErrorBody` removed; replaced by fixed-key per-arm
      body functions with no caller-supplied selector; every constructor takes
      `mode byte`; zero `mode: 0x` literals and zero `func(_ byte)` in body files.
- [ ] Every discrete struct is constructed by a body function (no orphans).
- [ ] `run.go` has one `#<Mode>` candidate entry per arm (no `#Error` catch-all);
      every `#`-entry resolves to an existing struct; no audit report cites a
      deleted file.
- [ ] Per-arm export entry + audit report + byte-fixture (with `// packet-audit:verify`
      marker) + evidence; `PARTY_OPERATION` and `BUDDYLIST` op-rows ✅ across
      gms_v83/v84/v87/v95 and jms_v185 (version-absent arms ⬜).
- [ ] v87/v95/jms `operations` tables populated for both writers (generated);
      `operations --check` exits 0.
- [ ] All `services/atlas-channel` call sites migrated; no string error selector
      remains; emitted bytes byte-identical to pre-migration for every error
      currently sent.
- [ ] Both families removed from `dispatcher-lint-baseline.yaml`; `exempt_families`
      is empty; `dispatcher-lint` exits 0 with no suppressed notes.
- [ ] `matrix --check`, `fname-doc --check`, `operations --check`,
      `go vet ./...`, `go test -race ./...`, `go build ./...`, and
      `docker buildx bake` for touched services all clean.
