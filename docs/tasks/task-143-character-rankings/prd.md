# Login-Screen Character Rankings — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-07-09
---

## 1. Overview

The MapleStory character-select screen displays a per-character info board that includes the character's overall rank and job rank within their world, each with an up/stay/down movement arrow. Atlas currently sends zeros for all four rank fields — the packet encoding already exists (`libs/atlas-packet/model/character_list_entry.go` writes rank/rankMove/jobRank/jobRankMove), but `services/atlas-login/atlas.com/login/character/model.go` hardcodes all four getters to `return 0`, and no service computes or stores rankings.

This task introduces a new **atlas-rankings** service that periodically computes per-world character rankings (overall and per job category) for each tenant, persists them, and exposes them over REST. atlas-login consumes the rankings when building the character list, replacing the hardcoded zeros. The service is deliberately built as a standalone rankings domain — not a login-internal feature — so it can later grow to serve website leaderboards (level, fame, guild GP, etc.) without rework.

Client behavior is IDA-verified for v83 (`MapleStory_dump.exe`, char-info fade-window constructor at `0x60292F`): the client gates the rank display on `rank != 0 && jobRank != 0`. When either is zero it renders a shorter info board with a "Ranking not available" string; when both are non-zero it renders "Ranked at %d" rows with arrows selected by the sign of rankMove/jobRankMove. The rankEnabled byte in the packet is not stored by the client — decode either reads the 16 rank bytes or memsets them to zero, so zeros and "rank disabled" are indistinguishable to the renderer. This means sending zeros for unranked/new characters is safe and degrades gracefully.

## 2. Goals

Primary goals:
- Characters on the select screen show real overall rank and job rank with correct movement arrows, updated periodically server-side.
- Rankings are computed per tenant, per world, from authoritative character data (level/exp), excluding GM characters.
- Recompute cadence is tenant-configurable.
- atlas-login degrades gracefully (zeros → client shows "Ranking not available") when rankings are missing or the rankings service is unavailable.
- The rankings service is a standalone, extensible domain service (future: website leaderboards for level, fame, guild GP).

Non-goals:
- In-game rank commands or notifications (Cosmic's `RankingCommandTask` equivalent).
- Website / atlas-ui leaderboard pages (future work the service should not preclude, but no UI ships in this task).
- Real-time / event-per-level-up rank updates; rankings refresh only on the periodic recompute.
- Cosmic's last-login-aware rankMove carry-over semantics (see FR-5 for the simplified semantics chosen).
- Packet changes (`libs/atlas-packet` encoding already exists).
- Fame/meso tiebreaks (Cosmic orders by `level DESC, exp DESC, lastExpGainTime ASC, fame DESC, meso DESC`; Atlas uses level/exp only — see FR-1).

## 3. User Stories

- As a player, I want to see my character's overall rank and job rank on the character-select screen so that I can compare my progress against other players in my world.
- As a player, I want up/down arrows showing how my rank changed since the last update so that I can see recent movement.
- As a player with a brand-new character, I want the info board to show "Ranking not available" rather than a bogus rank 0.
- As a server operator, I want to configure how often rankings recompute per tenant so that I can trade freshness against load.
- As a server operator, I want GM characters excluded from rankings so that staff characters don't pollute the leaderboard.

## 4. Functional Requirements

### 4.1 Ranking computation (atlas-rankings)

- **FR-1 (Overall rank):** For each tenant and each world, compute a dense 1-based ranking of all eligible characters ordered by `level DESC, exp DESC`, with `characterId ASC` as the final deterministic tiebreak. Rank 1 is the highest character.
- **FR-2 (Job rank):** For each tenant, world, and job category, compute the same ordering restricted to characters in that category. Job category is `jobId / 100` (Cosmic parity: 0 = beginner, 1 = warrior, 2 = magician, 3 = bowman, 4 = thief, 5 = pirate; Cygnus/Aran categories follow the same division for tenants whose version has them). Use `libs/atlas-constants/job` types — no locally invented job constants (DOM-21).
- **FR-3 (GM exclusion):** Characters flagged as GM are excluded from both rankings entirely (not ranked, and not counted when ranking others).
- **FR-4 (Cadence):** Recompute runs on a per-tenant configurable interval (default 60 minutes, matching Cosmic's `RANKING_INTERVAL`). The interval is read from tenant configuration (atlas-tenants); a tenant without explicit configuration uses the default. Configuration changes take effect without a service redeploy (exact mechanism — projection vs. re-read — decided in design).
- **FR-5 (Rank movement):** On each recompute, `rankMove = previousRank − newRank` (positive = moved up, negative = moved down, 0 = unchanged). Same formula for `jobRankMove` against the previous job rank. A character with no previous ranking entry gets move 0. Movement does not accumulate across cycles and is not gated on player login (deliberate simplification vs. Cosmic).
- **FR-6 (Character lifecycle):** Characters created since the last cycle appear in the next recompute. Deleted characters drop out no later than the next recompute. Stale entries for deleted characters must not be served with non-zero ranks indefinitely.
- **FR-7 (Multi-tenancy):** All computation, storage, and reads are tenant-scoped via `tenant.MustFromContext(ctx)` / tenant headers. One tenant's recompute failure must not affect other tenants' cycles.

### 4.2 Rankings exposure (atlas-rankings REST)

- **FR-8 (Bulk lookup):** A JSON:API endpoint returns rankings for a set of character ids in one call (login builds a char list of up to ~15 characters and must not make N calls).
- **FR-9 (Missing = zeros):** A character with no ranking entry (new, GM, or not yet computed) is represented with all four values zero. The endpoint returns an explicit zero-valued record (or omits the record and the caller defaults to zeros — design decides; either way login renders zeros).

### 4.3 Login integration (atlas-login)

- **FR-10 (Wiring):** When building the character list for world selection (and view-all), atlas-login fetches rankings for the account's characters from atlas-rankings and populates `Rank()`, `RankMove()`, `JobRank()`, `JobRankMove()` on `character.Model`, replacing the hardcoded zeros. Existing GM handling in the packet writer (`rankEnabled` byte = `!gm`) is unchanged.
- **FR-11 (Fail-open):** If the rankings request fails or times out, login proceeds with zeros for all characters and logs a warning. Login flow latency and availability must never depend on atlas-rankings being healthy.

## 5. API Surface

New service atlas-rankings, JSON:API via `api2go/jsonapi`, mounted under the standard REST server base path (`/api/`), tenant headers required.

### GET `/api/rankings/characters?ids={id},{id},...`

Bulk fetch. Returns one resource per requested character id that has a ranking entry (missing ids are simply absent; callers default to zeros).

```json
{
  "data": [
    {
      "type": "rankings",
      "id": "12345",
      "attributes": {
        "worldId": 0,
        "rank": 17,
        "rankMove": 2,
        "jobRank": 4,
        "jobRankMove": -1,
        "computedAt": "2026-07-09T12:00:00Z"
      }
    }
  ]
}
```

- `id` is the character id (string per JSON:API).
- `rankMove`/`jobRankMove` are signed integers server-side; atlas-login converts to the packet lib's `uint32` fields (two's-complement pass-through, matching the client's signed interpretation — the client calls `abs()` and branches on sign).
- Empty/invalid `ids` → 400. Unknown ids → omitted, not an error.

### GET `/api/rankings/characters/{characterId}`

Single fetch, same resource shape. 404 when no entry exists.

Error cases: missing tenant headers → 400 (standard middleware); DB failure → 500. No write endpoints — rankings are computed, never client-mutated.

## 6. Data Model

New Postgres database/schema owned by atlas-rankings (standard per-service DB pattern), GORM entity:

`character_rankings`
- `id` — uuid, surrogate PK (never a natural/business PK; see tenant-PK-collision bug pattern)
- `tenant_id` — uuid, indexed, part of unique key
- `character_id` — uint32
- `world_id` — byte (`world.Id`)
- `job_category` — uint16 (jobId / 100 at compute time)
- `overall_rank` — uint32 (1-based; 0 never stored — unranked characters have no row)
- `overall_rank_move` — int32
- `job_rank` — uint32
- `job_rank_move` — int32
- `computed_at` — timestamp
- Unique index on `(tenant_id, character_id)`
- Index on `(tenant_id, world_id)` for future leaderboard queries

Migration: AutoMigrate on service start (new table, no legacy data). Recompute upserts by `(tenant_id, character_id)` and deletes rows for characters no longer present/eligible.

Snapshot inputs (level/exp/job/world/gm per character) come from authoritative character data; whether atlas-rankings queries atlas-character REST at compute time or maintains its own projection from Kafka character events is a design-phase decision (see §9).

## 7. Service Impact

- **atlas-rankings (new):** Full new Go service at `services/atlas-rankings/atlas.com/rankings/`. Standard patterns: immutable models + builders, Processor interface + Impl, JSON:API REST, tenant middleware, task ticker for the recompute loop. New-service checklist: add to `.github/config/services.json` AND the hand-synced `go_services` list in `docker-bake.hcl`; k8s base manifest + overlays; readiness probe path must be `/api/readyz` (known bug pattern). No new shared libs anticipated (audit `libs/atlas-*` in design before concluding otherwise).
- **atlas-login:** `character/model.go` rank getters backed by real fields + builder setters; processor/requests addition to bulk-fetch rankings during char-list assembly; fail-open on error. No packet writer changes.
- **atlas-tenants:** Ranking recompute interval added to tenant configuration (design decides whether this is a new configuration resource or a field on an existing one, following the generic JSONB configurations pattern).
- **atlas-character:** No changes expected. If design chooses REST-scan acquisition, atlas-rankings uses existing character endpoints (a paged/bulk listing per world may need to be added — design to verify what exists; list-endpoint pagination work is in flight as task-117).
- **libs/atlas-packet:** No changes (encoding verified present).

## 8. Non-Functional Requirements

- **Performance:** A recompute cycle for a tenant must handle tens of thousands of characters per world without materially impacting other request handling; sorting is in-DB or in-memory per world. Login char-list assembly adds at most one bulk rankings call (≤ ~15 ids) with a tight client-side timeout.
- **Availability / fail-open:** atlas-login must not fail or block login flow on atlas-rankings outages (FR-11). atlas-rankings restart mid-cycle must not corrupt rankings (cycle is idempotent; partial writes acceptable only if a re-run converges).
- **Multi-tenancy:** All rows tenant-scoped; recompute iterates tenants independently; no cross-tenant leakage in REST reads (tenant middleware enforced).
- **Observability:** Structured logs per recompute cycle (tenant, world, characters ranked, duration, errors). Failures of a single tenant/world logged and skipped, not fatal (no `log.Fatalf` on per-tenant data issues — known crash-loop pattern).
- **Client compatibility:** v83 rendering of zeros verified (shows "Ranking not available"); rank values render as "Ranked at %d" with sign-driven arrows. v95 rendering of zeros is unverified but the same value-gated family behavior is expected; verify during implementation if a v95 tenant test is planned.

## 9. Open Questions

1. **Data acquisition (design phase):** Kafka projection of character events (level/exp/job changes) vs. REST scan of atlas-character at compute time. Projection avoids load spikes and coupling to atlas-character list endpoints but duplicates state; REST scan is simpler but needs an efficient per-world listing. Decide in `/design-task`, including whether task-117 (list-endpoint pagination) is a prerequisite.
2. **Config shape (design phase):** Exact tenant-configuration resource/field for the interval, and how atlas-rankings observes config changes (config-status projection adoption vs. re-read per cycle).
3. **v95 zero-rank rendering:** Unverified (v83 verified at `0x60292F`). Expected same behavior; check the v95 IDB if v95 tenants are in the test matrix for this task.
4. **GM flag semantics:** Confirm during design how the character `gm` field is populated in Atlas (Cosmic excludes `gm >= 2`; Atlas login treats `gm == 1` as GM). Exclusion rule should match whatever Atlas actually stores — verify against atlas-character source, do not assume.

## 10. Acceptance Criteria

- [ ] atlas-rankings service exists, builds (`go build`, `go vet`, `go test -race` clean), and `docker buildx bake atlas-rankings` succeeds from the repo root.
- [ ] Service is registered in `.github/config/services.json` and `docker-bake.hcl`'s `go_services`, with k8s base manifest (readiness probe on `/api/readyz`) and overlays.
- [ ] Recompute produces correct dense rankings for a seeded multi-world, multi-job, multi-tenant dataset: ordering by level DESC / exp DESC / characterId ASC; job categories per jobId/100; GM characters excluded (unit-tested).
- [ ] rankMove/jobRankMove equal previousRank − newRank across two consecutive recomputes (unit-tested), 0 for first-seen characters.
- [ ] Recompute interval is read from tenant configuration and honored; default 60 minutes when unconfigured.
- [ ] Bulk REST endpoint returns rankings for multiple character ids in one call, tenant-scoped; unknown ids omitted; single-id endpoint 404s when absent.
- [ ] atlas-login populates the four rank fields in the character list from atlas-rankings, and falls back to zeros (logged warning, no login failure) when the rankings call fails.
- [ ] With zeros (new character or pre-first-cycle), the v83 client shows the short info board with "Ranking not available"; with computed ranks it shows "Ranked at N" and correct arrow directions for positive/negative/zero movement (manually verified on a v83 tenant).
- [ ] Two tenants' rankings are fully isolated (integration-style test or manual verification on two tenants).
- [ ] Code review (`superpowers:requesting-code-review`) run before PR.
