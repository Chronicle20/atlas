# Monster Book Client Display Wiring — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-06-03
---

## 1. Overview

The Monster Book feature (task-056, PR #402) introduced the `atlas-monster-book` service that owns each character's card collection, cover, and derived stats. The server-side write path is complete and verified in production: picking up a monster-card item registers the card (`CARD_PICKED_UP` → `CARD_ADDED`), atlas-channel broadcasts the live `MONSTER_BOOK_SET_CARD` packet plus the card-get effect, and the book-level EXP bonus flows through `EXPERIENCE_CHANGED`.

However, the **display path is incomplete**: when a player opens the in-game Monster Book window, it shows **no collected cards and no cover**, even for cards they already own. The root cause is that the login `CharacterData` packet — the packet that seeds the client's Monster Book window — hardcodes an empty book:

```go
// libs/atlas-packet/character/data.go:676
func (m *CharacterData) encodeMonsterBook(w *response.Writer) {
	w.WriteInt(0)   // cover id  → always 0
	w.WriteByte(0)
	w.WriteShort(0) // card count → always 0
}
```

The owned-card list is never fetched or encoded, the `CharacterData` struct has no field to hold it, and the `MonsterBookCoverDecorator` that *would* supply the cover (already written in `atlas-channel`) is never applied in the login decorator chain. This task wires the display end-to-end so the window reflects the player's real collection on login and the cover can be set and viewed.

This work completes a requirement task-056's PRD already mandated but left unwired — login cover decoration (task-056 PRD §4.6, §7: login "MUST query atlas-monster-book for the character's cover … decorator pattern") — and adds the full-collection client-window display that was never specced for the in-game window (task-056 scoped full-collection viewing only to the atlas-ui admin widget, §4.7).

## 2. Goals

Primary goals:
- On login/warp, populate the `CharacterData` Monster Book section with the character's real **cover** and **full owned-card list** (each `cardId` + card level), read from `atlas-monster-book`.
- Apply the cover decorator (or a unified monster-book decorator) in the login decorator chain so the data is fetched.
- Make the Monster Book window show the correct collection immediately on opening, and continue to reflect live `SetCard` updates during the session.
- Get the **version-gating correct** for the login packet: present for **GMS ≤ 87 and JMS**, **absent for GMS v95** (monster book was removed in v95). Match the IDA-verified gate, not the current `GMS && MajorVersion > 28`.
- Verify the full **set-cover loop** in-game: open window → choose an owned card as cover → cover persists across relog.

Non-goals:
- `STATS_CHANGED` → client. There is no Monster Book stats packet in the v83/v87 protocol; book level and counts are computed client-side from the card list, and the EXP bonus already reaches the client via `EXPERIENCE_CHANGED` → `atlas-character`. `STATS_CHANGED` remains a server-internal/atlas-ui concern and is out of scope for atlas-channel.
- The atlas-ui admin Monster Book widget (already shipped in task-056 §4.7).
- "Fill the Book" / card-exchange NPC scripts, card-set bonuses, card scrolls (task-056 non-goals, still out of scope).
- Issue #655 (non-card `consumeOnPickup` data-loss for Monster Carnival / class-202 items) — tracked separately.
- Any change to `atlas-monster-book` persistence or its derived-stat formulas.

## 3. User Stories

- As a player, when I log in I want the Monster Book window to show every card I have collected, at the correct card level, so the window reflects my real progress.
- As a player, I want my chosen cover card to display in the Monster Book window and persist across logins.
- As a player, I want to open the Monster Book and select any card I already own as my cover, and have it take effect immediately and survive relog.
- As a player on a v95 client (where Monster Book was removed), I want my login/character-info packets to remain correctly aligned (no Monster Book block sent), so nothing desyncs.
- As another player viewing my character info, I want to see my cover card (where the client version supports it).

## 4. Functional Requirements

### 4.1 Login CharacterData — Monster Book section
- During `CharacterData` assembly, atlas-channel MUST fetch the character's Monster Book collection (cover) and full card list (`cardId`, `level`) from `atlas-monster-book` via a single login-time fetch (one decorator; see §4.4).
- `encodeMonsterBook` MUST write the real cover id and the real card list in the exact wire format the target client expects for each supported version (see §4.5). The current hardcoded `int(0)/byte(0)/short(0)` stub is replaced.
- The `CharacterData` struct MUST carry the monster-book data (cover id + ordered card list with levels) so the encoder has a source. `BuildCharacterData` (`services/atlas-channel/.../socket/writer/character_data.go`) MUST populate it.
- If the atlas-monster-book fetch fails (REST 404/network), encoding MUST degrade gracefully to the empty-book form (cover 0, 0 cards) so login still succeeds — matching the existing decorator's fail-open behavior.

### 4.2 Live updates during session (already partially working)
- `CARD_ADDED` → `MONSTER_BOOK_SET_CARD` and the card-get effect: already implemented; keep working. No regression.
- `COVER_CHANGED` → `MONSTER_BOOK_SET_COVER`: already implemented; keep working.
- With §4.1 in place, a card picked up mid-session continues to appear via the live `SetCard` packet, and on next login it appears via the seeded collection.

### 4.3 Set-cover loop
- The serverbound set-cover handler already exists (`socket/handler/monster_book_cover.go` → emits `MONSTER_BOOK.SET_COVER`). With the window now populated, selecting an owned card as cover MUST: emit `SET_COVER` → atlas-monster-book validates (owned, level ≥ 1, or 0 to clear) → `COVER_CHANGED` → `MONSTER_BOOK_SET_COVER` to the owner → cover persists and is reflected on next login (§4.1).

### 4.4 Login decorator
- Apply a monster-book decorator in the login decorator chain (`kafka/consumer/session/consumer.go`, the `cp.GetById(...)` decorator list).
- Per the user's decision, use **one unified fetch**: a single decorator that retrieves both the cover and the full card list from atlas-monster-book and attaches them to the character model (replacing/extending the currently-unused `MonsterBookCoverDecorator`). Avoid two separate round trips.
- Fetch cost (one REST call, up to ~340 cards for a maxed book) is acceptable; no special guard required.

### 4.5 Version correctness (critical)
- The login Monster Book block MUST be gated to match the IDA-verified behavior:
  - **GMS ≤ 87** (v83, v87): include the block.
  - **GMS v95**: **omit** the block entirely (Monster Book was removed; the client does not read it). The current call-site gate `(GMS && MajorVersion > 28) || JMS` (`data.go:131`) wrongly includes v95 and MUST be corrected to the equivalent of `(GMS && MajorVersion <= 87) || JMS`.
  - **JMS (v185)**: include the block; note the JMS decode order differs from GMS (see audit references) — encode to match.
- The exact per-card wire format (field widths, ordering, the trailing/leading flag byte, and how card levels are encoded) MUST be confirmed against the client during design using the existing IDA exports and audits — do not guess:
  - `docs/packets/ida-exports/_pending.md` (monster book gate notes: "GMS < 87 only; absent in v95"; gate corrected `< 87` → `<= 87` for v87).
  - `docs/packets/ida-exports/gms_v87.json` (CMonsterBook data 1–5, `currentMobTemplate`).
  - `docs/packets/ida-exports/gms_jms_185.json` (`SomethingMonsterBook`, JMS decode order).
  - `docs/packets/audits/gms_v87/CharacterInfo.{json,md}`.
- **Distinguish the two surfaces** (design must not conflate them):
  - `CharacterData` (login, `data.go`): seeds the player's **own** window — cover + full owned-card list. This is the primary fix.
  - `CharacterInfo` (info popup, `info.go`): the cover/summary shown when **another** player inspects you (audited as a version-gated block). Verify the cover displays here for supported versions; this is a secondary in-scope check, not a new feature.

### 4.6 atlas-monster-book read surface
- Consume `atlas-monster-book`'s existing endpoints:
  - `GET /characters/{characterId}/monster-book` — collection summary incl. `coverCardId` (already modeled in atlas-channel).
  - `GET /characters/{characterId}/monster-book/cards` — the per-card list (`cardId`, `level`, `isSpecial`). atlas-channel currently has **no** model for this; add a REST model + provider.
- No new endpoints are required in atlas-monster-book; confirm the `/cards` response includes `cardId` and `level`. If pagination applies, fetch the full set.

## 5. API Surface

No new public/JSON:API endpoints. Changes are internal:

- **atlas-channel (new REST consumption):** add a request + RestModel + `Extract` for `GET /characters/{characterId}/monster-book/cards` in `services/atlas-channel/.../monsterbook/` (alongside the existing collection model). Shape (to confirm against atlas-monster-book):
  ```
  GET {DATA-style monster-book root}/characters/{characterId}/monster-book/cards
  → { data: [ { id, attributes: { cardId, level, isSpecial } }, ... ] }
  ```
- **atlas-monster-book:** no changes expected. Verify `/cards` returns `cardId` + `level`.

## 6. Data Model

- No database or migration changes. atlas-channel is stateless with respect to Monster Book; all data is owned by atlas-monster-book and fetched per login.
- In-memory only: extend the atlas-channel `character.Model` (or the CharacterData build input) to carry `coverCardId` and an ordered list of `{cardId, level}` for encoding. Multi-tenancy is preserved via the existing tenant-scoped REST calls (tenant header) and `tenant.MustFromContext`.

## 7. Service Impact

- **libs/atlas-packet** (`character/data.go`): add a Monster Book field to `CharacterData`; rewrite `encodeMonsterBook` to encode real cover + card list per version; correct the call-site version gate (§4.5). Possibly touch `character/info.go` for the CharacterInfo cover verification (§4.5).
- **atlas-channel**:
  - `socket/writer/character_data.go` (`BuildCharacterData`): fetch/populate Monster Book data.
  - `kafka/consumer/session/consumer.go`: add the monster-book decorator to the login chain.
  - `character/processor.go`: replace/extend `MonsterBookCoverDecorator` with a unified decorator that also fetches the card list.
  - `monsterbook/{requests.go,rest.go,processor.go}`: add the `/cards` request + model + provider.
  - Existing `kafka/consumer/monsterbook/consumer.go` (CARD_ADDED/COVER_CHANGED) and `socket/handler/monster_book_cover.go` (SET_COVER): unchanged, verify no regression.
- **atlas-monster-book**: none expected (read-only consumer of existing endpoints).

## 8. Non-Functional Requirements

- **Performance:** one additional REST round trip on login (collection + cards in a unified fetch). Acceptable; no caching mandated for v1.
- **Resilience:** fail-open — a monster-book fetch failure degrades to an empty book and never blocks login.
- **Correctness/compat:** version gating must be byte-exact per client; verify with byte-level packet tests (the packet-audit tooling and `libs/atlas-packet` test patterns). A wrong gate or format desyncs the entire login packet — this is the highest risk.
- **Multi-tenancy:** all atlas-monster-book reads are tenant-scoped via the standard tenant header.
- **Observability:** log fetch failures at the decorator (consistent with existing decorator logging); no new metrics required.

## 9. Open Questions

1. **Exact per-card wire format** in `CharacterData.encodeMonsterBook` for v83/v87 and JMS — field widths, ordering, and the meaning of the byte between cover and count. Resolve from IDA exports during design (§4.5); do not guess.
2. **JMS decode-order divergence** — confirm the JMS v185 monster book block ordering relative to GMS (per `gms_jms_185.json` notes) and whether the same encoder can serve both with a version branch.
3. **CharacterInfo cover (secondary surface)** — confirm whether the cover already encodes correctly in `info.go` for supported versions, or whether it needs the same decorator data. Scope to verify + minimal fix only.
4. **`/cards` ordering & pagination** — does atlas-monster-book return cards in a stable order, and is the full set returned in one response? Confirm and, if needed, request all pages.
5. **v95 reachability** — confirm whether v95 clients actually exercise this `CharacterData` path in the current deployment; regardless, the gate must be corrected.

## 10. Acceptance Criteria

- [ ] On login (GMS v83), the Monster Book window shows all owned cards at correct card levels, sourced from atlas-monster-book.
- [ ] The configured cover card displays in the window on login and persists across relog.
- [ ] Setting a cover in-game (open window → select an owned card) updates immediately and survives relog (full loop verified live).
- [ ] Picking up a new card mid-session still shows it via the live `SetCard` packet (no regression) and it is present on next login.
- [ ] The login Monster Book block is present for GMS v83/v87 and JMS, and **absent** for GMS v95 (no packet desync on v95); verified by byte-level packet tests for each supported version.
- [ ] `CharacterInfo` cover display verified for supported versions (secondary).
- [ ] Graceful degradation: with atlas-monster-book unavailable, login still succeeds and the window renders empty.
- [ ] `go build ./...`, `go vet ./...`, `go test -race ./...` clean in every changed module; `docker buildx bake` for any service whose `go.mod` changed; `tools/redis-key-guard.sh` clean.
- [ ] backend-guidelines-reviewer (DOM-*) pass on the diff.
