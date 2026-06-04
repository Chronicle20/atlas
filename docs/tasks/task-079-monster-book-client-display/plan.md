# Monster Book Client Display Wiring Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Populate the login `CharacterData` Monster Book section with the character's real cover and owned-card list (fetched per-login from `atlas-monster-book`), and correct the version gate so the block is present for GMS ≤ 87 and JMS but absent for GMS v95.

**Architecture:** A single unified `MonsterBookDecorator` in atlas-channel fetches the collection (cover) + card list via REST and attaches them to the immutable `character.Model`. `BuildCharacterData` maps that onto a new `CharacterData.MonsterBook` field. `libs/atlas-packet` encodes it in the IDA-verified mode-0 wire format (cover `int32`, mode `byte=0`, `short` count, then per card `short(cardId−2380000)` + `byte(level)`). The change is byte-identical to today for an empty book, so it is strictly additive for supported versions; the only removal is the monster-book block for GMS v95, achieved by splitting the existing combined gate.

**Tech Stack:** Go 1.x, `libs/atlas-packet` (`response.Writer`/`request.Reader`, little-endian), `libs/atlas-rest` (`requests.SliceProvider`/`Provider`), JSON:API via `api2go/jsonapi`, `libs/atlas-constants/item`. Tests: standard `testing`, `httptest`, the `libs/atlas-packet/test` round-trip harness.

---

## Read first

- `docs/tasks/task-079-monster-book-client-display/design.md` — wire format (§2), gate (§3), data flow (§4).
- `docs/tasks/task-079-monster-book-client-display/context.md` — verified facts, file map, **two corrections to the design** (cover decorator is in use; the gate must be split, not blanket-moved).

Every implementer subagent prompt MUST `cd` into the worktree
`<repo-root>/.worktrees/task-079-monster-book-client-display`
first and run `git branch --show-current` (must be `task-079-monster-book-client-display`) after each commit.

---

## Task 1: `MonsterBookCardBase` constant

**Files:**
- Modify: `libs/atlas-constants/item/constants.go`
- Test: `libs/atlas-constants/item/constants_test.go`

- [ ] **Step 1: Write the failing test**

Append to `libs/atlas-constants/item/constants_test.go`:

```go
func TestMonsterBookCardBase(t *testing.T) {
	if MonsterBookCardBase != Id(2380000) {
		t.Errorf("MonsterBookCardBase = %d, want 2380000", MonsterBookCardBase)
	}
	// Base must be the classification 238 times 10000 so cardId-base yields the
	// familiar card index the client adds back (design §2.1).
	if uint32(MonsterBookCardBase) != uint32(ClassificationConsumableMonsterCard)*10000 {
		t.Errorf("MonsterBookCardBase %d != ClassificationConsumableMonsterCard*10000", MonsterBookCardBase)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd libs/atlas-constants && go test ./item/ -run TestMonsterBookCardBase -v`
Expected: FAIL — `undefined: MonsterBookCardBase`.

- [ ] **Step 3: Add the constant**

In `libs/atlas-constants/item/constants.go`, add an `Id` constant near the monster-card classification (`ClassificationConsumableMonsterCard = Classification(238)`). Add a new `const` block (or extend an existing `Id` block if one exists):

```go
// MonsterBookCardBase is the item-id base for monster-book cards. A card's
// wire index in the CharacterData monster-book block is (cardId - this base);
// the client adds it back to reconstruct the full item id (design §2.1).
const MonsterBookCardBase = Id(2380000)
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd libs/atlas-constants && go test ./item/ -run TestMonsterBookCardBase -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-constants/item/constants.go libs/atlas-constants/item/constants_test.go
git commit -m "feat(constants): add MonsterBookCardBase item id base"
```

---

## Task 2: Packet types + `encodeMonsterBook`/`decodeMonsterBook` rewrite

**Files:**
- Modify: `libs/atlas-packet/character/data.go` (struct lines 82–91; functions 676–686; imports 3–14)
- Test: `libs/atlas-packet/character/data_test.go`

- [ ] **Step 1: Write the failing byte-level tests**

Append to `libs/atlas-packet/character/data_test.go` (add `"bytes"`, `"github.com/Chronicle20/atlas/libs/atlas-socket/response"` and `testlog "github.com/sirupsen/logrus/hooks/test"` to its imports):

```go
func TestEncodeMonsterBook_Empty(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	w := response.NewWriter(l)
	cd := CharacterData{}
	cd.encodeMonsterBook(w)
	got := w.Bytes()
	// cover int(0) | mode byte(0) | count short(0) — byte-identical to the old stub.
	want := []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	if !bytes.Equal(got, want) {
		t.Errorf("empty book bytes = % x, want % x", got, want)
	}
}

func TestEncodeMonsterBook_Populated(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	w := response.NewWriter(l)
	cd := CharacterData{
		MonsterBook: MonsterBookData{
			CoverCardId: 2380001,
			Cards: []MonsterBookCard{
				{CardId: 2380005, Level: 2},
				{CardId: 2382000, Level: 5},
			},
		},
	}
	cd.encodeMonsterBook(w)
	got := w.Bytes()
	// cover 2380001 (LE 61 4C 24 00) | mode 00 | count 2 (02 00)
	// | card 5 (05 00) lvl 2 (02) | card 2000 (D0 07) lvl 5 (05)
	want := []byte{
		0x61, 0x4C, 0x24, 0x00,
		0x00,
		0x02, 0x00,
		0x05, 0x00, 0x02,
		0xD0, 0x07, 0x05,
	}
	if !bytes.Equal(got, want) {
		t.Errorf("populated book bytes = % x, want % x", got, want)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd libs/atlas-packet && go test ./character/ -run TestEncodeMonsterBook -v`
Expected: FAIL — `MonsterBookData`/`MonsterBookCard` undefined and/or `CharacterData has no field MonsterBook`.

- [ ] **Step 3: Add the types and the `MonsterBook` field**

In `libs/atlas-packet/character/data.go`, add the `item` import:

```go
import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/Chronicle20/atlas/libs/atlas-packet/model"
	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/sirupsen/logrus"
)
```

Add the types just above `type CharacterData struct {`:

```go
// MonsterBookCard is a single owned monster-book card and its level, as
// carried in the CharacterData login packet.
type MonsterBookCard struct {
	CardId item.Id
	Level  byte
}

// MonsterBookData is the player's monster-book state for the login window:
// the chosen cover (full item id, 0 if none) and the full owned-card list.
type MonsterBookData struct {
	CoverCardId item.Id
	Cards       []MonsterBookCard
}
```

Add the field to `CharacterData`:

```go
type CharacterData struct {
	Stats           CharacterStats
	BuddyCapacity   byte
	Meso            uint32
	Inventory       InventoryData
	Skills          []SkillEntry
	Cooldowns       []CooldownEntry
	StartedQuests   []QuestProgress
	CompletedQuests []QuestCompleted
	MonsterBook     MonsterBookData
}
```

- [ ] **Step 4: Rewrite `encodeMonsterBook` / `decodeMonsterBook`**

Replace the bodies at `data.go:676-686`:

```go
func (m *CharacterData) encodeMonsterBook(w *response.Writer) {
	w.WriteInt(uint32(m.MonsterBook.CoverCardId)) // cover: full item id (flag 0x20000)
	w.WriteByte(0)                                // mode 0: simple list (flag 0x10000)
	w.WriteShort(uint16(len(m.MonsterBook.Cards)))
	for _, c := range m.MonsterBook.Cards {
		w.WriteShort(uint16(uint32(c.CardId) - uint32(item.MonsterBookCardBase)))
		w.WriteByte(c.Level)
	}
}

// decodeMonsterBook is the symmetric reader for atlas's own mode-0 output. The
// server only ever emits mode 0, so only mode 0 is decoded (the client-side
// mode-1 bitmap form is never produced here).
func (m *CharacterData) decodeMonsterBook(r *request.Reader) {
	m.MonsterBook.CoverCardId = item.Id(r.ReadUint32())
	_ = r.ReadByte() // mode selector (always 0 on the wire we emit)
	count := r.ReadUint16()
	m.MonsterBook.Cards = make([]MonsterBookCard, count)
	for i := uint16(0); i < count; i++ {
		m.MonsterBook.Cards[i].CardId = item.MonsterBookCardBase + item.Id(r.ReadUint16())
		m.MonsterBook.Cards[i].Level = r.ReadByte()
	}
}
```

- [ ] **Step 5: Run the byte-level tests to verify they pass**

Run: `cd libs/atlas-packet && go test ./character/ -run TestEncodeMonsterBook -v`
Expected: PASS (both).

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-packet/character/data.go libs/atlas-packet/character/data_test.go
git commit -m "feat(packet): encode real monster-book cover + card list in CharacterData"
```

---

## Task 3: Split the version gate (monster book ≤ 87; tail unchanged)

**Files:**
- Modify: `libs/atlas-packet/character/data.go` (encode block 131–140; decode block 186–195)
- Test: `libs/atlas-packet/character/data_test.go`

> **Why split, not blanket-move:** the `(GMS && >28) || JMS` gate at line 131 wraps
> the monster book **and** newYear/area/trailing shorts. Moving the whole block to
> `<= 87` would also drop those tails for v95, which the design never verified. The
> split removes only the monster-book bytes for v95 and leaves the tail at its
> current gate. See **Step 0**.

- [ ] **Step 0: Verify the v95 CharacterData tail (IDA / repo exports)**

Confirm whether GMS v95's `CharacterData::Decode` still reads the newYear/area/trailing
shorts after the (absent) monster book. Check `docs/packets/ida-exports/gms_v95.json`
and `docs/packets/ida-exports/_pending.md`; if inconclusive, load the v95 IDB.

- **If the tail is still read in v95 (expected default):** use the split gate in Step 3
  below (monster book `<= 87`, tail `> 28`).
- **If v95 also dropped the tail:** change the *second* `if` predicate (and its decode
  twin) to `(GMS && <= 87) || JMS` as well, and update the v95 expectation in Step 1 to
  expect alignment with no tail bytes. Record the finding in a commit message.

Record the verification result inline in the commit message for this task.

- [ ] **Step 1: Write the failing round-trip / v95-absence tests**

Append to `libs/atlas-packet/character/data_test.go`:

```go
func TestCharacterDataMonsterBookRoundTrip(t *testing.T) {
	for _, v := range pt.Variants {
		t.Run(v.Name, func(t *testing.T) {
			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
			input := CharacterData{
				Stats: CharacterStats{Id: 3000, Name: "Booker", Level: 30, JobId: 100, MapId: 100000000},
				Inventory: InventoryData{
					EquipCapacity: 24, UseCapacity: 24, SetupCapacity: 24,
					EtcCapacity: 24, CashCapacity: 24, Timestamp: 94354848000000000,
				},
				MonsterBook: MonsterBookData{
					CoverCardId: 2380001,
					Cards: []MonsterBookCard{
						{CardId: 2380005, Level: 2},
						{CardId: 2382000, Level: 5},
					},
				},
			}
			output := CharacterData{}
			// RoundTrip fails if any byte is left unconsumed — the gate-alignment guard.
			pt.RoundTrip(t, ctx, input.Encode, output.Decode, nil)

			bookPresent := (v.Region == "GMS" && v.MajorVersion <= 87) || v.Region == "JMS"
			if bookPresent {
				if output.MonsterBook.CoverCardId != input.MonsterBook.CoverCardId {
					t.Errorf("cover: got %d, want %d", output.MonsterBook.CoverCardId, input.MonsterBook.CoverCardId)
				}
				if len(output.MonsterBook.Cards) != len(input.MonsterBook.Cards) {
					t.Fatalf("card count: got %d, want %d", len(output.MonsterBook.Cards), len(input.MonsterBook.Cards))
				}
				for i := range output.MonsterBook.Cards {
					if output.MonsterBook.Cards[i] != input.MonsterBook.Cards[i] {
						t.Errorf("card[%d]: got %+v, want %+v", i, output.MonsterBook.Cards[i], input.MonsterBook.Cards[i])
					}
				}
			} else {
				// v95 (and v28): monster book absent — encoder wrote nothing, decoder read nothing.
				if output.MonsterBook.CoverCardId != 0 || len(output.MonsterBook.Cards) != 0 {
					t.Errorf("expected empty monster book for %s, got cover=%d cards=%d",
						v.Name, output.MonsterBook.CoverCardId, len(output.MonsterBook.Cards))
				}
			}
		})
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `cd libs/atlas-packet && go test ./character/ -run TestCharacterDataMonsterBookRoundTrip -v`
Expected: FAIL for `GMS v95` — with the current single `>28` gate the encoder still
writes the monster book for v95, so `output.MonsterBook` is populated (cover 2380001)
when the test expects it empty.

- [ ] **Step 3: Split the encode gate**

In `data.go`, replace the encode block (currently lines 131–140):

```go
		// Monster book: present for GMS <= 87 and JMS; removed in GMS v95+.
		if (t.Region() == "GMS" && t.MajorVersion() <= 87) || t.Region() == "JMS" {
			m.encodeMonsterBook(w)
		}
		// New-year cards / area popup / trailing short — gate unchanged (GMS > 28 || JMS).
		if (t.Region() == "GMS" && t.MajorVersion() > 28) || t.Region() == "JMS" {
			if t.Region() == "GMS" {
				m.encodeNewYear(w)
				m.encodeArea(w)
			} else if t.Region() == "JMS" {
				w.WriteShort(0)
			}
			w.WriteShort(0)
		}
```

- [ ] **Step 4: Split the decode gate (symmetric)**

In `data.go`, replace the decode block (currently lines 186–195):

```go
		// Monster book: present for GMS <= 87 and JMS; removed in GMS v95+.
		if (t.Region() == "GMS" && t.MajorVersion() <= 87) || t.Region() == "JMS" {
			m.decodeMonsterBook(r)
		}
		// New-year cards / area popup / trailing short — gate unchanged (GMS > 28 || JMS).
		if (t.Region() == "GMS" && t.MajorVersion() > 28) || t.Region() == "JMS" {
			if t.Region() == "GMS" {
				m.decodeNewYear(r)
				m.decodeArea(r)
			} else if t.Region() == "JMS" {
				_ = r.ReadUint16()
			}
			_ = r.ReadUint16()
		}
```

- [ ] **Step 5: Run the full packet test suite to verify it passes**

Run: `cd libs/atlas-packet && go test ./character/ -v`
Expected: PASS for all variants — including the pre-existing `TestCharacterDataMinimalRoundTrip`/`...WithSkills`/`...WithQuests` (which exercise v95 alignment), and the new monster-book round-trip.

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-packet/character/data.go libs/atlas-packet/character/data_test.go
git commit -m "fix(packet): gate CharacterData monster book to GMS<=87+JMS (absent v95)"
```

---

## Task 4: atlas-channel `/cards` REST consumption

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/monsterbook/requests.go`
- Modify: `services/atlas-channel/atlas.com/channel/monsterbook/rest.go`
- Modify: `services/atlas-channel/atlas.com/channel/monsterbook/processor.go`
- Test: `services/atlas-channel/atlas.com/channel/monsterbook/rest_test.go`

- [ ] **Step 1: Write the failing tests**

Append to `services/atlas-channel/atlas.com/channel/monsterbook/rest_test.go`:

```go
func TestCardRestModel_Unmarshal(t *testing.T) {
	body := []byte(`{
		"data": [
			{"type":"monster-book-card","id":"2380005","attributes":{"level":2,"isSpecial":false}},
			{"type":"monster-book-card","id":"2382000","attributes":{"level":5,"isSpecial":true}}
		]
	}`)
	var rms []CardRestModel
	if err := jsonapi.Unmarshal(body, &rms); err != nil {
		t.Fatalf("jsonapi.Unmarshal: %v", err)
	}
	if len(rms) != 2 {
		t.Fatalf("len = %d, want 2", len(rms))
	}
	if rms[0].CardId != item.Id(2380005) || rms[0].Level != 2 || rms[0].IsSpecial {
		t.Errorf("card[0] = %+v", rms[0])
	}
	if rms[1].CardId != item.Id(2382000) || rms[1].Level != 5 || !rms[1].IsSpecial {
		t.Errorf("card[1] = %+v", rms[1])
	}
}

func TestGetCardsByCharacterId_RoundTrip(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/characters/42/monster-book/cards") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(`{
			"data": [
				{"type":"monster-book-card","id":"2380005","attributes":{"level":2,"isSpecial":false}},
				{"type":"monster-book-card","id":"2382000","attributes":{"level":5,"isSpecial":true}}
			]
		}`))
	}))
	defer srv.Close()
	defer SetBaseURLForTest(srv.URL)()

	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	cards, err := NewProcessor(logrus.New(), ctx).GetCardsByCharacterId(character.Id(42))
	if err != nil {
		t.Fatalf("GetCardsByCharacterId: %v", err)
	}
	if len(cards) != 2 {
		t.Fatalf("len = %d, want 2", len(cards))
	}
	if cards[0].CardId() != item.Id(2380005) || cards[0].Level() != 2 {
		t.Errorf("card[0] = cardId %d level %d", cards[0].CardId(), cards[0].Level())
	}
}

func TestGetCardsByCharacterId_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	defer SetBaseURLForTest(srv.URL)()

	tm := newTestTenant(t)
	ctx := tenant.WithContext(context.Background(), tm)
	_, err := NewProcessor(logrus.New(), ctx).GetCardsByCharacterId(character.Id(42))
	if err == nil {
		t.Fatal("expected error on 404, got nil")
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./monsterbook/ -run 'Card' -v`
Expected: FAIL — `CardRestModel` and `GetCardsByCharacterId` undefined.

- [ ] **Step 3: Add the `/cards` request builder**

In `monsterbook/requests.go`, fold the resource constants into one block and add the request builder:

```go
const (
	// Resource is the path template for a character's monster book collection.
	Resource = "characters/%d/monster-book"
	// CardsResource is the path template for a character's owned card list.
	CardsResource = "characters/%d/monster-book/cards"
)

func requestCardsByCharacterId(characterId character.Id) requests.Request[[]CardRestModel] {
	return requests.GetRequest[[]CardRestModel](fmt.Sprintf(getBaseRequest()+CardsResource, characterId))
}
```

(Replace the existing single-line `Resource` const declaration with the block above.)

- [ ] **Step 4: Add the `CardRestModel` wire model**

In `monsterbook/rest.go`, add (mirrors atlas-monster-book `card/rest.go`):

```go
// CardRestModel is the JSON:API representation of a single owned monster-book
// card returned by atlas-monster-book's /cards endpoint.
type CardRestModel struct {
	CardId    item.Id `json:"-"`
	Level     uint8   `json:"level"`
	IsSpecial bool    `json:"isSpecial"`
}

func (r CardRestModel) GetName() string { return "monster-book-card" }

func (r CardRestModel) GetID() string { return strconv.FormatUint(uint64(r.CardId), 10) }

func (r *CardRestModel) SetID(strId string) error {
	if strId == "" {
		r.CardId = 0
		return nil
	}
	id, err := strconv.ParseUint(strId, 10, 32)
	if err != nil {
		return err
	}
	r.CardId = item.Id(id)
	return nil
}

func (r CardRestModel) GetReferences() []jsonapi.Reference                { return []jsonapi.Reference{} }
func (r CardRestModel) GetReferencedIDs() []jsonapi.ReferenceID           { return []jsonapi.ReferenceID{} }
func (r *CardRestModel) SetToOneReferenceID(_ string, _ string) error     { return nil }
func (r *CardRestModel) SetToManyReferenceIDs(_ string, _ []string) error { return nil }

// ExtractCard converts the wire model into the immutable domain Card.
func ExtractCard(rm CardRestModel) (Card, error) {
	return Card{cardId: rm.CardId, level: rm.Level, isSpecial: rm.IsSpecial}, nil
}
```

- [ ] **Step 5: Add the `Card` domain type + provider to the processor**

In `monsterbook/processor.go`, add the domain type and getters near `Collection`:

```go
// Card is the immutable domain representation of a single owned monster-book card.
type Card struct {
	cardId    item.Id
	level     uint8
	isSpecial bool
}

func (c Card) CardId() item.Id { return c.cardId }
func (c Card) Level() uint8    { return c.level }
func (c Card) IsSpecial() bool { return c.isSpecial }
```

Add to the `Processor` interface:

```go
	GetCardsByCharacterId(characterId character.Id) ([]Card, error)
	CardsByCharacterIdProvider(characterId character.Id) model.Provider[[]Card]
```

Add the impl methods:

```go
// CardsByCharacterIdProvider returns a provider that fetches the character's
// owned monster-book cards from atlas-monster-book.
func (p *ProcessorImpl) CardsByCharacterIdProvider(characterId character.Id) model.Provider[[]Card] {
	return requests.SliceProvider[CardRestModel, Card](p.l, p.ctx)(requestCardsByCharacterId(characterId), ExtractCard, model.Filters[Card]())
}

// GetCardsByCharacterId fetches and returns the owned card list for the character.
func (p *ProcessorImpl) GetCardsByCharacterId(characterId character.Id) ([]Card, error) {
	return p.CardsByCharacterIdProvider(characterId)()
}
```

- [ ] **Step 6: Run the tests to verify they pass**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./monsterbook/ -v`
Expected: PASS (new card tests + existing collection tests).

- [ ] **Step 7: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/monsterbook/
git commit -m "feat(channel): consume atlas-monster-book /cards endpoint"
```

---

## Task 5: `character.Model` — owned-card field

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/character/model.go` (struct ~62; getters ~352)
- Modify: `services/atlas-channel/atlas.com/channel/character/builder.go` (struct ~60; CloneModel ~108; setters ~149; Build ~194)
- Test: `services/atlas-channel/atlas.com/channel/character/builder_test.go`

> The model stores the domain type `monsterbook.Card` (cardId + level + isSpecial).
> `character/processor.go` already imports `atlas-channel/monsterbook`; `monsterbook`
> does **not** import the channel `character` package (it imports
> `atlas-constants/character`), so there is no import cycle. If a cycle nonetheless
> appears at build time, fall back to a local `type MonsterBookCard struct { CardId
> item.Id; Level uint8 }` in the character package.

- [ ] **Step 1: Write the failing test**

Append to `services/atlas-channel/atlas.com/channel/character/builder_test.go` (add imports `"atlas-channel/monsterbook"` and `"github.com/Chronicle20/atlas/libs/atlas-constants/item"` if not present):

```go
func TestModel_MonsterBookCards(t *testing.T) {
	cards := []monsterbook.Card{}
	m := NewModelBuilder().SetId(7).SetMonsterBookCards(cards).MustBuild()
	if got := m.MonsterBookCards(); len(got) != 0 {
		t.Fatalf("expected empty cards, got %d", len(got))
	}
	// Setter on the model returns a clone carrying the new value.
	m2 := m.SetCoverCardId(item.Id(2380001))
	if m2.CoverCardId() != item.Id(2380001) {
		t.Errorf("cover not threaded through clone")
	}
	if m2.Id() != 7 {
		t.Errorf("id not preserved through clone: %d", m2.Id())
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./character/ -run TestModel_MonsterBookCards -v`
Expected: FAIL — `SetMonsterBookCards`/`MonsterBookCards` undefined.

- [ ] **Step 3: Add the field + getter/setter to `model.go`**

In the `Model` struct (after `coverCardId item.Id`, ~line 62) add:

```go
	monsterBookCards   []monsterbook.Card
```

Add the import `"atlas-channel/monsterbook"` to `model.go`'s import block. Add after `SetCoverCardId` (~line 359):

```go
// MonsterBookCards returns the character's owned monster-book cards. Empty on an
// undecorated model; populated by Processor.MonsterBookDecorator on success.
func (m Model) MonsterBookCards() []monsterbook.Card {
	return m.monsterBookCards
}

// SetMonsterBookCards returns a clone of the model with the card list set.
func (m Model) SetMonsterBookCards(v []monsterbook.Card) Model {
	return CloneModel(m).SetMonsterBookCards(v).MustBuild()
}
```

- [ ] **Step 4: Thread the field through `builder.go`**

Add the import `"atlas-channel/monsterbook"` to `builder.go`. In the `modelBuilder` struct (after `coverCardId item.Id`, ~line 60):

```go
	monsterBookCards   []monsterbook.Card
```

In `CloneModel` (after `coverCardId: m.coverCardId,`, ~line 108):

```go
		monsterBookCards:   m.monsterBookCards,
```

Add the setter (near `SetCoverCardId`, ~line 149):

```go
func (b *modelBuilder) SetMonsterBookCards(v []monsterbook.Card) *modelBuilder { b.monsterBookCards = v; return b }
```

In `Build()` (after `coverCardId: b.coverCardId,`, ~line 194):

```go
		monsterBookCards:   b.monsterBookCards,
```

- [ ] **Step 5: Run to verify it passes**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./character/ -run TestModel_MonsterBookCards -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/character/model.go services/atlas-channel/atlas.com/channel/character/builder.go services/atlas-channel/atlas.com/channel/character/builder_test.go
git commit -m "feat(channel): carry monster-book card list on character model"
```

---

## Task 6: Unified `MonsterBookDecorator` (replaces cover-only decorator)

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/character/processor.go` (interface ~34; impl ~173-183)
- Modify: `services/atlas-channel/atlas.com/channel/character/mock/processor.go` (~75)
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_info_request.go:31`
- Test: `services/atlas-channel/atlas.com/channel/character/processor_test.go`

> `MonsterBookCoverDecorator` is currently used by `character_info_request.go:31` and
> stubbed in the mock — both must be updated, or the build breaks (context.md
> correction #1).

- [ ] **Step 1: Write the failing fail-open test**

Append to `services/atlas-channel/atlas.com/channel/character/processor_test.go` (ensure these imports exist: `"atlas-channel/monsterbook"`, `"context"`, `"net/http"`, `"net/http/httptest"`, `"strings"`, `tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"`, `"github.com/Chronicle20/atlas/libs/atlas-constants/item"`, `"github.com/google/uuid"`, `"github.com/sirupsen/logrus"`):

```go
func TestMonsterBookDecorator_FailOpen(t *testing.T) {
	// Upstream down → decorator returns the model unchanged (cover 0, no cards).
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	defer monsterbook.SetBaseURLForTest(srv.URL)()

	tm, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), tm)
	p := NewProcessor(logrus.New(), ctx)

	m := NewModelBuilder().SetId(42).MustBuild()
	got := p.MonsterBookDecorator(m)
	if got.CoverCardId() != 0 {
		t.Errorf("cover should be 0 on fail-open, got %d", got.CoverCardId())
	}
	if len(got.MonsterBookCards()) != 0 {
		t.Errorf("cards should be empty on fail-open, got %d", len(got.MonsterBookCards()))
	}
}

func TestMonsterBookDecorator_Populates(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/vnd.api+json")
		if strings.HasSuffix(r.URL.Path, "/monster-book/cards") {
			_, _ = w.Write([]byte(`{"data":[{"type":"monster-book-card","id":"2380005","attributes":{"level":2,"isSpecial":false}}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"data":{"type":"monster-book","id":"42","attributes":{"coverCardId":2380001}}}`))
	}))
	defer srv.Close()
	defer monsterbook.SetBaseURLForTest(srv.URL)()

	tm, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), tm)
	p := NewProcessor(logrus.New(), ctx)

	m := NewModelBuilder().SetId(42).MustBuild()
	got := p.MonsterBookDecorator(m)
	if got.CoverCardId() != item.Id(2380001) {
		t.Errorf("cover = %d, want 2380001", got.CoverCardId())
	}
	if len(got.MonsterBookCards()) != 1 || got.MonsterBookCards()[0].CardId() != item.Id(2380005) {
		t.Errorf("cards not populated: %+v", got.MonsterBookCards())
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./character/ -run TestMonsterBookDecorator -v`
Expected: FAIL — `p.MonsterBookDecorator` undefined.

- [ ] **Step 3: Replace the decorator in `processor.go`**

In the `Processor` interface (line 34) replace `MonsterBookCoverDecorator(m Model) Model` with:

```go
	MonsterBookDecorator(m Model) Model
```

Replace the impl (lines 173–183) with:

```go
// MonsterBookDecorator fetches the character's monster-book collection (cover)
// and owned-card list from atlas-monster-book and attaches both to the model.
// Failures (REST 404, network errors) fail open: the model is returned with
// whatever was fetched so far (possibly nothing), so login still succeeds.
func (p *ProcessorImpl) MonsterBookDecorator(m Model) Model {
	mb := monsterbook.NewProcessor(p.l, p.ctx)
	col, err := mb.GetByCharacterId(character.Id(m.Id()))
	if err != nil {
		p.l.WithError(err).Debugf("Unable to fetch monster-book collection for character [%d]; rendering empty book.", m.Id())
		return m
	}
	m = m.SetCoverCardId(col.CoverCardId())
	cards, err := mb.GetCardsByCharacterId(character.Id(m.Id()))
	if err != nil {
		p.l.WithError(err).Debugf("Unable to fetch monster-book cards for character [%d]; rendering cover only.", m.Id())
		return m
	}
	return m.SetMonsterBookCards(cards)
}
```

- [ ] **Step 4: Update the mock**

In `character/mock/processor.go` (line 75) rename:

```go
func (m *MockProcessor) MonsterBookDecorator(c character.Model) character.Model {
	return c
}
```

- [ ] **Step 5: Update the CharacterInfo caller**

In `socket/handler/character_info_request.go` line 31, replace the append with:

```go
		decorators = append(decorators, cp.MonsterBookDecorator)
```

- [ ] **Step 6: Run the package tests to verify they pass**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./character/... ./socket/... -v`
Expected: clean build (no stale `MonsterBookCoverDecorator` references) + PASS.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/character/processor.go services/atlas-channel/atlas.com/channel/character/processor_test.go services/atlas-channel/atlas.com/channel/character/mock/processor.go services/atlas-channel/atlas.com/channel/socket/handler/character_info_request.go
git commit -m "feat(channel): unified MonsterBookDecorator (cover + cards, fail-open)"
```

---

## Task 7: `BuildCharacterData` — populate the monster book

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/socket/writer/character_data.go` (~line 90)
- Test: `services/atlas-channel/atlas.com/channel/socket/writer/character_data_test.go` (create if absent)

- [ ] **Step 1: Write the failing test**

Create or append `services/atlas-channel/atlas.com/channel/socket/writer/character_data_test.go`:

```go
package writer

import (
	"testing"

	"atlas-channel/buddylist"
	"atlas-channel/character"
	"atlas-channel/monsterbook"

	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
)

func TestBuildCharacterData_MonsterBook(t *testing.T) {
	cards := []monsterbook.Card{} // domain cards built via the monsterbook package
	c := character.NewModelBuilder().
		SetId(99).
		SetCoverCardId(item.Id(2380001)).
		SetMonsterBookCards(cards).
		MustBuild()

	cd := BuildCharacterData(c, buddylist.Model{})

	if cd.MonsterBook.CoverCardId != item.Id(2380001) {
		t.Errorf("cover = %d, want 2380001", cd.MonsterBook.CoverCardId)
	}
	if len(cd.MonsterBook.Cards) != len(cards) {
		t.Errorf("card count = %d, want %d", len(cd.MonsterBook.Cards), len(cards))
	}
}
```

> Note: `monsterbook.Card` has unexported fields, so the test uses an empty slice
> (the cover + count assertion proves the wiring). The per-card mapping path is
> covered by Task 4's REST round-trip and Task 2's byte tests. If `BuildCharacterData`
> already has a `_test.go`, append the function instead of recreating the file/header.

- [ ] **Step 2: Run to verify it fails**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./socket/writer/ -run TestBuildCharacterData_MonsterBook -v`
Expected: FAIL — `cd.MonsterBook` not populated (cover 0).

- [ ] **Step 3: Populate `MonsterBook` in `BuildCharacterData`**

In `character_data.go`, after the Quests loops and before `return cd` (~line 90), add:

```go
	// Monster book: cover + owned cards (empty when the decorator failed open).
	cd.MonsterBook.CoverCardId = c.CoverCardId()
	for _, mc := range c.MonsterBookCards() {
		cd.MonsterBook.Cards = append(cd.MonsterBook.Cards, charpkt.MonsterBookCard{
			CardId: mc.CardId(),
			Level:  mc.Level(),
		})
	}
```

- [ ] **Step 4: Run to verify it passes**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./socket/writer/ -run TestBuildCharacterData_MonsterBook -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/socket/writer/character_data.go services/atlas-channel/atlas.com/channel/socket/writer/character_data_test.go
git commit -m "feat(channel): populate CharacterData monster book in BuildCharacterData"
```

---

## Task 8: Wire `MonsterBookDecorator` into the login chain

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/kafka/consumer/session/consumer.go:166`

- [ ] **Step 1: Add the decorator to the `GetById` chain**

At `consumer.go:166`, append `cp.MonsterBookDecorator` to the decorator list:

```go
					c, err := cp.GetById(cp.InventoryDecorator, cp.PetAssetEnrichmentDecorator, cp.SkillModelDecorator, cp.QuestModelDecorator, cp.MonsterBookDecorator)(params.CharacterId)
```

- [ ] **Step 2: Build to verify it compiles**

Run: `cd services/atlas-channel/atlas.com/channel && go build ./...`
Expected: clean build (no output).

- [ ] **Step 3: Run the session consumer tests**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./kafka/consumer/session/... -v`
Expected: PASS (or "no test files" — the decorator itself is covered by Task 6; this is a single chain entry).

- [ ] **Step 4: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/kafka/consumer/session/consumer.go
git commit -m "feat(channel): fetch monster book on login via decorator chain"
```

---

## Task 9: Secondary — CharacterInfo cover (verify + minimal fix)

**Files:**
- Modify: `libs/atlas-packet/character/clientbound/info.go` (constructor ~36; encode ~98; decode ~163)
- Modify: `libs/atlas-packet/character/clientbound/info_test.go`
- Modify: `services/atlas-channel/atlas.com/channel/socket/writer/character_info.go:53`

> The CharacterInfo path already fetches the cover (the decorator runs via
> `character_info_request.go:31`), but `info.go:98` hardcodes `WriteInt(0)`. This task
> threads the real cover through. Cover is written as the **full item id** (consistent
> with CharacterData; confirm live per design §2.3.1).

- [ ] **Step 1: Write the failing test**

In `libs/atlas-packet/character/clientbound/info_test.go`, add (follow the file's existing round-trip helper usage; the key assertion):

```go
func TestCharacterInfo_MonsterBookCover(t *testing.T) {
	ctx := pt.CreateContext("GMS", 83, 1)
	in := NewCharacterInfo(1, 10, 100, 0, "", nil, nil, 0, 2380001) // new trailing cover arg
	out := CharacterInfo{}
	pt.RoundTrip(t, ctx, in.Encode, out.Decode, nil)
	if out.MonsterBookCover() != 2380001 {
		t.Errorf("cover = %d, want 2380001", out.MonsterBookCover())
	}
}
```

> If the existing `info_test.go` already calls `NewCharacterInfo`, update those call
> sites to pass the new trailing `0` cover argument (otherwise the package won't compile).

- [ ] **Step 2: Run to verify it fails**

Run: `cd libs/atlas-packet && go test ./character/clientbound/ -run TestCharacterInfo_MonsterBookCover -v`
Expected: FAIL — constructor arity / `MonsterBookCover` undefined.

- [ ] **Step 3: Thread the cover through `info.go`**

- Add field `monsterBookCover uint32` to the `CharacterInfo` struct and getter:

```go
func (m CharacterInfo) MonsterBookCover() uint32 { return m.monsterBookCover }
```

- Add a trailing `monsterBookCover uint32` param to `NewCharacterInfo` (line 36) and set it in the struct literal.
- At the encode site (line 98) replace `w.WriteInt(0) // cover` with `w.WriteInt(m.monsterBookCover)`.
- At the decode site (line ~163) replace `_ = r.ReadUint32() // cover` with `m.monsterBookCover = r.ReadUint32()`.

- [ ] **Step 4: Update the channel writer caller**

In `socket/writer/character_info.go:53`, pass the cover from the (already decorated) model:

```go
				return charpkt.NewCharacterInfo(
					c.Id(), c.Level(), uint16(c.JobId()), c.Fame(), guildName,
					pets, wishListSNs, medalId, uint32(c.CoverCardId()),
				).Encode(l, ctx)(options)
```

- [ ] **Step 5: Run the tests to verify they pass**

Run:
```bash
cd libs/atlas-packet && go test ./character/clientbound/ -v
cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./socket/... -v
```
Expected: PASS / clean build.

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-packet/character/clientbound/info.go libs/atlas-packet/character/clientbound/info_test.go services/atlas-channel/atlas.com/channel/socket/writer/character_info.go
git commit -m "feat(packet): render monster-book cover in CharacterInfo"
```

---

## Task 10: Full verification

**Files:** none (verification only).

- [ ] **Step 1: Confirm the v95 tail finding (from Task 3 Step 0) is recorded**

Ensure the Task 3 commit message documents whether v95 retains newYear/area/trailing.
If it could not be confirmed against the v95 IDB, note it as an open item for the
in-game v95 acceptance check (PRD §10: "no packet desync on v95").

- [ ] **Step 2: Per-module Go checks**

Run from each changed module:

```bash
cd libs/atlas-constants && go test -race ./... && go vet ./... && go build ./...
cd libs/atlas-packet && go test -race ./... && go vet ./... && go build ./...
cd services/atlas-channel/atlas.com/channel && go test -race ./... && go vet ./... && go build ./...
```
Expected: all clean.

- [ ] **Step 3: Redis key guard**

Run from the worktree root: `tools/redis-key-guard.sh`
Expected: clean (this task adds no Redis usage).

- [ ] **Step 4: Docker bake for the changed service**

Only `atlas-channel`'s service `go.mod` changed (libs are workspace modules; `atlas-monster-book` is unchanged). Run from the worktree root:

```bash
docker buildx bake atlas-channel
```
Expected: build succeeds.

- [ ] **Step 5: Code review**

Run `superpowers:requesting-code-review` (dispatches `plan-adherence-reviewer` + `backend-guidelines-reviewer`; no atlas-ui TS changed). Address findings, re-run Step 2 if code changed.

- [ ] **Step 6: Acceptance (in-game, per PRD §10)**

These are manual/live checks, recorded in the PR description:
- GMS v83 login → Monster Book window shows owned cards at correct levels + cover.
- Set-cover loop: open window → pick an owned card → persists across relog.
- Mid-session card pickup still shows via live `SetCard` and on next login.
- atlas-monster-book down → login still succeeds, window renders empty.
- GMS v95 login → no desync (login completes, character loads).
- Cover renders as a full id (confirm design §2.3.1; flip to `−2380000` offset only if it renders wrong).

---

## Self-review notes

- **Spec coverage:** PRD §4.1 (Tasks 2,7,8), §4.3 set-cover loop (unchanged handler; verified Task 10 §6), §4.4 unified decorator (Task 6), §4.5 gate + surfaces (Tasks 3, 9), §4.6 `/cards` consumption (Task 4), §8 fail-open (Task 6 test), §10 byte tests + verify (Tasks 2,3,10). Design §2 wire format (Tasks 1,2), §3 gate split (Task 3), §4 components (Tasks 4–8), §7 testing (per-task TDD).
- **Out of scope (untouched):** `kafka/consumer/monsterbook/consumer.go` (CARD_ADDED/COVER_CHANGED), `socket/handler/monster_book_cover.go` (SET_COVER), atlas-monster-book persistence.
- **Type consistency:** `MonsterBookData`/`MonsterBookCard` (packet) vs `monsterbook.Card` (channel domain) vs `card.RestModel` (monster-book service) are deliberately distinct; `BuildCharacterData` (Task 7) is the only mapping point. Accessors named consistently: `CardId()`, `Level()`, `IsSpecial()`, `CoverCardId()`, `MonsterBookCards()`, `MonsterBookCover()`.
