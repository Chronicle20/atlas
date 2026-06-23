# Party + Buddy Dispatcher Family — Design

Task: task-105-party-buddy-dispatcher-family
Status: Draft for review
Created: 2026-06-23
Companion to: `prd.md` (approved), `docs/packets/DISPATCHER_FAMILY.md` (governing pattern),
task-103 (`guild`) / task-104 (`message`) designs (the executed playbook this mirrors)

---

## 1. Problem & Current State (grounded)

`party` (`PARTY_OPERATION` → `CWvsContext::OnPartyResult`) and `buddy` (`BUDDYLIST` →
`CWvsContext::OnFriendResult`) are the **last two** mode-prefix dispatcher families still
baselined in `docs/packets/dispatcher-lint-baseline.yaml`:

```yaml
exempt_families:
  - CWvsContext::OnPartyResult
  - CWvsContext::OnFriendResult
```

Both already have discrete structs for their *non-error* arms; only the **error arms**
remain folded into a single shared catch-all struct selected by a caller string — the
task-096 footgun the canonical pattern bans. Verified findings from the current tree:

| Concern | Evidence (file:line) | Invariant |
|---|---|---|
| `PartyErrorBody(code string, name string)` — caller-supplied operation selector | `libs/atlas-packet/party/clientbound/operation_body.go:78-82` | AP-4 / INV-3 |
| Party `Error` struct (`mode + AsciiString name`) fronts every party error arm | `party/clientbound/error.go:13-47`; run.go `#Error` `cmd/run.go:1373` | AP-1 / AP-7 catch-all |
| `BuddyErrorBody(errorCode string)` — caller-supplied selector named `errorCode` (escaped the original by-name INV-3 check; caught by task-101 semantic hardening) | `libs/atlas-packet/buddy/operation_body.go:50-55` | AP-4 / INV-3 |
| Buddy `Error` struct (`mode` + optional trailing byte via `hasExtra`) fronts ~10 buddy error arms | `buddy/clientbound/error.go:15-50`; run.go `#Error` `cmd/run.go:1130` | AP-1 / AP-7 catch-all |
| No `docs/packets/dispatchers/party.yaml` or `buddy.yaml` source-of-truth file exists | `ls docs/packets/dispatchers/` | FR-5.1 gap |

Call sites that pass a string selector into the catch-all (FR-6):

- `socket/handler/party_operation.go:97` → `PartyErrorBody("UNABLE_TO_FIND_THE_CHARACTER", sp.Name())`
- `socket/handler/party_operation.go:106` → `PartyErrorBody("UNABLE_TO_FIND_THE_REQUESTED_CHARACTER_IN_THIS_CHANNEL", sp.Name())`
- `kafka/consumer/invite/consumer.go:171` → `PartyErrorBody("HAVE_DENIED_REQUEST_TO_THE_PARTY", targetName)`
- `kafka/consumer/party/consumer.go:452` → `PartyErrorBody(errorType, name)` — **runtime** selector
- `kafka/consumer/buddylist/consumer.go:238` → `BuddyErrorBody(errorCode)` — **runtime** selector

The non-error party arms (`Created`/`Disband`/`Left`/`Join`/`Update`/`ChangeLeader`/`Invite`/
`TownPortal`) and non-error buddy arms (`ListUpdate`/`BuddyUpdate`/`Invite`/`ChannelChange`/
`CapacityUpdate`) are already discrete with fixed-key body funcs and per-arm `#`-entries —
out of scope beyond what the split mechanically touches (PRD §2 non-goals).

**Governing rule (`DISPATCHER_FAMILY.md`):** `matrix ✅` means codec byte-correct, nothing
more. Discrete-per-mode shape, config-driven mode resolution, footgun-free APIs, codec
usability, operations-table population, and honest IDA grounding are *separate* requirements
proven by the §8 gates, not by a green cell.

## 2. Goal / Definition of Done

Split both error catch-alls into discrete per-mode structs, replace
`PartyErrorBody`/`BuddyErrorBody` with fixed-key per-arm body functions, author
`party.yaml`/`buddy.yaml`, drive every supported arm to ✅ across `gms_v83`, `gms_v84`,
`gms_v87`, `gms_v95`, `jms_v185` (version-absent arms ⬜), populate the empty v87/v95/jms
`operations` tables, migrate all call sites, and remove **both** families from the
dispatcher-lint baseline so `exempt_families` is empty and the `dispatcher-family`
campaign is complete. All §8 gates exit 0.

## 3. Architecture — the canonical pattern applied to party & buddy

Copy the migrated exemplars (guild task-103, message task-104, `field_effect`): discrete
per-mode structs in one consolidated `clientbound` file + per-mode fixed-key body
functions + one dispatcher YAML per family. The shape:

```
libs/atlas-packet/party/
├── clientbound/
│   ├── error.go                 # REPLACED: one discrete struct per OnPartyResult error arm
│   │                            #   (Error struct + NewError removed once all arms split)
│   ├── operation_body.go        # PartyErrorBody removed; one fixed-key body func per error arm
│   └── {created,disband,…}.go   # unchanged non-error arms
libs/atlas-packet/buddy/
├── clientbound/
│   └── error.go                 # REPLACED: one discrete struct per OnFriendResult error arm
│                                #   (Error struct + NewBuddyError removed once all arms split)
├── operation_body.go            # BuddyErrorBody removed; one fixed-key body func per error arm
└── {operation,…}.go             # unchanged non-error arms
docs/packets/dispatchers/
├── party.yaml                   # NEW — per-version OnPartyResult mode table (source of truth)
└── buddy.yaml                   # NEW — per-version OnFriendResult mode table (source of truth)
tools/packet-audit/cmd/run.go    # #Error catch-all → one #<Mode> entry per error arm (both families)
```

**Package-layout asymmetry (preserve each family's existing convention — do not normalize):**
party's body functions live in package `clientbound`
(`party/clientbound/operation_body.go`); buddy's live in the parent package `buddy`
(`buddy/operation_body.go`, calling `clientbound.New*`). The split keeps each family where
it already is; harmonizing the two layouts is a separate, unrequested refactor (YAGNI).

Data flow per error arm (identical to the exemplar):

```
handler/consumer → Party<Arm>Body(arm data)  /  Buddy<Arm>Body(arm data)
   → WithResolvedCode("operations", <FIXED_KEY const>,        resolves mode byte
        func(mode byte) → clientbound.New<Arm>(mode, data))    from tenant operations table
   → struct.Encode writes mode byte + full arm body            (clientbound/error.go)
```

## 4. Key Design Decisions (alternatives + tradeoffs)

### D1 — Split each shared `Error` struct into one discrete struct per mode

`party/clientbound/error.go` serves every party error arm through `Error{mode, name}`;
`buddy/clientbound/error.go` serves ~10 buddy error arms through `Error{mode, hasExtra}`.
`DISPATCHER_FAMILY.md` (AP-1, AP-8): "Bodyless notice/error arms are still their own
discrete struct (`struct { mode byte }`) — discrete means *discrete*, even when two arms
share a wire shape."

- **Chosen:** one discrete struct per supported error arm, all in that family's single
  consolidated `clientbound/error.go` (AP-8: one family → one struct file, no `*_modes.go`
  sprawl). Each struct's `Encode` writes the mode byte then the **full arm body** verified
  from the decompile (FR-2.3): a party arm that carries a name →
  `struct{ mode byte; name string }`; a mode-only party/buddy arm →
  `struct{ mode byte }`; the buddy arm with the trailing extra int →
  `struct{ mode byte }` whose `Encode` writes the extra byte (no `hasExtra` flag — the
  arm identity *is* the struct). The exact arm set is the IDA-enumerated `OnPartyResult` /
  `OnFriendResult` switch (§4 / FR-2.1, FR-2.2), not invented from the operations-key list.
- **Rejected — keep the shared struct, add fixed-key body funcs only:** body funcs pass
  INV-3, but one struct mapped by >1 `#`-entry fails INV-1 and is the exact AP-1 the doc
  bans; it also defeats per-mode matrix grading.
- **Rejected — codegen the bodyless structs:** no codegen precedent in the packet lib;
  guild/message hand-wrote theirs. Hand-write for reviewability and per-struct decompile
  citation. Tradeoff: `error.go` grows, but stays one file (AP-8 holds).

Naming derives from the operation-key semantics so `#`-entry, struct, body func, and
operations key line up — e.g. `PartyUnableToFindCharacter`, `PartyRequestDenied`;
`BuddyListFull`, `BuddyAlreadyBuddy`, `BuddyUnknownError`. Final names settled against the
existing operation-key consts during execution.

### D2 — Replace `PartyErrorBody` / `BuddyErrorBody` with per-error-mode body functions

Both let the caller pick the mode (AP-4) and are the live INV-3 violations. Remove both.
Per error arm, add a fixed-key body func taking only that arm's body data:

```go
func PartyUnableToFindCharacterBody(name string) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
    return atlas_packet.WithResolvedCode("operations", PartyOperationUnableToFindCharacter, func(mode byte) packet.Encoder {
        return clientbound.NewPartyUnableToFindCharacter(mode, name)
    })
}

func BuddyListFullBody() func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
    return atlas_packet.WithResolvedCode("operations", BuddyOperationErrorListFull, func(mode byte) packet.Encoder {
        return clientbound.NewBuddyListFull(mode)
    })
}
```

Every constructor takes `mode byte` first; the body func passes the **resolved** mode
through `func(mode byte)` — never `func(_ byte)`, never a `mode: 0x..` literal (INV-2). No
body func takes any parameter that flows into the `WithResolvedCode` key (INV-3 semantic).
Per FR-3.4 / DISPATCHER_FAMILY.md step 6, arms Atlas does not currently emit still get a
body func so the codec is a usable API (the future-feature entry points) — every discrete
struct is constructed by ≥1 body func (no orphan, INV-5).

### D3 — Runtime `errorType` / `errorCode` → specific body func: explicit switch at the call site

Two sites pass a **runtime** string into the catch-all (`consumer/party:452` `errorType`,
`consumer/buddylist:238` `errorCode`). FR-6.1/6.2 require mapping that runtime value to a
specific body func, *not* threading it into a generic one.

- **Chosen:** an explicit `switch`/lookup **at the call site** (in the consumer) mapping
  the runtime value → the matching `Party<Arm>Body(...)` / `Buddy<Arm>Body(...)`, with a
  logged default for an unrecognized value (the misconfiguration signal; never silently
  emit a wrong arm). The packet lib stays free of the runtime error enum — the body funcs
  expose only fixed keys, and the runtime→key translation is the consumer's concern (it
  already owns the upstream Kafka contract that produced the string).
- **Rejected — a `map[string]bodyFunc` exported from the packet lib:** re-introduces a
  string→mode indirection inside `libs/atlas-packet`, the exact AP-4 surface we're deleting;
  one typo'd map key resurrects the footgun.
- **Rejected — keep a single generic body func that switches internally:** that *is*
  `PartyErrorBody`/`BuddyErrorBody` renamed; INV-3 still fails.

The three literal-string sites (`party_operation.go:97,106`, `invite/consumer.go:171`)
become direct one-to-one calls to the matching body func — no switch needed.

### D4 — Remove the `#Error` catch-all; one `#`-entry per supported mode

`candidatesFromFName` returns the shared `Error` struct for `CWvsContext::OnPartyResult#Error`
(run.go:1373) and `CWvsContext::OnFriendResult#Error` (run.go:1130). Per FR-4.1 / INV-4:
delete both catch-all cases and add one `case "CWvsContext::OnPartyResult#<Mode>":` /
`"#<Mode>":` per supported arm, each returning that arm's discrete struct
`{name, pkg, dir: clientbound}`. No `#`-entry may point at a deleted struct; no audit
report may cite the removed `Error` file (AP-5 / INV-4). The non-error arms' existing
`#`-entries are unchanged.

### D5 — `party.yaml` / `buddy.yaml` as the per-version source of truth

Create both in the `guild.yaml` format: `writer`, `fname`, `op`, `direction`, and an
`operations:` list of `{ key, modes: { gms_v83, gms_v84, gms_v87, gms_v95, jms_v185 } }`.
Each mode value is **IDA-verified per version** (FR-1.5), cited to that version's decompiled
switch case/line — never folded from a sibling, never from general MapleStory knowledge.
The header records the 5 per-version function addresses (FR-1.1/1.2), the v95 non-uniform
shift note (FR-1.3 — read each v95 arm from the v95 switch and cross-check the decrypted
StringPool message where the arm shows one, same family as the opcode-table/guild bug),
and any version-absent arm (FR-1.4 — omit that version's mode, matrix cell ⬜, mirroring
guild's jms `SET_SKILL_RESPONSE` precedent). The existing v83/v84 templates already carry
the currently-emitted arms' modes and must continue to match this file.

### D6 — Populate the empty v87/v95/jms `operations` tables (the guild gap)

Per `bug_operations_mode_tables_missing_v87_v95_jms` (guild hit this in task-103): the
v87/v95/jms seed templates likely carry an **empty** `operations` map for these writers, so
`operations --check` will report MISSING for every v87/v95/jms key until populated. After
authoring the yamls, run `packet-audit operations` (generate) to populate those tables from
the new yamls; v83/v84 stay unchanged (already populated) and must continue to match
(FR-5.2). `operations --check` exits 0 for both families across all applicable versions
afterward (FR-5.3). Per `bug_new_opcodes_not_in_live_tenant_config`, this is a *seed-template*
edit; LIVE v87/v95/jms tenants are not retro-patched by this task — the PRD records that
they will need the config patch + channel restart to use newly-split arms (operational, not
a code change in this task's scope).

### D7 — Grounding workflow (IDA per version)

Per FR-1 and project rules: resolve the IDB via `list_instances`/`select_instance`
(gms_v83 :13342, gms_v84 :13337, gms_v87 :13341, gms_v95 :13340, jms_v185 :13339), confirm
the version + function address before reading, decompile the actual `OnPartyResult` /
`OnFriendResult` switch, and cite function+address in each struct/test comment. v84 has no
distinct switch divergence expected (v84 ≡ v83 structurally per task-083) but is **read,
not assumed**; any divergence is folded in with per-arm v84 fixtures. An unresolved fname
is a **stop-and-ask** — never an auto-re-export, substituted fname, or faked hash
(`feedback_unresolved_fname_escalate`).

### D8 — Verification: per-arm fixtures + worst-of aggregation

Per FR-4.2/4.3: each arm gets a synthetic `#`-suffixed export entry, an audit report, a
byte-fixture with a `// packet-audit:verify` marker (expected bytes hand-computed from the
IDA read order, IDA-cited in the comment), and pinned evidence where the grader requires it
(`VERIFYING_A_PACKET.md`). The `PARTY_OPERATION` / `BUDDYLIST` op-rows aggregate worst-of
all arms and reach ✅ for a version only when **every** supported arm for that version is
verified — the FIELD_EFFECT model; the family is **not** added to `families.yaml`.
Determinism gate (NFR): a regression byte-comparison of the old shared-`Error` output vs the
new discrete-struct output must be **byte-identical** for every error currently emitted.

## 5. Component-by-component scope

- **`libs/atlas-packet/party/clientbound/error.go`** — replace shared `Error` with discrete
  structs (one per OnPartyResult error arm); each `Encode` mode + full body, decompile-cited;
  delete `Error`/`NewError` when fully split.
- **`libs/atlas-packet/party/clientbound/operation_body.go`** — remove `PartyErrorBody`; add
  per-arm fixed-key body funcs.
- **`libs/atlas-packet/buddy/clientbound/error.go`** — replace shared `Error` with discrete
  structs (one per OnFriendResult error arm, incl. the trailing-extra-int arm); delete
  `Error`/`NewBuddyError` when fully split.
- **`libs/atlas-packet/buddy/operation_body.go`** — remove `BuddyErrorBody`; add per-arm
  fixed-key body funcs.
- **`libs/atlas-packet/{party,buddy}/clientbound/*_test.go`** — per-arm byte fixtures with
  `// packet-audit:verify` markers + IDA citations; regression byte-comparison vs old output.
- **`tools/packet-audit/cmd/run.go`** — `#Error` catch-all → one `#<Mode>` entry per arm,
  both families.
- **`docs/packets/dispatchers/party.yaml` + `buddy.yaml`** — new per-version mode tables.
- **`docs/packets/dispatcher-lint-baseline.yaml`** — remove **both** entries;
  `exempt_families` becomes empty.
- **`docs/packets/audits/STATUS.md` + `status.json`** — regenerated (toolSha stamp).
- **Seed templates** (`gms_87`/`gms_95`/`jms_185`) — populated party/buddy `operations` maps
  (generated, not hand-edited); v83/v84 unchanged.
- **`services/atlas-channel`** — call-site migration:
  `socket/handler/party_operation.go:97,106`, `kafka/consumer/invite/consumer.go:171`
  (literal → direct body func); `kafka/consumer/party/consumer.go:452`,
  `kafka/consumer/buddylist/consumer.go:238` (runtime → call-site switch, D3). No string
  error selector remains anywhere (FR-6.3).

No other service changes; no Kafka topic/contract changes; no new endpoints; no REST surface.

## 6. Testing strategy

- **Per-arm byte fixtures** (clientbound): expected bytes hand-computed from the IDA read
  order, `// packet-audit:verify` marker, IDA citation in the comment; one per supported
  version (a version is ⬜ only when the arm is genuinely version-absent, else ✅).
- **Regression guard:** for every error Atlas currently emits, a byte-comparison test
  proves the new discrete-struct output equals the old shared-`Error` output (NFR
  determinism — no wire change).
- **Tooling gates:** `dispatcher-lint`, `matrix --check`, `fname-doc --check`,
  `operations --check` all exit 0 with an **empty** baseline.
- **Build/test:** `go build ./...`, `go vet ./...`, `go test -race ./...` clean in every
  changed module (`libs/atlas-packet`, `tools/packet-audit`, `services/atlas-channel`);
  `tools/redis-key-guard.sh` clean; `docker buildx bake atlas-channel` from the worktree
  root (its `go.mod` is the only service `go.mod` touched — `libs/atlas-packet` and
  `tools/packet-audit` are not bake targets, but confirm during execution).

## 7. Execution phasing (for the plan phase)

1. **Enumerate** — decompile `OnPartyResult` + `OnFriendResult` switches per version;
   author `party.yaml`/`buddy.yaml`; reconcile against existing operation-key consts;
   resolve the §8 open questions (arm set, name-bearing arms, buddy extra-int arm).
2. **Party split** — discrete structs + per-arm body funcs; remove `PartyErrorBody`.
3. **Buddy split** — discrete structs + per-arm body funcs; remove `BuddyErrorBody`.
4. **run.go rewire** — per-mode `#`-entries; remove both `#Error` catch-alls.
5. **Operations tables** — generate v87/v95/jms maps from the yamls; `operations --check` 0.
6. **Fixtures + matrix** — per-arm byte fixtures all five versions + regression compare;
   regenerate STATUS.
7. **Wiring** — atlas-channel call sites → per-arm bodies (literal direct; runtime switch).
8. **De-baseline + gates** — remove both families from the baseline (now empty); all four
   packet-audit checks + build/vet/test/bake/redis-guard green.
9. **Review + PR** — modular reviewer agents, then PR mirroring the family-complete
   checklist; CI green on PR HEAD. (Campaign-complete: baseline empty.)

## 8. Open questions (resolved during execution, not now)

- **Exact supported-arm count** behind each catch-all — set by the IDA enumeration of the
  `OnPartyResult` / `OnFriendResult` switch (drives the struct count).
- **Which party error arms carry a body beyond `mode`** — the current emitted shape is
  `mode + AsciiString name`; confirm per arm from the decompile (PRD §9), adding fields if
  the switch reveals them (e.g. an apprenticeship/expedition variant).
- **Buddy `UNKNOWN_ERROR` extra-field semantics** — which mode(s) carry the trailing int
  (current `hasExtra` gates one trailing byte for `UNKNOWN_ERROR`); confirm from the
  `OnFriendResult` decompile (PRD §9).
- **Version-absent vs present per arm** — resolved per-arm against each IDB (absent → ⬜).
- **v95 non-uniform shift** — each v95 mode read from the v95 switch + StringPool
  cross-check, never folded from v83 (D5).

## 9. Out of scope (from PRD §2 non-goals)

The latent `pet` `PetDespawnBody(reason string)` AP-4 footgun (separate unenrolled family);
`CWvsContext::OnAllianceResult` (unimplemented, not baselined); any new party/buddy gameplay
behavior, error conditions, or error messages; changing the already-discrete non-error arms
beyond what the split mechanically requires; normalizing the party-vs-buddy package-layout
asymmetry (D3 of §3); live-tenant config patching (operational, recorded not executed here).
