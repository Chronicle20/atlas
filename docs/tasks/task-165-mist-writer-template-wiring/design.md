# Mist Broadcast Writer Wiring — Design

Task: task-165-mist-writer-template-wiring
Status: Approved PRD → design
Created: 2026-07-10

---

## 1. Problem recap (one paragraph)

The `AffectedAreaCreated` / `AffectedAreaRemoved` clientbound writers are fully
implemented and IDA-audited in `libs/atlas-packet/field/clientbound/`, and the
atlas-channel mist consumer already emits them — but the name→opcode resolution
chain is broken in two independent places: `produceWriters()` in
`services/atlas-channel/atlas.com/channel/main.go` never declares either name
(so `BuildWriterProducer` filters them out of the writer map), and no seed
template registers them under `socket.writers`. Every mist broadcast dies in
`session.Announce` with "writer not found". This design covers the two wiring
fixes, the SPAWN_MIST matrix promotion (❌→✅ ×5), live-tenant rollout, and
end-to-end verification. No codec, consumer, or producer changes.

## 2. Architecture context (what already works)

Data flow, verified against source:

```
atlas-maps mist domain ──┐
atlas-monsters (mob AoE) ─┴─> EVENT_TOPIC_MIST (created/destroyed)
        │
        v
atlas-channel kafka/consumer/mist/consumer.go
        │  handleMistCreated  → fieldpkt.NewAffectedAreaCreated(...)
        │  handleMistDestroyed → fieldpkt.NewAffectedAreaRemoved(...)
        v
_map.ForSessionsInMap(f, session.Announce(...)(wp)(WriterName)(body.Encode))
        │
        v  wp lookup: WriterName → opcode   ← BROKEN LINK (both sides)
client socket
```

The only broken link is the `wp` lookup. `BuildWriterProducer`
(`libs/atlas-opcodes/producer.go`) intersects two inputs:

1. the service's declared writer list (`produceWriters()` — Go, name only), and
2. the tenant's `socket.writers` config (name → opCode, from seed template at
   tenant creation).

A name absent from either side is silently dropped (startup warning only).
Both sides are missing here. This is the known
`routedElsewhere && !routed` template-wiring gap family.

## 3. Design decisions

### 3.1 FR-1 — Go writer declaration (no alternatives worth taking)

Add two entries to the `produceWriters()` slice
(`services/atlas-channel/atlas.com/channel/main.go:608`):

- `fieldcb.AffectedAreaCreatedWriter` ("AffectedAreaCreated")
- `fieldcb.AffectedAreaRemovedWriter` ("AffectedAreaRemoved")

Placement: append near the other `fieldcb.*` field writers (the list is not
strictly ordered; match the surrounding cluster). No other Go change — the
consumer already references the constants, so there is no import or call-site
work.

Accepted side effect: tenants on reduced templates (gms_92/gms_12) gain one
more line each in the existing startup warning enumerating
declared-but-unmapped writers. This is the same posture as ~140 other writers
and needs no mitigation.

### 3.2 FR-2 — Seed template wiring

Add two `{"opCode": "0xNNN", "writer": "<name>"}` entries (no `options` — the
`Encode` closures ignore the options map) to `socket.writers` in each of the
five full templates in `services/atlas-configurations/seed-data/templates/`:

| Template | AffectedAreaCreated | AffectedAreaRemoved |
|---|---|---|
| `template_gms_83_1.json` | 0x111 | 0x112 |
| `template_gms_84_1.json` | 0x118 | 0x119 |
| `template_gms_87_1.json` | 0x122 | 0x123 |
| `template_gms_95_1.json` | 0x148 | 0x149 |
| `template_jms_185_1.json` | 0x126 | 0x127 |

Opcodes are pinned by `docs/packets/audits/STATUS.md` rows SPAWN_MIST (:367)
and REMOVE_MIST (:369) — the per-version registries already agree; no new
opcode discovery is needed.

Placement: numeric opcode order within each template's `socket.writers` array.
Verified in `template_gms_83_1.json`: the array is opcode-sorted and currently
jumps to `SpawnDoor` at 0x113 with no 0x111/0x112 entries — the two new
entries slot immediately before it. Same rule per template (find the first
entry with a higher opcode, insert before).

### 3.3 FR-3 — SPAWN_MIST matrix promotion: extend the existing test file

**Current state (verified):**

- `affected_area_test.go` already has strong Created tests:
  `TestAffectedAreaCreatedWireShape` (39/43-byte totals for v83/v87/JMS185/v95)
  and `TestAffectedAreaCreatedFields` (per-field offsets, abs-RECT math, v83 +
  v95). Neither carries `packet-audit:verify` markers, v84 is untested, and
  neither pins a full expected-bytes buffer.
- REMOVE_MIST is the in-file template: `TestAffectedAreaRemovedByteOutput`
  pins the full wire body across all five versions with five markers
  (`affected_area_test.go:102-106`) and five evidence YAMLs.
- Matched audit records exist for all five versions with all rows Verdict 0:
  v83 `0x431a63`, v84 `0x4326ca`, v87 `0x432f3f`, v95 `0x437ec0`,
  jms `0x436572` (`docs/packets/audits/<ver>/FieldAffectedAreaCreated.json`).

**Approaches considered:**

- **A (chosen): add one `TestAffectedAreaCreatedByteOutput` mirroring the
  Removed pattern** — a single test with per-version subtests, each comparing
  the full encoded body against an explicit expected byte slice
  (`want := []byte{...}`), with the five verify markers in its doc comment.
  Keeps SPAWN_MIST and REMOVE_MIST verification side by side in one file, one
  convention, and the marker `verifies` reference is a single test name per
  version.
- **B (rejected): retrofit markers onto the existing WireShape/Fields tests** —
  those tests assert lengths and individual offsets, not a pinned full-body
  buffer. The playbook's bar (§5–6) is a derived expected-bytes fixture;
  length assertions are exactly the "passes on 1 byte"-family shortcut the
  project has been burned by. The existing tests stay as-is (they still guard
  the RECT math and version gates); the new byte-output test is additive.
- **C (rejected): per-version sibling test files** — no precedent in this
  package; the sibling convention is one `<packet>_test.go` per codec pair.

**Per-version read order** (from the checked-in audit records; each execution
step re-derives from the decompile per the playbook, using ida-pro-mcp or
`docs/packets/ida-exports/`):

- v83/v84/v87/jms185 (39 bytes): dwId(4) nType(4) dwOwnerId(4) nSkillID(4)
  nSLV(1) phase(2) rcArea(16, abs LTRB) tEnd(4).
- v95 (43 bytes): same with tStart(4) inserted before tEnd.

**PRD correction (resolved here, no escalation needed):** PRD FR-3 line 68
says "v95/jms fixtures must include [tStart]". That contradicts both the codec
gate (`Region()=="GMS" && MajorVersion()>=95`,
`affected_area_created.go:92`) and the jms_v185 audit record, whose tEnd row
explicitly notes "NO tStart". The PRD's own §2 non-goal states the gate as
`GMS>=95`, which excludes JMS. Design ruling: **jms_v185 fixtures must NOT
include tStart**; only gms_v95 does. If the execution-time decompile of
jms_v185 contradicts this (i.e. shows a tStart read), that triggers the FR-3
STOP-and-escalate clause, since fixing it would be a codec change.

**Evidence records:** five new files
`docs/packets/evidence/<ver>/field.clientbound.FieldAffectedAreaCreated.yaml`,
shaped exactly like the existing Removed evidence (packet, direction, version,
`category: TIER1-FIXTURE`, ida function/address/decompile_sha256, `verifies:`
pointing at `affected_area_test.go#TestAffectedAreaCreatedByteOutput`). The
`decompile_sha256` must be computed from the actual decompile text used for
derivation — never copied or guessed. IDA addresses come from the audit
records above; if a decompile at that address doesn't resolve or hash cleanly,
that is a stop-and-ask per the unresolved-fname rule.

**Matrix regen:** `go run ./tools/packet-audit matrix` then
`matrix --check`; SPAWN_MIST row flips to ✅ ×5 and the template-wiring
conflict clears (FR-2 supplies the template routes). Exit-0 required.

### 3.4 FR-4 — Live tenant rollout

Seed templates only apply at tenant creation (known bug pattern:
`bug_new_opcodes_not_in_live_tenant_config`), so existing tenants are patched
directly:

1. **Enumerate** tenants per environment via atlas-tenants REST; select those
   on the five affected versions (region+majorVersion from the tenant model).
2. **Patch** each tenant's channel `socket.writers` configuration resource:
   read-modify-write the existing configuration JSON, appending the two
   version-correct entries (idempotency guard: skip if `"AffectedAreaCreated"`
   already present). Applied via the existing tenant-configuration update path
   (same UI/REST surface used for prior opcode rollouts). Remember the
   JSON:API envelope requirement for raw REST calls.
3. **Restart** atlas-channel pods per environment after patching — the
   configuration projection does not hot-reload writer maps.
4. **Document** the exact per-version patch payload and the enumerate → patch
   → restart → verify sequence in
   `docs/tasks/task-165-mist-writer-template-wiring/rollout.md` so it is
   reproducible per environment. Repo-relative paths and placeholder tenant
   ids only.

Alternative rejected: a one-off migration script in atlas-configurations that
rewrites stored tenant configs. Heavier than needed for two entries × a
handful of tenants, adds code that runs once, and prior rollouts of this exact
family were done via config PATCH + restart. The documented manual procedure
is the established pattern.

Full sweep, not spot-check: the patch step enumerates **every** tenant on the
five versions in each target environment; the rollout doc records which
tenants were patched.

### 3.5 FR-5 — End-to-end verification

In a deploy environment (`deploy-env` PR label flow):

- **Mob mist:** trigger a mob with an AREA_POISON mob skill; mist renders and
  disappears on expiry. Concrete mob/map chosen at execution time from WZ/repo
  data — not from memory.
- **Player mist:** a mist skill cast renders for caster and a second observer
  on the same map. Concrete skill id likewise resolved from repo/WZ data at
  execution time (the codec test uses 2121006 as a plausible fixture input,
  but the e2e skill must be re-verified against WZ before use).
- **Logs:** atlas-channel shows no `writer not found` for either writer and no
  `Unable to broadcast AffectedArea...` lines on the patched tenant; the
  startup writer-map warning no longer lists either name for patched tenants.

## 4. Component/change inventory

| Area | File(s) | Change |
|---|---|---|
| atlas-channel | `services/atlas-channel/atlas.com/channel/main.go` | +2 lines in `produceWriters()` |
| atlas-configurations | 5× `seed-data/templates/template_*.json` | +2 `socket.writers` entries each, opcode-ordered |
| libs/atlas-packet | `field/clientbound/affected_area_test.go` | +1 byte-output test (5 subtests) + 5 verify markers |
| docs/packets | `evidence/<ver>/field.clientbound.FieldAffectedAreaCreated.yaml` ×5 | new evidence pins |
| docs/packets | `audits/STATUS.md` | regenerated (SPAWN_MIST ✅ ×5) |
| task docs | `docs/tasks/task-165-mist-writer-template-wiring/rollout.md` | rollout procedure + per-env record |
| live config | atlas-tenants channel config per tenant | +2 writer entries; channel restart |

Explicitly unchanged: both codec files, the mist consumer, atlas-maps mist
domain, atlas-monsters producers, gms_92/gms_12 reduced templates.

## 5. Sequencing

1. FR-3 first (fixtures + evidence + matrix regen) — verification before
   wiring, so the wiring lands on fixture-verified codecs and any read-order
   surprise (the STOP clause) fires before templates are touched.
2. FR-1 + FR-2 together (Go declaration + templates), then
   `packet-audit matrix --check` exit 0.
3. Build gate: `go test -race ./...` + `go vet ./...` in libs/atlas-packet and
   atlas-channel; `go build ./...` in atlas-channel;
   `docker buildx bake atlas-channel` from the worktree root;
   `tools/redis-key-guard.sh`.
4. Code review, PR with `deploy-env` label.
5. FR-4 rollout (patch + restart) in the deploy environment, then FR-5 e2e.

## 6. Error handling & failure modes

- **Read order ≠ codec for any version (FR-3):** STOP and escalate — codec
  changes are out of scope by PRD non-goal. Expected not to fire (all audit
  rows are Verdict 0), but it is the designated exit.
- **jms tStart ambiguity:** ruled in §3.3; only a contradicting jms decompile
  reopens it, via the same STOP clause.
- **Decompile/evidence hash trouble:** unresolved address or export mismatch
  is a stop-and-ask (unresolved-fname rule), never a substituted hash.
- **Tenant patch validation rejection:** full-sweep enumeration and
  read-modify-write (not blind overwrite) prevent partial/clobbered configs;
  idempotency guard makes re-runs safe.
- **"Fix didn't work" in deploy env:** first check the running image contains
  the change and the tenant config contains the entries (known first-step:
  verify binary/config before re-debugging), then pod logs.

## 7. Testing strategy

- **Unit/byte level:** new `TestAffectedAreaCreatedByteOutput` (full-body pins
  ×5 versions incl. v84, which is currently untested); existing WireShape /
  Fields / Removed tests unchanged and still passing.
- **Machine checks:** `packet-audit matrix --check` exit 0; verify markers ↔
  evidence ↔ STATUS.md consistency is enforced by the tool, not prose.
- **Service level:** existing mist consumer tests (broadcaster seam) are
  untouched; the writer-map change is config+declaration only and is covered
  by the e2e pass.
- **E2E:** §3.5 in the deploy environment — the acceptance bar is visible
  rendering, absence of the two error log lines, both mob-cast and
  player-cast paths.

## 8. Open questions

None. The PRD's v95/jms tStart parenthetical is resolved in §3.3 (jms has no
tStart); everything else was pinned during the PRD interview.
