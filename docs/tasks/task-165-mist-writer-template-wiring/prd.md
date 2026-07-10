# Mist Broadcast Writer Wiring — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-07-10
---

## 1. Overview

Mist (affected-area) broadcasts are silently dropped for every tenant. The clientbound writers `AffectedAreaCreated` and `AffectedAreaRemoved` are fully implemented and IDA-audited in `libs/atlas-packet/field/clientbound/` (SPAWN_MIST / REMOVE_MIST), and the atlas-channel mist consumer (`services/atlas-channel/atlas.com/channel/kafka/consumer/mist/consumer.go`) already translates `EVENT_TOPIC_MIST` created/destroyed events into these packets — but the name→opcode resolution chain is broken in two places, so `session.Announce` fails with "writer not found" and nothing reaches the client:

1. **Go side:** `produceWriters()` in `services/atlas-channel/atlas.com/channel/main.go` does not declare either writer name, so `BuildWriterProducer` (`libs/atlas-opcodes/producer.go`) filters them out of the writer map even when config supplies an opcode.
2. **Config side:** no seed template registers either writer under `socket.writers` (confirmed across all seven templates in `services/atlas-configurations/seed-data/templates/`).

The user-visible symptom: mob AREA_POISON mists tick invisibly (players take poison damage from an area they cannot see), and player-cast mist skills never render for anyone. Mist lifecycle is owned by atlas-maps (`services/atlas-maps/atlas.com/maps/mist/`), with atlas-monsters emitting for mob skills — those producers and the channel consumer are correct and unchanged; this task is purely the writer wiring, matrix verification of the SPAWN_MIST codec, and rollout to existing tenants.

This is the same `routedElsewhere && !routed` template-wiring gap family as prior fixes. Additionally, SPAWN_MIST is ❌ in the packet coverage matrix for all five audited versions (`docs/packets/audits/STATUS.md:367`) — the codec has matched audit records (read-order verified against IDA, all rows Verdict 0) but no `packet-audit:verify` byte fixtures or evidence pins. REMOVE_MIST is already ✅ across all five. This task closes that verification gap too, so the wiring lands backed by fixture-verified codecs.

## 2. Goals

Primary goals:
- Register `AffectedAreaCreated` and `AffectedAreaRemoved` in atlas-channel's declared writer list (`produceWriters()`).
- Register both writers with their per-version opcodes in the five full seed templates (gms_83, gms_84, gms_87, gms_95, jms_185).
- Promote SPAWN_MIST (`field/clientbound/FieldAffectedAreaCreated`) from ❌ to ✅ for all five versions in the coverage matrix (byte-fixture tests + verify markers + evidence records + regenerated STATUS.md).
- Patch live tenant configurations for all existing tenants on the five versions and restart atlas-channel, so the fix reaches running environments (seed templates apply only at tenant creation).
- End-to-end verification in a deploy environment: mists visibly render.

Non-goals:
- No changes to the packet codecs (`affected_area_created.go` / `affected_area_removed.go`) — the v83-vs-v95 layout divergence was already fixed (abs-RECT layout, `tStart` gated `GMS>=95`).
- No changes to the mist consumer, atlas-maps mist domain, or atlas-monsters producers.
- No wiring for the reduced templates `gms_92` and `gms_12` — they have no packet registry or IDB to source/verify opcodes from (same posture as the parked v92 mount-food handler).
- No mist gameplay logic (damage ticks, durations) — server-side behavior already works; this is broadcast visibility only.

## 3. User Stories

- As a player, I want to see the poison cloud a mob casts so that I can move out of it instead of taking damage from an invisible area.
- As a player casting a mist skill, I want my mist to render for me and everyone else on the map so that the skill is usable.
- As an operator, I want existing tenants (not just newly created ones) to receive the fix so that live environments render mists after rollout.
- As a packet-audit maintainer, I want SPAWN_MIST fixture-verified across all five versions so that the matrix reflects reality and future opcode reshifts are caught.

## 4. Functional Requirements

### FR-1 Go writer declaration (atlas-channel)

- Add `fieldcb.AffectedAreaCreatedWriter` and `fieldcb.AffectedAreaRemovedWriter` (constants `"AffectedAreaCreated"` / `"AffectedAreaRemoved"` from `libs/atlas-packet/field/clientbound`) to the `produceWriters()` slice in `services/atlas-channel/atlas.com/channel/main.go`.
- No other Go changes. The consumer already calls `session.Announce(...)(fieldpkt.AffectedAreaCreatedWriter)` / `(fieldpkt.AffectedAreaRemovedWriter)`.
- Known side effect (acceptable): tenants on reduced templates (gms_92/gms_12) will log the existing warning `Service declares writer [X] but tenant config has no opcode mapping for it.` at startup, as they already do for ~140 other declared writers.

### FR-2 Seed template wiring (atlas-configurations)

Add two entries to the `socket.writers` array of each of the five full templates in `services/atlas-configurations/seed-data/templates/`, using the entry shape `{"opCode": "0xNNN", "writer": "<name>"}` (no `options` — these writers take none). Opcodes are pinned by `docs/packets/audits/STATUS.md:367,369` and the per-version registries:

| Template | AffectedAreaCreated (SPAWN_MIST) | AffectedAreaRemoved (REMOVE_MIST) |
|---|---|---|
| `template_gms_83_1.json` | 0x111 | 0x112 |
| `template_gms_84_1.json` | 0x118 | 0x119 |
| `template_gms_87_1.json` | 0x122 | 0x123 |
| `template_gms_95_1.json` | 0x148 | 0x149 |
| `template_jms_185_1.json` | 0x126 | 0x127 |

Place entries in each template's existing opcode-order position (match surrounding file convention).

### FR-3 SPAWN_MIST matrix promotion (❌ → ✅ ×5)

Follow `docs/packets/audits/VERIFYING_A_PACKET.md` for `field/clientbound/FieldAffectedAreaCreated` on each of gms_v83, gms_v84, gms_v87, gms_v95, jms_v185:

- Byte-fixture test(s) in `libs/atlas-packet/field/clientbound/affected_area_test.go` (or sibling file per convention) with a `// packet-audit:verify packet=field/clientbound/FieldAffectedAreaCreated version=<ver> ida=<addr>` marker per version, mirroring the existing five REMOVE_MIST markers at `affected_area_test.go:102-106`.
- Read order derived from the client decompile (ida-pro-mcp or the checked-in exports in `docs/packets/ida-exports/`). Existing matched audit records at `docs/packets/audits/<ver>/FieldAffectedAreaCreated.json` (e.g. v83 address `0x431a63`) are the starting point, but fixtures must be derived from the actual per-version read order, including the `tStart` `GMS>=95` gate (v95/jms fixtures must include it; v83/v84/v87 must not).
- Pin evidence records under `docs/packets/evidence/<ver>/field.clientbound.FieldAffectedAreaCreated.yaml` (the Removed evidence files already exist as the shape reference).
- Regenerate the matrix; `STATUS.md` row SPAWN_MIST shows ✅ for all five versions.
- If any per-version read order does not match the current codec, STOP and escalate — do not adjust the codec silently (that would contradict the "no codec changes" boundary and needs a design decision).

### FR-4 Live tenant rollout

Seed templates apply only at tenant creation; existing tenants must be patched directly (known bug pattern: new opcodes missing from live tenant config → packet silently dropped):

- For every existing tenant on the five affected versions, in each environment: add the two writer entries (with that tenant's version-correct opcodes from FR-2) to the channel service `socket.writers` configuration in atlas-tenants.
- Restart atlas-channel pods after patching — the configuration projection does not hot-reload writer maps.
- Document the exact patch payload/procedure in the task folder so it is reproducible per environment.

### FR-5 End-to-end verification

In a deploy environment (e.g. via the `deploy-env` PR label flow):

- A mob with an AREA_POISON mob skill casts it and the mist visibly renders on the client; it disappears when the mist expires.
- A player-cast mist skill renders for the caster and for a second observing client on the same map. (Identify a concrete test skill from repo/WZ data during execution — do not assume skill IDs from memory.)
- atlas-channel logs show no `Unable to broadcast AffectedArea...` errors and no `writer not found` for these writers on the patched tenant.

## 5. API Surface

No REST API changes. Config-data changes only:

- Seed template JSON: two new `socket.writers` entries ×5 templates (FR-2).
- Live atlas-tenants channel configuration resource: same two entries per existing tenant (FR-4), applied via the existing tenant-configuration update path.

## 6. Data Model

No schema or entity changes. No migrations. The `socket.writers` array shape (`opCode`, `writer`, optional `options`) is unchanged.

## 7. Service Impact

- **atlas-channel** — one-line-per-writer addition to `produceWriters()` in `main.go`. Full verification gate applies: `go test -race ./...`, `go vet ./...`, `go build ./...`, and `docker buildx bake atlas-channel` (its `go.mod` module is touched by workspace resolution; bake is mandatory per repo policy).
- **atlas-configurations** — seed template JSON edits only; no Go changes.
- **libs/atlas-packet** — new byte-fixture tests + verify markers for `FieldAffectedAreaCreated` (test files only; no codec changes). `go test -race ./...` in the lib module.
- **atlas-maps / atlas-monsters** — no changes (producers already correct).

## 8. Non-Functional Requirements

- **Multi-tenancy:** opcodes are per-version tenant config, never hard-coded in Go (DOM-25 posture); the Go change only declares writer names.
- **Observability:** rely on existing log seams — the `BuildWriterProducer` startup warning enumerates declared-but-unmapped writers per tenant; consumer error logs (`Unable to broadcast AffectedArea...`) must be absent post-rollout.
- **Performance:** no new hot-path work; the broadcast path (`ForSessionsInMap` + `session.Announce`) is the same one used by all field broadcasts.
- **Safety:** template-wiring conflicts must be clean — `packet-audit matrix --check` (and the fname-doc/operations checks if applicable) exit 0 after the change.

## 9. Open Questions

None. Resolved during interview: gms_92/gms_12 are out of scope; live tenant patching is in scope; SPAWN_MIST matrix promotion is in scope; end-to-end rendering in a deploy environment is the acceptance bar.

## 10. Acceptance Criteria

- [ ] `produceWriters()` in atlas-channel `main.go` includes `AffectedAreaCreatedWriter` and `AffectedAreaRemovedWriter`.
- [ ] All five full seed templates contain both writer entries with the exact opcodes in the FR-2 table; `grep '"AffectedAreaCreated"'` matches once per template (and `"AffectedAreaRemoved"` likewise).
- [ ] `docs/packets/audits/STATUS.md` SPAWN_MIST row is ✅ for v83/v84/v87/v95/JMS185, backed by five `packet-audit:verify` markers and five evidence records; REMOVE_MIST row remains ✅.
- [ ] `packet-audit matrix --check` exits 0; the SPAWN_MIST template-wiring conflict is gone.
- [ ] `go test -race ./...`, `go vet ./...`, `go build ./...` clean in atlas-channel and libs/atlas-packet; `tools/redis-key-guard.sh` clean; `docker buildx bake atlas-channel` succeeds from repo root.
- [ ] Live tenant configs patched for every existing tenant on the five versions (procedure documented in the task folder); atlas-channel restarted.
- [ ] Deploy-env verification: mob AREA_POISON mist renders and expires visibly; a player mist skill renders for caster and observer; no `writer not found` / `Unable to broadcast AffectedArea` log lines.
