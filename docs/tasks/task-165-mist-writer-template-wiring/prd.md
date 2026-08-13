# Mist Broadcast Writer Wiring — Product Requirements Document

Version: v2
Status: Draft
Created: 2026-07-10
Revised: 2026-08-07 — rescoped to **all supported versions** (11 seed templates, 9 matrix columns) after the branch was brought current with main.
---

## 1. Overview

Mist (affected-area) broadcasts are silently dropped for every tenant. The clientbound writers `AffectedAreaCreated` and `AffectedAreaRemoved` are fully implemented in `libs/atlas-packet/field/clientbound/` (SPAWN_MIST / REMOVE_MIST), and the atlas-channel mist consumer (`services/atlas-channel/atlas.com/channel/kafka/consumer/mist/consumer.go`) already translates `EVENT_TOPIC_MIST` created/destroyed events into these packets — but the name→opcode resolution chain is broken in two places, so `session.Announce` fails with "writer not found" and nothing reaches the client:

1. **Go side:** `produceWriters()` in `services/atlas-channel/atlas.com/channel/main.go` does not declare either writer name, so `BuildWriterProducer` (`libs/atlas-opcodes/producer.go`) filters them out of the writer map even when config supplies an opcode.
2. **Config side:** no seed template registers either writer under `socket.writers` — verified 2026-08-07 across **all eleven** templates in `services/atlas-configurations/seed-data/templates/` (`grep -c AffectedArea` = 0 in every file).

The user-visible symptom: mob AREA_POISON mists tick invisibly (players take poison damage from an area they cannot see), and player-cast mist skills never render for anyone. Mist lifecycle is owned by atlas-maps (`services/atlas-maps/atlas.com/maps/mist/`), with atlas-monsters emitting for mob skills — those producers and the channel consumer are correct and unchanged; this task is purely the writer wiring, matrix verification, and rollout to existing tenants.

This is the same `routedElsewhere && !routed` template-wiring gap family as prior fixes.

### 1.1 What changed since v1 of this PRD

The v1 PRD scoped five versions (gms_83/84/87/95/jms_185) and declared gms_92/gms_12 out of scope for lack of an IDB. Both premises are now stale:

- **Four more matrix columns exist.** `packet-audit` now tracks nine versions (`tools/packet-audit/internal/matrix/model.go:14`): gms_v48, v61, v72, v79, v83, v84, v87, v95, jms_v185. The gms_61/72/79 bring-up (task-113) landed while this branch sat idle.
- **SPAWN_MIST is already ✅ on v61/v72/v79**, and **REMOVE_MIST is only 🟡ᶠ there** — the inverse of the v83+ situation. The verification gap moved rather than shrank.
- **gms_92 and gms_12 now have packet-named IDBs** (~950 and ~230 functions respectively), so "no IDB to source opcodes from" no longer holds. They remain the only two templates with no packet-audit column.
- **The v48 mist opcodes are undiscovered, not absent.** `CAffectedAreaPool::OnAffectedAreaCreated/Removed` resolve to `unresolved: true` / "function not found in IDB" placeholders in `docs/packets/ida-exports/gms_v48.json`, and the v48 registry is a self-declared "clean PARTIAL" with deep pool leaves deferred. There is **no** positive-absence evidence, so ⬜ is not currently justified as a ruling.

## 2. Goals

Primary goals:
- Register `AffectedAreaCreated` and `AffectedAreaRemoved` in atlas-channel's declared writer list (`produceWriters()`).
- Register both writers with their per-version opcodes in **every seed template whose opcode is IDB-sourced** — the eight in §4 FR-2 today, plus any of gms_48/gms_92/gms_12 that FR-2b discovery resolves.
- Close the packet-coverage gap for the mist pair across all nine matrix columns: SPAWN_MIST ❌→✅ on v83/v84/v87/v95/jms_185, REMOVE_MIST 🟡ᶠ→✅ on v61/v72/v79, and replace the length-only v79 SPAWN_MIST fixture with a real byte pin.
- Patch live tenant configurations for all existing tenants on every wired version and restart atlas-channel, so the fix reaches running environments (seed templates apply only at tenant creation).
- End-to-end verification in a deploy environment: mists visibly render.

Non-goals:
- No changes to the packet codecs (`affected_area_created.go` / `affected_area_removed.go`) for any **already-wire-verified** version. The `tStart` gate (`Region()=="GMS" && MajorVersion()>=95`) is frozen for v61–jms_185. If v48/v92/v12 discovery reveals a divergent layout, extending the codec with a new version gate is in scope (FR-2b), but it must not change bytes on any version that is already ✅.
- No changes to the mist consumer, atlas-maps mist domain, or atlas-monsters producers.
- **No derived-unverified opcodes.** A version is wired only when its opcode is read out of that version's IDB. Interpolating an opcode band from neighbouring versions (the task-088 summon precedent) is explicitly rejected here — a wrong opcode routes a packet the client silently drops, which is indistinguishable from the bug being fixed.
- No mist gameplay logic (damage ticks, durations) — server-side behavior already works; this is broadcast visibility only.

## 3. User Stories

- As a player, I want to see the poison cloud a mob casts so that I can move out of it instead of taking damage from an invisible area.
- As a player casting a mist skill, I want my mist to render for me and everyone else on the map so that the skill is usable.
- As an operator, I want existing tenants (not just newly created ones) to receive the fix so that live environments render mists after rollout.
- As an operator running a legacy-version tenant (v48/v61/v72/v79) or a reduced-template tenant (v92/v12), I want mists to work there too, rather than only on the modern versions.
- As a packet-audit maintainer, I want both mist ops fixture-verified across every tracked version so that the matrix reflects reality and future opcode reshifts are caught.

## 4. Functional Requirements

### FR-1 Go writer declaration (atlas-channel)

- Add `fieldcb.AffectedAreaCreatedWriter` and `fieldcb.AffectedAreaRemovedWriter` (constants `"AffectedAreaCreated"` / `"AffectedAreaRemoved"` from `libs/atlas-packet/field/clientbound`) to the `produceWriters()` slice in `services/atlas-channel/atlas.com/channel/main.go:608`.
- No other Go changes. The consumer already calls `session.Announce(...)(fieldpkt.AffectedAreaCreatedWriter)` / `(fieldpkt.AffectedAreaRemovedWriter)`.
- Known side effect (acceptable): tenants on templates that end up unwired will log the existing warning `Service declares writer [X] but tenant config has no opcode mapping for it.` at startup, as they already do for many other declared writers.

This requirement is version-independent — one declaration serves all versions.

### FR-2 Seed template wiring — the eight IDB-sourced versions

Add two entries to the `socket.writers` array of each template below, using the entry shape `{"opCode": "0xNNN", "writer": "<name>"}` (no `options` — these writers take none). Opcodes come from the per-version registries under `docs/packets/registry/` and agree with `docs/packets/audits/STATUS.md:346-347`.

| Template | AffectedAreaCreated (SPAWN_MIST) | AffectedAreaRemoved (REMOVE_MIST) | Insert before |
|---|---|---|---|
| `template_gms_61_1.json` | 0x0D2 | 0x0D3 | `SpawnDoor` 0x0D4 |
| `template_gms_72_1.json` | 0x0F3 | 0x0F4 | `SpawnDoor` 0x0F5 |
| `template_gms_79_1.json` | 0x0FB | 0x0FC | `SpawnDoor` 0x0FD |
| `template_gms_83_1.json` | 0x111 | 0x112 | `SpawnDoor` 0x113 |
| `template_gms_84_1.json` | 0x118 | 0x119 | `SpawnDoor` 0x11A |
| `template_gms_87_1.json` | 0x122 | 0x123 | `SpawnDoor` 0x124 |
| `template_gms_95_1.json` | 0x148 | 0x149 | `SpawnDoor` 0x14A |
| `template_jms_185_1.json` | 0x126 | 0x127 | `SpawnDoor` 0x128 |

Placement is **ascending opcode order**, enforced by `tools/template-opcode-order-guard.sh` (`docs/packets/TEMPLATE_CONVENTIONS.md`). The layout is uniform across all eight: `DropDestroy` sits at (SPAWN_MIST − 4) and `SpawnDoor` at (SPAWN_MIST + 2), so both new entries slot immediately before the `SpawnDoor` entry in every file.

### FR-2b Opcode discovery for the three remaining templates (gms_48, gms_92, gms_12)

These three templates are in scope but have no sourced opcode today:

| Template | Blocker | Discovery anchor |
|---|---|---|
| `template_gms_48_1.json` | `CAffectedAreaPool::*` unnamed in the v48 IDB; `docs/packets/ida-exports/gms_v48.json` carries `unresolved: true` placeholders for both functions; registry is a "clean PARTIAL" with pool leaves deferred | v48 IDB session (`GMS_v48_1_DEVM.exe`). v61's `CAffectedAreaPool::OnPacket` @0x423eb7 is a two-arm dispatcher (`a2==210` → Created, `a2==211` → Removed); locate the v48 analogue by positional correlation off the named v61 pool dispatchers |
| `template_gms_92_1.json` | No registry, no matrix column, `CAffectedAreaPool::*` unnamed | v92 IDB session. Method of record is dispatcher positional correlation against the PDB-backed v95 IDB (`CField::OnPacket` v92=0x5406b0 / v95=0x546d50); opcode shift is irregular — align by structure, never by opcode number |
| `template_gms_12_1.json` | No registry, no matrix column, no open IDB session | v12 IDB on disk; stage dispatcher `CField::OnPacket`@0x47502d. v12 opcode drift vs v48 is non-uniform — match by class + payload fingerprint, never by opcode number |

Requirements:

- For each of the three, run a bounded discovery pass to read the SPAWN_MIST / REMOVE_MIST opcode out of that version's IDB, naming the functions in the IDB as you go.
- **Outcome A — opcode found:** name the functions in the IDB, add the ops to that version's registry where one exists (v48), wire the template exactly as FR-2, and verify the codec's read order for that version (FR-3).
- **Outcome B — feature proven absent:** record positive absence evidence and leave the template unwired. For gms_48 (which has a matrix column) that means an entry in `docs/packets/feature-na-evidence.yaml` meeting its stated bar — "the opcode slot is a different op, a binary-wide search for the op's construction/gate found nothing, the receive handler lacks the feature branch". A failed name search is explicitly **not** evidence.
- **Outcome C — inconclusive:** leave the template unwired and record what was walked and what remains, in the task folder. Do **not** fall back to a derived/interpolated opcode (§2 non-goal).
- Standing up a full packet-audit column for gms_92/gms_12 is **not** required by this task — only the two opcodes.

### FR-3 Coverage-matrix completion for the mist pair

Follow `docs/packets/audits/VERIFYING_A_PACKET.md`. Target end state for both rows across all nine columns:

| Version | SPAWN_MIST now | REMOVE_MIST now | Work required |
|---|---|---|---|
| gms_v48 | ⬜ | ⬜ | Depends on FR-2b outcome: verify both if discovered; justified ⬜ with n-a evidence if proven absent |
| gms_v61 | ✅ | 🟡ᶠ | REMOVE_MIST byte fixture + evidence (`CAffectedAreaPool::OnAffectedAreaRemoved` @0x4246b0) |
| gms_v72 | ✅ | 🟡ᶠ | REMOVE_MIST byte fixture + evidence (@0x42ec4e) |
| gms_v79 | ✅ (soft) | 🟡ᶠ | REMOVE_MIST byte fixture + evidence (@0x42f0de) **and** SPAWN_MIST re-pin — see below |
| gms_v83 | ❌ | ✅ | SPAWN_MIST byte fixture + evidence |
| gms_v84 | ❌ | ✅ | SPAWN_MIST byte fixture + evidence |
| gms_v87 | ❌ | ✅ | SPAWN_MIST byte fixture + evidence |
| gms_v95 | ❌ | ✅ | SPAWN_MIST byte fixture + evidence (incl. the `tStart` field) |
| jms_v185 | ❌ | ✅ | SPAWN_MIST byte fixture + evidence (no `tStart`) |

**The v79 SPAWN_MIST ✅ is soft and must be hardened.** Its evidence record (`docs/packets/evidence/gms_v79/field.clientbound.FieldAffectedAreaCreated.yaml`) points at `TestAffectedAreaCreatedWireShape`, whose only v79 assertion is `len(b) != 39` — a length check, not a byte pin. That is the exact "passes on 1 byte"-family shortcut the design rejected. Re-point the v79 evidence at a real byte-pinning test.

Common requirements for every cell promoted:

- A `// packet-audit:verify packet=<packet> version=<ver> ida=<addr>` marker on a test that pins the **full expected wire body** against an explicit byte slice — not a length or an offset spot-check.
- Read order derived from the client decompile (ida-pro-mcp or the checked-in `docs/packets/ida-exports/`), per version.
- An evidence record under `docs/packets/evidence/<ver>/field.clientbound.FieldAffectedArea<Created|Removed>.yaml` with a tool-computed `decompile_sha256` — never hand-written or copied.
- `packet-audit matrix --check` exits 0 after regeneration. Baseline re-verified on this branch post-merge (2026-08-07): **exit 0**.
- If any per-version read order does not match the current codec on an already-✅ version, STOP and escalate — do not adjust the codec silently.

### FR-4 Live tenant rollout

Seed templates apply only at tenant creation; existing tenants must be patched directly (known bug pattern: new opcodes missing from live tenant config → packet silently dropped):

- For every existing tenant on **every wired version**, in each environment: add the two writer entries (with that tenant's version-correct opcodes from FR-2/FR-2b) to the channel service `socket.writers` configuration in atlas-tenants.
- Restart atlas-channel pods after patching — the configuration projection does not hot-reload writer maps.
- Document the exact patch payload/procedure in the task folder so it is reproducible per environment, with a row per version actually wired.

### FR-5 End-to-end verification

In a deploy environment (e.g. via the `deploy-env` PR label flow):

- A mob with an AREA_POISON mob skill casts it and the mist visibly renders on the client; it disappears when the mist expires.
- A player-cast mist skill renders for the caster and for a second observing client on the same map. (Identify a concrete test skill from repo/WZ data during execution — do not assume skill IDs from memory. Note the version-specific skill-ID caveat: a mist skill id valid on v83 is not necessarily the same id on v48/v61.)
- atlas-channel logs show no `Unable to broadcast AffectedArea...` errors and no `writer not found` for these writers on the patched tenant.
- E2E is required on **at least one modern version (v83+) and one legacy version (v61/v72/v79)** — the legacy path exercises a different opcode band and a different template generation. Per-version e2e beyond that is best-effort and recorded as such.

## 5. API Surface

No REST API changes. Config-data changes only:

- Seed template JSON: two new `socket.writers` entries × each wired template (FR-2, FR-2b).
- Live atlas-tenants channel configuration resource: same two entries per existing tenant (FR-4), applied via the existing tenant-configuration update path.

## 6. Data Model

No schema or entity changes. No migrations. The `socket.writers` array shape (`opCode`, `writer`, optional `options`) is unchanged.

## 7. Service Impact

- **atlas-channel** — one-line-per-writer addition to `produceWriters()` in `main.go`. Full verification gate applies: `go test -race ./...`, `go vet ./...`, `go build ./...`, and `docker buildx bake atlas-channel`.
- **atlas-configurations** — seed template JSON edits only; no Go changes. `tools/template-opcode-order-guard.sh` must pass.
- **libs/atlas-packet** — new byte-fixture tests + verify markers for `FieldAffectedAreaCreated` (5–6 versions) and `FieldAffectedAreaRemoved` (3 versions); test files only, no codec changes unless FR-2b Outcome A requires a new version gate. `go test -race ./...` in the lib module.
- **tools/packet-audit** — registry addition for gms_v48 only if FR-2b discovers the op; otherwise consumed read-only.
- **atlas-maps / atlas-monsters** — no changes (producers already correct).

## 8. Non-Functional Requirements

- **Multi-tenancy:** opcodes are per-version tenant config, never hard-coded in Go (DOM-25 posture); the Go change only declares writer names.
- **Observability:** rely on existing log seams — the `BuildWriterProducer` startup warning enumerates declared-but-unmapped writers per tenant; consumer error logs (`Unable to broadcast AffectedArea...`) must be absent post-rollout.
- **Performance:** no new hot-path work; the broadcast path (`ForSessionsInMap` + `session.Announce`) is the same one used by all field broadcasts.
- **Safety:** `packet-audit matrix --check`, `tools/template-opcode-order-guard.sh`, and `tools/template-movement-types-guard.sh` all exit 0 after the change.

## 9. Open Questions

- **FR-2b outcomes are genuinely unknown.** Whether gms_48, gms_92, and gms_12 get wired depends on what the discovery passes find. Each resolves to Outcome A (wire it), B (n-a evidence), or C (documented inconclusive) during execution — the task is not blocked on answering them up front, but the PRD's acceptance criteria are conditional on the outcome per version.
- Everything else was pinned during the PRD interview and the 2026-08-07 rescope.

## 10. Acceptance Criteria

- [ ] `produceWriters()` in atlas-channel `main.go` includes `AffectedAreaCreatedWriter` and `AffectedAreaRemovedWriter`.
- [ ] All eight FR-2 templates contain both writer entries with the exact opcodes in the FR-2 table; `grep -c '"AffectedAreaCreated"'` returns 1 per wired template (and `"AffectedAreaRemoved"` likewise); every file still parses as JSON.
- [ ] Each of gms_48 / gms_92 / gms_12 has a recorded FR-2b outcome: wired with an IDB-sourced opcode (A), carrying positive absence evidence (B), or documented inconclusive with the walk recorded (C). No template carries a derived/interpolated opcode.
- [ ] `docs/packets/audits/STATUS.md` SPAWN_MIST row is ✅ for v83/v84/v87/v95/JMS185 and remains ✅ for v61/v72/v79; REMOVE_MIST row is ✅ for v61/v72/v79 and remains ✅ for v83+. Every ✅ in both rows is backed by a full-body byte fixture, including a re-pinned v79 SPAWN_MIST.
- [ ] `packet-audit matrix --check` exits 0; `tools/template-opcode-order-guard.sh` and `tools/template-movement-types-guard.sh` exit 0.
- [ ] `go test -race ./...`, `go vet ./...`, `go build ./...` clean in atlas-channel, atlas-configurations, and libs/atlas-packet; `tools/redis-key-guard.sh` clean; `docker buildx bake atlas-channel` succeeds from the worktree root.
- [ ] Live tenant configs patched for every existing tenant on every wired version (procedure documented in the task folder); atlas-channel restarted.
- [ ] Deploy-env verification on at least one modern and one legacy version: mob AREA_POISON mist renders and expires visibly; a player mist skill renders for caster and observer; no `writer not found` / `Unable to broadcast AffectedArea` log lines.
