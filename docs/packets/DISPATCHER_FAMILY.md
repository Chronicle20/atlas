# Implementing a mode-prefix dispatcher family

A **mode-prefix dispatcher** is one opcode whose first byte is a mode/discriminator
that the client `switch`es on, routing to N sub-handlers ("arms"), each reading a
different body. Examples: `CITC::OnNormalItemResult` (MTS), `CCashShop::OnCashItemResult`
(cash shop), `CShopDlg::OnPacket` (npc shop), `CTrunkDlg::OnPacket` (storage),
`CUIMessenger::OnPacket` (messenger), `CMiniRoomBaseDlg::OnPacketBase` (player
interaction), `CField::OnFieldEffect`, `CWvsContext::OnPartyResult`,
`CWvsContext::OnGuildResult`.

This file is the canonical recipe + the **enforced invariants**. It exists because
implementing these families went wrong repeatedly (task-096); the failures and their
prevention are recorded so the next family is done right the first time.

> **`matrix ✅` is NOT "family complete."** The matrix grades **codec byte-correctness**.
> It does not check discrete-per-mode, config-driven mode resolution, footgun-free
> APIs, or whether a feature can even *use* the codec. Those are the invariants below,
> enforced by `packet-audit dispatcher-lint` (CI-gated) and this checklist — not by the
> matrix.

## The canonical pattern

Reference implementations to copy: `libs/atlas-packet/npc/clientbound/conversation.go`
(discrete per-type structs) and `libs/atlas-packet/field/clientbound/mts_operation.go`
(discrete per-mode structs) + `libs/atlas-packet/field/mts_operation_body.go`
(per-mode body functions). Also `messenger/`, `storage/`, `field_effect`.

For **each mode the family supports**:

1. **One discrete struct, in the family's single consolidated clientbound file**
   (e.g. `mts_operation.go`, not a `*_result_*_modes.go` sprawl). The struct holds
   `mode byte` + that arm's body fields. Bodyless notice/error arms are still their
   own discrete struct (`struct { mode byte }`) — discrete means *discrete*, even when
   two arms share a wire shape.

2. **The struct's `Encode` writes the mode byte then the full arm body**, every field
   cited to a decompile line. A struct that writes only the mode byte for an arm that
   has a body is a false pass.

3. **A per-mode body function** (in `<pkg>/operation_body.go` or `<pkg>/<family>_body.go`)
   that FIXES the operation key and resolves the per-version mode byte from the tenant
   `operations` table:

   ```go
   func XxxYyyDoneBody(/* arm data params, NO mode/op/code selector */) func(logrus.FieldLogger, context.Context) func(map[string]interface{}) []byte {
       return atlas_packet.WithResolvedCode("operations", XxxYyyDone /* fixed key const */, func(mode byte) packet.Encoder {
           return clientbound.NewXxxYyyDone(mode /* resolved */, /* arm data */)
       })
   }
   ```
   The constructor takes `mode byte` (first param); the body func passes the **resolved**
   mode through. The key is a hard-coded const matching the `operations` table /
   `docs/packets/dispatchers/<family>.yaml`.

4. **`run.go` `candidatesFromFName`**: one `case "<Fname>#<Mode>":` per supported mode,
   returning that mode's discrete struct `{name, pkg, dir: clientbound}`.

5. **Per-mode verification** (see `VERIFYING_A_PACKET.md`): synthetic `#`-suffixed
   export entry, audit report, byte-fixture with a `// packet-audit:verify` marker, and
   evidence where the grader needs it. The op-row aggregates worst-of all arms and
   reaches ✅ only when every supported arm is verified (the FIELD_EFFECT model — the
   family is NOT in `families.yaml`).

6. **Wire it to a feature** (a consumer/handler calls the body function) — or, if no
   feature emits it yet, the body-function layer is the usable API a future feature
   calls. A verified codec with no body function is unusable.

## Anti-patterns — these are BANNED (each is a real task-096 failure)

`dispatcher-lint` fails on these; do not reintroduce them.

| # | Anti-pattern | Why it's wrong | Linted by |
|---|---|---|---|
| AP-1 | **Shared-by-shape struct** — one struct serving >1 mode (`MtsResultEmpty` for 19 modes) | obscures which mode; can't map mode↔definition; can't grade per-mode | INV-1 |
| AP-2 | **Hard-coded mode byte** in a struct constructor (`mode: 0x1E`) | bypasses the per-tenant `operations` table; wrong byte if a tenant remaps or a version differs | INV-2 |
| AP-3 | **Discarding the resolved code** — `WithResolvedCode("operations", KEY, func(_ byte)…)` | resolves the mode then ignores it (same as AP-2) | INV-2 |
| AP-4 | **Caller-specified mode** — a body func taking `op/code/mode/key string` (or `*Mode`) | lets a caller send the wrong mode → client crash; the packet maps to ONE op, so fix the key | INV-3 |
| AP-5 | **Dangling candidate / phantom representative** — a `#`-entry or report pointing at a deleted struct/file | freezes a stale cell; the next report-gen silently skips it | INV-4 |
| AP-6 | **Orphaned codec** — a verified struct no body function constructs | "verified" but no feature can ever send it; `matrix ✅` hides it | INV-5 |
| AP-7 | **Mode-byte-only stub** for an arm that has a body | the "passes because we only read 1 byte" false pass | checklist (not statically lintable) |
| AP-8 | **File sprawl** — `<family>_result_<shape>_modes.go` proliferation | one family → one clientbound struct file | convention |

## Enforced invariants (`packet-audit dispatcher-lint`)

Run `go run ./tools/packet-audit dispatcher-lint` (CI-gated alongside `matrix --check`).
It scans every dispatcher family (those with `#`-suffixed `candidatesFromFName` entries)
except those baselined in `docs/packets/dispatcher-lint-baseline.yaml`:

- **INV-1** no clientbound struct is mapped by >1 dispatcher `#`-entry.
- **INV-2** no `mode:\s*0x` literal in a dispatcher struct constructor; no `func(_ byte)`
  in a body-function file (must be `func(mode byte)` passthrough).
- **INV-3** no exported `*Body(` function lets the caller pick the operation.
  Two complementary signals: (a) by-name — a param literally named
  `op`/`code`/`mode`/`key`; (b) **semantic** — a param of ANY name (e.g.
  `errorCode`, `reason`) that flows into the `WithResolvedCode("operations", …)`
  key. A fixed const, a `string(Const)` cast, or a string literal is fine; a
  *parameter* as the key is the AP-4 footgun. The semantic signal exists because
  the by-name list let `BuddyErrorBody(errorCode string)` escape (task-101).
- **INV-4** every `#`-entry candidate resolves to an existing `type <name> struct`; no
  committed audit report cites a deleted Atlas file.
- **INV-5** every dispatcher clientbound struct is constructed by a body function.

## Client-table values INSIDE bodies (not just mode bytes)

INV-2/INV-3 cover the dispatcher MODE byte. The same rule extends to any value
inside an arm's body that the client interprets through its own lookup switch
— e.g. the `CITC::NoticeFailReason` codes read by the mode-24 reason-notice
arm (task-102). These MUST also be config-resolved from a tenant writer
options table, per-version, never Go literals — "IDA-verified identical
across versions" does not exempt them (task-103 uniformity ruling).

Pattern (see atlas-channel `kafka/consumer/mts/consumer.go` `failNoticeOr` +
`noticeFailReasons` in the gms seed templates):

- The DOMAIN service emits a SEMANTIC key (string, e.g. `NOT_ENOUGH_NX`) on
  its Kafka event — domain services never speak client bytes (the WishOrigin
  layering).
- The CHANNEL resolves the key against a writer-options table. For OPTIONAL
  tables, soft-resolve with a fallback to the bare/legacy arm on a missing
  table or key — never let `ResolveCode`'s 99-on-miss reach the client.
- The table lands in EVERY supported version's seed template, and the
  feature's rollout notes call out the live-tenant patch (seed templates
  never retroactively apply — bug_new_opcodes_not_in_live_tenant_config).

The **baseline** (`docs/packets/dispatcher-lint-baseline.yaml`) lists families
not yet migrated. It is now **empty** (`exempt_families: []`) — guild (task-103),
message (task-104), party, and buddy (task-105) all graduated to discrete-per-mode.
The linter fails on any violation **outside** the baseline (i.e. any violation at
all now). Migrating a family = removing its baseline entry; the baseline only
shrinks and has reached empty — new families must be authored discrete-per-mode
from the start.

Separately, `docs/packets/evidence/families.yaml` (the mode-prefix membership list
that caps an op at the `🧩` `family` state) is also empty — every dispatcher arm
graduated (task-096) — so `🧩` currently caps **no** op. A newly-added family that
should cap needs an explicit `families.yaml` entry.

## Serverbound dispatcher files are out of scope for FAM-CAP

**What FAM-CAP models.** The family cap (`docs/packets/evidence/families.yaml`,
enforced by `dispatcher-lint`'s FAM-CAP check) exists for a CLIENTBOUND
mode-prefix demultiplexer: one opcode whose leading `switch(Decode1())` fans
out to N arms that each read a DIFFERENT body. A single byte fixture only
proves the one arm it exercises, so an un-migrated family is capped at
`🧩` rather than allowed to read `✅` off one arm's evidence.

**Why serverbound is out of scope.** A serverbound
`docs/packets/dispatchers/*.yaml` is not that shape — it is the source of
truth for a HANDLER's `options.operations` ROUTING table, i.e. the mode byte
Atlas must map to a handler key. The modes are written by N INDEPENDENT
client call sites (the client has no single request builder that could be
"only partially proven" by one fixture), and each one already carries its
own discrete struct with its own `packet-audit:fname` marker. There is no
client demultiplexer to cap.

**Why capping anyway would be harmful.** `grade.go` applies
`in.Families[baseFName(ref.FName)]` per-op with **no direction check**, so
adding a serverbound fname to `families.yaml` would demote already
byte-fixture-verified cells from `✅ verified` down to `🧩 family`. This was
measured, not estimated: adding task-206's two coupon/cash-shop serverbound
fnames to `families.yaml` demoted **17 cells** (10 `COUPON_CODE` + 7
`CASHSHOP_OPERATION`, all serverbound) — which is why the check now skips
`direction: serverbound` files instead.

**The behaviour is fail-safe.** Only an explicit `direction: serverbound`
opts a file out. A dispatcher file that omits `direction:` (the clientbound
default) is still fully guarded — verified by
`TestFamilyCapServerboundSkipped` in
`tools/packet-audit/cmd/family_cap_test.go`, which asserts a clientbound
sibling in the same directory still fails FAM-CAP while the serverbound file
(with or without a `fname:`) is silently skipped.

**Precedent.** `docs/packets/evidence/families.yaml` already carries a note
that serverbound `RPS_ACTION` is deliberately unlisted for the same reason;
this section documents the rule the comment implements so it isn't only
discoverable by reading `tools/packet-audit/cmd/dispatcher_lint.go`.

### Known limitations (the linter's blind spots)

The linter only sees a family it is *enrolled* on — i.e. a base FName with **>1**
`#`-suffixed clientbound entry in `candidatesFromFName`. Two consequences:

- **A catch-all `#`-entry can hide AP-1 from INV-1.** INV-1 fires only when a
  struct is mapped by >1 `#`-entry. A single catch-all arm (e.g.
  `CWvsContext::OnPartyResult#Error` fronting ~15 error sub-ops through one
  `Error` struct) maps the struct by ONE entry, so INV-1 stays silent. In
  practice these catch-alls route through a string-keyed body func and INV-3
  (semantic) catches them — but a catch-all constructed some other way would
  slip through. When you see a `#`-entry comment admitting "sub-op enum deferred
  to _pending.md", that arm is a catch-all: split it.
- **A mode-dispatcher writer not enrolled as a family is invisible.** Example
  (latent, task-101): `libs/atlas-packet/pet/clientbound/activated_body.go`
  `PetDespawnBody(…, reason string)` passes `reason` as the operations key — the
  same AP-4 footgun as the error bodies — but `pet` has no multi-arm `#`-entry
  family in `run.go`, so the linter never scans it. Enrolling pet (split the
  despawn modes into `#`-entries) would bring it under INV-3. Tracked as a
  future "wire or migrate" item, not fixed here.

## "Family complete" checklist

Before claiming a dispatcher family done (mirror this in the PR description):

- [ ] One discrete struct per supported mode, in ONE consolidated clientbound file.
- [ ] Each struct's `Encode` writes the **full arm body** (not just the mode byte), every field decompile-cited.
- [ ] Every constructor takes `mode byte`; every body func resolves it via `WithResolvedCode("operations", FIXED_KEY, func(mode byte)…)` — **zero** `mode: 0x` literals, **zero** `func(_ byte)`.
- [ ] No body func takes a caller-supplied op/code/mode/key selector.
- [ ] No struct serves >1 mode; no dangling `#`-entry / report; no orphaned struct.
- [ ] Per-mode `#`-entry + export entry + audit report + byte-fixture(marker) + evidence; op-row ✅ across applicable versions (version-absent → ⬜).
- [ ] `dispatcher-lint`, `matrix --check`, `fname-doc --check`, `operations --check` all exit 0; `go build/vet/test -race` clean.
- [ ] Family removed from `dispatcher-lint-baseline.yaml` (if it was there).
