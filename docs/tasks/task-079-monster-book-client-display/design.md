# Monster Book Client Display Wiring — Design

Version: v1
Status: Draft
Created: 2026-06-03
PRD: `docs/tasks/task-079-monster-book-client-display/prd.md`

---

## 1. Problem & Approach

The server-side Monster Book write path (task-056) is complete: cards register, live
`MONSTER_BOOK_SET_CARD` / `MONSTER_BOOK_SET_COVER` packets broadcast, and the EXP bonus
flows through `EXPERIENCE_CHANGED`. The **display path is unwired**: the login
`CharacterData` packet hardcodes an empty book, so opening the in-game Monster Book window
shows no cards and no cover even for cards the player owns.

This design wires the display end-to-end:

1. Encode the player's real cover + owned-card list in `CharacterData.encodeMonsterBook`
   (`libs/atlas-packet`), per a **client-verified** wire format (§2).
2. Fetch that data at login via a single unified monster-book decorator in atlas-channel.
3. Correct the version gate so the block is present for GMS ≤ 87 and JMS, absent for GMS v95.
4. Verify the set-cover loop and the `CharacterInfo` cover surface (secondary).

The headline finding from reverse engineering (§2) is that **the current atlas encode
structure is already correct for an empty book** — the fix is purely *additive* (populate
the cover, append the card loop). No structural realignment of `CharacterData` is required.
This substantially de-risks the task.

---

## 2. Wire format — verified against the v83 client (NOT memory)

Per CLAUDE.md ("Verification Over Memory"), the format below was extracted by decompiling
the GMS v83 client (`MapleStory_dump.exe`, md5 `80ff438c…`) in IDA, not cited from
MapleStory server lore. The repo IDA exports treat `CharacterData::Decode` as an *envelope
boundary* (inner shape "audited under character domain"), so they do **not** contain this
format — live decompilation was the only ground truth.

### 2.1 What the client actually decodes

`CharacterData::Decode` (`0x4e592d`) is **flag-driven**: it first reads a 64-bit
`dbcharFlag` (atlas writes `WriteInt64(-1)` = all bits set, `data.go:99`), then conditionally
decodes each section based on a flag bit. Two **separate, adjacent** flag-gated blocks carry
the monster book, near the end of the packet:

| Flag bit | Field | Decode | Stored at object offset |
|---|---|---|---|
| `0x20000` | **Cover** | one `Decode4` (int32) | `+1523` (= `SetMonsterBookCover` target, `0x95fb3e`) |
| `0x10000` | **Card list** | `sub_4E4DEA` (see §2.2) | `+1499` ZMap (= the map `CMonsterBookAccessor::UpdateInfo` walks, `0x685c56`) |

The card-list decoder `sub_4E4DEA` (`0x4e4dea`) supports **two wire encodings**, selected by
a leading byte:

- **Mode 0** (leading byte `== 0`) — simple list:
  ```
  byte   mode (= 0)
  short  cardCount
  repeat cardCount:
      short  cardId - 2380000     // card index off the 238xxxx monster-card base
      byte   level
  ```
- **Mode 1** (leading byte `!= 0`) — bitmap + packed nibbles:
  `short ownedCount + byte bitmapLen + bitmap[bitmapLen] + byte nibbleLen + nibbles[nibbleLen]`,
  where the bitmap marks owned card indices over the full card table and levels are packed
  as alternating 4-bit nibbles. The client supports it; **the server need not emit it.**

The client adds `2380000` back to the wire short (`v166 = *m + 2380000`, line 998) to form the
full item id. For the valid card range `[2380000, 2390000)` this equals the familiar
`cardId % 10000` (2380000 is a multiple of 10000) — so the lore value coincides, but we now
*know* the field is `cardId - MonsterBookCardBase`, not an accident.

### 2.2 Reconciliation with the current atlas stub

Current `encodeMonsterBook` (`data.go:676`):
```go
w.WriteInt(0)    // cover  → flag 0x20000 block
w.WriteByte(0)   // mode   → flag 0x10000 block, mode-0 selector
w.WriteShort(0)  // count  → mode-0 card count
```
This is **exactly mode-0 with an empty book**: cover `0`, mode `0`, count `0`. The byte the
PRD flagged as "unknown" (§9 Q1) is the **mode selector**. The structure is correct; only the
data and the card loop are missing.

### 2.3 Target encode (GMS v83/v87)

```go
func (m *CharacterData) encodeMonsterBook(w *response.Writer) {
    w.WriteInt(uint32(m.MonsterBook.CoverCardId)) // 0x20000: full cover item id
    w.WriteByte(0)                                // 0x10000: mode-0 (simple list)
    w.WriteShort(uint16(len(m.MonsterBook.Cards)))
    for _, c := range m.MonsterBook.Cards {
        w.WriteShort(uint16(c.CardId - MonsterBookCardBase)) // 2380000
        w.WriteByte(c.Level)
    }
}
```
For an empty book this is **byte-identical to today**, so the change is strictly additive and
cannot desync existing logins. `decodeMonsterBook` (atlas's own round-trip decoder) is updated
to the symmetric shape (cover int, mode byte, count short, then `count × {short, byte}`).

### 2.3.1 Cover representation — execution verify point
The decode stores the cover `Decode4` raw at `+1523` (`SetMonsterBookCover` writes the same
offset). We encode the **full** item id (atlas-channel already models `CoverCardId` as
`item.Id`). One execution check: confirm in-game that the cover renders correctly as a full id
vs. a `-2380000` offset (the card *list* uses the offset; the cover field's raw store suggests
full id, but this is the one cover detail to confirm live + by the CharacterInfo audit).

### 2.4 `MonsterBookCardBase` constant
`2380000` MUST be a named constant, not a magic number (DOM-21). During execution, check
`libs/atlas-constants/item` for an existing monster-card base/classification; if none exists,
add `MonsterBookCardBase item.Id = 2380000` there (preferred) or in `libs/atlas-packet/character`.

### 2.5 JMS (v185) — deferred to execution, method proven
The same approach applies, but JMS decode **order/gating differs** (PRD §9 Q2; the repo notes
flag JMS `SomethingMonsterBook` / a divergent decode sequence). The v83 reverse-engineering
method (load IDB → decompile `CharacterData::Decode` → locate the flag-gated card block →
read `sub_*` card decoder) is proven and reusable. **Execution task:** reload the JMS v185 IDB
(`MapleStory_dump_SCY.exe`) and confirm the JMS card-block format + position before encoding a
JMS branch; gate any JMS-specific divergence behind `t.Region() == "JMS"`. Do not assume JMS ==
v83 on the wire.

### 2.6 v87 — confirm, expect identical
v87 shares the v83 client lineage; the repo already corrected the v87 *CharacterInfo* gate to
`<= 87`. **Execution task:** load the v87 IDB and confirm `CharacterData::Decode` monster-book
blocks match v83 (expected identical); add a v87 byte-level test regardless.

---

## 3. Version gating (critical)

The login block is currently gated at the call site by `(GMS && MajorVersion > 28) || JMS`
(`data.go:131`), which **wrongly includes GMS v95** (Monster Book was removed in v95). Correct to:

```go
if (t.Region() == "GMS" && t.MajorVersion() <= 87) || t.Region() == "JMS" {
    m.encodeMonsterBook(w)
    // newYear / area / trailing shorts unchanged
}
```

This matches the IDA-verified gate (repo `_pending.md`: "GMS < 87 only; absent in v95", gate
corrected to `<= 87` for v87 in `info.go`). The symmetric `decodeMonsterBook` call site
(`data.go:150`-region) is updated identically so round-trip tests hold.

**Scope caution:** the `(GMS && >28) || JMS` predicate appears multiple times in `data.go`
(lines 98, 109, 131, 658, 669, …) guarding *unrelated* blocks (SN list, linked name, teleports).
Only the **monster-book** call sites (encode + decode) change to `<= 87`. The others are out of
scope — do not blanket-replace the predicate.

---

## 4. Architecture & data flow

```
login (session consumer, consumer.go:166)
  └─ cp.GetById(Inventory, PetAsset, Skill, Quest, MonsterBook)(characterId)
        └─ MonsterBookDecorator(m)                         [character/processor.go]
              ├─ monsterbook.GetByCharacterId  → cover     [GET /characters/{id}/monster-book]
              └─ monsterbook.GetCardsByCharacterId → cards  [GET /characters/{id}/monster-book/cards]
              └─ m.SetCoverCardId(cover).SetMonsterBookCards(cards)   (fail-open: returns m on error)
  └─ BuildCharacterData(m, …)                              [socket/writer/character_data.go]
        └─ data.MonsterBook = { CoverCardId, Cards[] }
  └─ CharacterData.Encode → encodeMonsterBook (§2.3)       [libs/atlas-packet]
```

### 4.1 Components

1. **`libs/atlas-packet/character/data.go`**
   - Add a `MonsterBook` field to the `CharacterData` struct: `{ CoverCardId item.Id; Cards []MonsterBookCard }`,
     where `MonsterBookCard = { CardId item.Id; Level byte }`.
   - Rewrite `encodeMonsterBook` / `decodeMonsterBook` (§2.3).
   - Correct the monster-book gate to `<= 87` (§3).
   - `MonsterBookCardBase` constant (§2.4).

2. **atlas-channel `monsterbook/` package** (REST consumption — new `/cards` surface)
   - `requests.go`: add `CardsResource = "characters/%d/monster-book/cards"` + `requestCardsByCharacterId`.
   - `rest.go`: add `CardRestModel { CardId item.Id (json:"-" / GetID), Level uint8, IsSpecial bool }`
     mirroring atlas-monster-book `card/rest.go` (id = cardId, attributes = level/isSpecial) + `ExtractCard`.
   - `processor.go`: add domain `Card { cardId item.Id; level uint8; isSpecial bool }` + getters,
     `CardsByCharacterIdProvider`, and `GetCardsByCharacterId(characterId) ([]Card, error)`.

3. **atlas-channel `character/`**
   - `model.go`: add `monsterBookCards []MonsterBookCard` field (cardId + level) + getter
     `MonsterBookCards()` + `SetMonsterBookCards`, threaded through `CloneModel`/builder
     (mirrors `coverCardId`).
   - `processor.go`: **replace** `MonsterBookCoverDecorator` with `MonsterBookDecorator(m)` —
     one decorator that fetches the collection (cover) **and** the card list, attaches both,
     and fails open (returns `m` on any REST error). Remove the now-unused cover-only decorator.

4. **atlas-channel `socket/writer/character_data.go`** (`BuildCharacterData`)
   - Populate `data.MonsterBook.CoverCardId` from `m.CoverCardId()` and `data.MonsterBook.Cards`
     from `m.MonsterBookCards()`.

5. **atlas-channel `kafka/consumer/session/consumer.go`**
   - Add `cp.MonsterBookDecorator` to the `cp.GetById(...)` decorator list at line ~166.

6. **`info.go` (CharacterInfo, secondary)** — verify the cover encodes for supported versions;
   minimal fix only if the audit/byte-test shows a gap. No new feature.

### 4.2 Unchanged (verify no regression)
- `kafka/consumer/monsterbook/consumer.go` (CARD_ADDED → SetCard, COVER_CHANGED → SetCover).
- `socket/handler/monster_book_cover.go` (serverbound SET_COVER).
- atlas-monster-book: read-only; no changes (endpoints already exist).

---

## 5. Data model & multi-tenancy

- **No DB / migration changes.** atlas-channel is stateless w.r.t. Monster Book; all data is
  owned by atlas-monster-book and fetched per login.
- In-memory only: `character.Model` gains `monsterBookCards`; `CharacterData` gains `MonsterBook`.
- Multi-tenancy preserved: both REST reads go through the existing tenant-scoped
  `monsterbook.NewProcessor(l, ctx)` (tenant header via `tenant.MustFromContext`).

### 5.1 `/cards` ordering & pagination (PRD §9 Q4)
The atlas-monster-book `card` handler returns the full JSON:API collection (no pagination in
the current handler — confirm during execution). Card **order on the wire is not
client-significant** (the client builds a keyed map), so a stable order is nice-to-have, not
required. If atlas-monster-book paginates, fetch all pages; otherwise consume the single
response. Execution must confirm the endpoint returns `cardId` + `level` (it does: `card/rest.go`).

---

## 6. Alternatives considered

**A. Two decorators (cover + cards) vs. one unified decorator.**
Chosen: **one unified `MonsterBookDecorator`** (PRD §4.4 decision). It makes two REST calls but
is one decorator in the chain, one fail-open boundary, and one place to reason about. Two
separate decorators would double the chain entries and split the failure handling for no
benefit. (Cost: ~2 REST calls/login, up to ~340 cards — acceptable per PRD §8.)

**B. Wire encoding: mode-0 simple list vs. mode-1 bitmap.**
Chosen: **mode 0.** The client decodes both; mode-0 is trivial to emit and matches the existing
stub's structure (additive change). Mode-1 (bitmap+nibbles) saves bytes for large books but
adds real encoding complexity (full card-table enumeration, nibble packing) for a payload that
is already acceptable. YAGNI.

**C. Send cover as full id vs. `-2380000` offset.**
Chosen: **full id** (the cover field is stored raw by the client; atlas already models it as a
full `item.Id`). Flagged as the single cover detail to confirm live (§2.3.1).

**D. Trust the well-known (HeavenMS) format vs. verify in IDA.**
Chosen: **verify** — and it paid off: we confirmed the leading byte is a *mode selector* (not a
constant), discovered the alternate bitmap encoding, and the two-block flag gating. The lore
value for the card short happened to coincide, but the structure understanding (and the cover
gating) would have been guesswork otherwise. Per CLAUDE.md, memory is not evidence.

---

## 7. Testing strategy

Correctness here is byte-exact: a wrong gate or width desyncs the *entire* login packet, so
tests are the primary safety net.

1. **`libs/atlas-packet` byte-level tests** (per existing `data.go` test patterns):
   - GMS v83: empty book → byte-identical to the pre-change empty stub (regression guard).
   - GMS v83: N cards + non-zero cover → exact bytes
     `int(cover) | byte(0) | short(N) | N×{short(cardId-2380000), byte(level)}`.
   - GMS v87: same as v83 (after IDB confirmation, §2.6).
   - GMS v95: monster-book block **absent** — assert the encoded packet has no monster-book
     bytes and the surrounding bytes (newYear/area/trailing) align (gate regression guard).
   - JMS v185: per the execution IDB confirmation (§2.5); byte test for the JMS shape.
   - Round-trip: `encodeMonsterBook` → `decodeMonsterBook` for empty + populated.
2. **atlas-channel `monsterbook` tests**: `CardRestModel` extract from a JSON:API `/cards`
   fixture (reuse `rest_test.go` httptest pattern); `MonsterBookDecorator` fail-open
   (REST 404/network → undecorated model, cover 0 / no cards).
3. **`BuildCharacterData`**: model with cover+cards → `CharacterData.MonsterBook` populated.
4. **In-game verification (acceptance):** v83 login shows owned cards at correct levels + cover;
   set-cover loop (open window → pick owned card → persists across relog); mid-session pickup
   still appears via live `SetCard` and on next login; atlas-monster-book down → empty book,
   login succeeds.

---

## 8. Risks

| Risk | Severity | Mitigation |
|---|---|---|
| Wrong card/level width or base desyncs login | High | v83 format IDA-verified (§2); byte-level tests per version; empty-book test proves additive-only change |
| JMS decode order differs, breaks JMS login | Med | Deferred, gated; reload JMS IDB + confirm before encoding JMS branch (§2.5); JMS byte test |
| v95 gate regression | High | Correct only the monster-book call sites to `<= 87` (§3); v95 absent-block byte test |
| Over-broad predicate replacement in `data.go` | Med | Only monster-book encode/decode call sites change; other `>28` guards untouched (§3) |
| Cover full-id vs offset | Low | Confirm live + CharacterInfo audit (§2.3.1); easy one-line flip |
| `/cards` pagination | Low | Confirm endpoint returns full set; fetch all pages if paginated (§5.1) |

---

## 9. Out of scope (per PRD §2)

`STATS_CHANGED` → client (no v83/v87 stats packet; counts are client-derived); atlas-ui admin
widget; Fill-the-Book / card-exchange / card scrolls; issue #655 (non-card `consumeOnPickup`);
any atlas-monster-book persistence or derived-stat changes.

---

## 10. Execution checklist (carried into plan)

- [ ] `MonsterBookCardBase` constant (check `libs/atlas-constants/item` first; §2.4).
- [ ] `CharacterData.MonsterBook` field + `MonsterBookCard` type; rewrite encode/decode (§2.3).
- [ ] Correct monster-book gate to `<= 87` (encode + decode call sites only; §3).
- [ ] atlas-channel `monsterbook`: `/cards` request + `CardRestModel` + `Card` + provider.
- [ ] `character.Model`: `monsterBookCards` field + getter/setter + builder threading.
- [ ] Replace `MonsterBookCoverDecorator` with unified `MonsterBookDecorator` (fail-open).
- [ ] `BuildCharacterData`: populate `MonsterBook` from the model.
- [ ] Add `MonsterBookDecorator` to the login `GetById` chain (`consumer.go:166`).
- [ ] **Reload JMS v185 IDB; confirm + encode JMS branch** (§2.5).
- [ ] **Confirm v87 IDB; byte test** (§2.6).
- [ ] Verify `CharacterInfo` cover (secondary; §4.1 item 6).
- [ ] Byte-level tests for v83/v87/v95/JMS (§7); fail-open + build tests.
- [ ] `go build/vet/test -race` clean; `docker buildx bake` for atlas-channel + atlas-monster-book
      (if `go.mod` touched — likely only atlas-channel + libs); `redis-key-guard.sh` clean.
- [ ] backend-guidelines-reviewer (DOM-*) pass.
```
