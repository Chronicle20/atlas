# Mystic Door Party-State Reconciliation — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the per-event door↔party delta handlers with one convergent `ReconcileParty`, fix the atlas-parties DISBAND event to carry the departing leader, and make the party town-portal array update on reslot — eliminating orphaned doors, the "platform below" flicker, and the stale-array crash precondition.

**Architecture:** Door party scope becomes a pure function of `(owner, current authoritative membership)`. On any party event the atlas-doors consumer resolves authoritative membership and calls `ReconcileParty(partyId, members, joiners, leavers)`, which computes desired state per affected door and emits only the minimal `CREATED`/`REMOVED`/`SLOT_CHANGED` deltas (existing event vocabulary). atlas-parties emits the full former-member list on disband. atlas-channel gains one bounded addition: `handleSlotChanged` reconciles the party town-portal array.

**Tech Stack:** Go, Redis-backed door registry, Kafka status events, `go test`, `docker buildx bake`.

**Design reference:** `docs/tasks/task-093-mystic-door/design-reconciliation.md` (root causes, v95 IDB verification).

---

## File Structure

- `services/atlas-parties/atlas.com/parties/party/processor.go` — **modify** `:427-462` leave/disband path to capture the full member list before `RemoveMember`.
- `services/atlas-parties/atlas.com/parties/party/processor_test.go` — **modify** add disband-includes-leader test.
- `services/atlas-doors/atlas.com/doors/door/reconcile.go` — **create** `ReconcileParty` + `reconcileMemberDoor` + `dropDoorToSolo` helpers.
- `services/atlas-doors/atlas.com/doors/door/reconcile_test.go` — **create** the reconciler test suite.
- `services/atlas-doors/atlas.com/doors/door/processor.go` — **modify** remove `JoinPartyDoor`, `LeavePartyDoor`, `DisbandPartyDoors`, `ShowPartyDoorsToCharacter`, `HidePartyDoorsFromCharacter`.
- `services/atlas-doors/atlas.com/doors/door/reslot.go` — **delete** `ReslotParty` (folded into the reconciler).
- `services/atlas-doors/atlas.com/doors/door/reslot_test.go` — **delete** (tests the removed `ReslotParty`).
- `services/atlas-doors/atlas.com/doors/door/processor_test.go` — **modify** delete the three method tests (`TestLeavePartyDoor…`, `TestJoinPartyDoor…`, `TestDisbandPartyDoors…`).
- `services/atlas-doors/atlas.com/doors/kafka/consumer/party/consumer.go` — **modify** the five handlers to call `ReconcileParty`.
- `services/atlas-channel/atlas.com/channel/kafka/message/door/kafka.go` — **modify** add `AreaX`/`AreaY` to `SlotChangedBody` (additive).
- `services/atlas-channel/atlas.com/channel/kafka/consumer/door/consumer.go` — **modify** `handleSlotChanged` to reconcile the party town-portal array.
- `services/atlas-doors/atlas.com/doors/door/producer.go` — **modify** `slotChangedEventProvider` to populate `AreaX`/`AreaY`; mirror struct field in the doors-side `SlotChangedBody`.

The reconciler is the only new responsibility; everything else is rewire/remove. `Spawn`, `RemoveByOwner`, `RemoveByOwnerIfLeftField`, and the low-level `Reslot` primitive stay.

---

## Reconciler contract (the spec the tests encode)

`ReconcileParty(p, partyId, members, joiners, leavers, townPortalsByMap)`:

- `members`: authoritative post-change ordered list (leader index 0); empty on disband.
- `joiners`: ids that joined THIS event (gain visibility of existing party doors).
- `leavers`: ids that left THIS event (expelled/left/all former members on disband).
- `participants = dedup(members ++ leavers)` — everyone who saw a party-`partyId` door before this change.

Per candidate owner `o ∈ participants`, for each of `o`'s doors `d` where `d.PartyId()==partyId || o ∈ members`:

1. **`o` is a current member, `d` already in `partyId`:** reslot to `ComputeSlot(members,o)` if changed (town/array only — **never** re-send the area door). Otherwise emit nothing.
2. **`o` is a current member, `d` solo/other-party:** adopt into `partyId` at the computed slot; emit a targeted `CREATED` to every OTHER current member (they gain the area door) and a `SLOT_CHANGED` for the owner (town/array transition, no area re-send).
3. **`o` is not a current member (a leaver), `d` still in `partyId`:** emit a targeted `REMOVED` to every OTHER participant, then re-key `d` to solo (party 0, slot 0) and emit a solo `CREATED` (`forCharacterId 0`, reaches only the owner).

Then:
- **Joiners gain visibility:** for each `j ∈ joiners` still in `members`, emit a targeted `CREATED(d, forCharacterId=j)` for every other current member's `partyId` door.
- **Leavers lose visibility:** for each `o ∈ leavers` not in `members`, emit a targeted `REMOVED(d, forCharacterId=o)` for every current member's `partyId` door.

Idempotent: with empty `joiners`/`leavers` and unchanged membership, every branch is a no-op → zero events.

---

## Task 1: atlas-parties — DISBAND event carries the departing leader

**Files:**
- Modify: `services/atlas-parties/atlas.com/parties/party/processor.go:427-462`
- Test: `services/atlas-parties/atlas.com/parties/party/processor_test.go`

- [ ] **Step 1: Read the existing leave/disband flow**

Read `services/atlas-parties/atlas.com/parties/party/processor.go:415-475`. Confirm: `:427 disbandParty = party.LeaderId()==characterId`, `:429 RemoveMember(characterId)`, `:462 disbandEventProvider(characterId, partyId, worldId, party.Members())`. The member list at `:462` is read AFTER removal, so it excludes the leader.

- [ ] **Step 2: Write the failing test**

Add to `processor_test.go` (match the file's existing harness for constructing a 2-member party and capturing emitted events — mirror the nearest existing leave/expel test). The assertion that matters:

```go
// When the leader leaves and the party disbands, the DISBAND event body must
// carry the FULL former member list (including the departing leader), so the
// door service can transition every member's door to solo.
func TestLeaderLeaveDisbandEventIncludesLeader(t *testing.T) {
	// Arrange: party with leader L (id 1) and member M (id 5); capture emitted events.
	// (Use the same registry/builder/emit-capture setup as the existing leave test.)
	leaderId := uint32(1)
	memberId := uint32(5)
	// ... build party {1,5} with leader 1, wire a capturing message buffer ...

	// Act: leader (1) leaves -> party disbands.
	// ... call the leave processor method for characterId=1 ...

	// Assert: a DISBAND event was emitted whose Body.Members == [1,5] (order: leader first).
	got := decodeDisbandMembers(t, capturedMessages) // helper: find the disband event, return Body.Members
	want := []uint32{leaderId, memberId}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("disband Members = %v, want %v (must include the departing leader)", got, want)
	}
}
```

Write `decodeDisbandMembers` to scan the captured kafka messages for `type == EventPartyStatusTypeDisband` and return `Body.Members`.

- [ ] **Step 3: Run the test, verify it fails**

Run: `cd services/atlas-parties/atlas.com/parties && go test ./party/ -run TestLeaderLeaveDisbandEventIncludesLeader -v`
Expected: FAIL — `Members = [5], want [1 5]`.

- [ ] **Step 4: Capture the full member list before removal**

In `processor.go`, immediately before `:427`'s `RemoveMember`, capture the pre-removal list, and use it at `:462`:

```go
var disbandParty = party.LeaderId() == characterId
formerMembers := make([]uint32, 0, len(party.Members()))
for _, m := range party.Members() {
	formerMembers = append(formerMembers, m.Id()) // adjust to the actual member id accessor
}
```

Then change the `:462` emit from `party.Members()` to `formerMembers`:

```go
err = mb.Put(EnvEventStatusTopic, disbandEventProvider(characterId, partyId, c.WorldId(), formerMembers))
```

(`disbandEventProvider` already takes `members []uint32` — `producer.go:108`. Confirm the member-id accessor; if `party.Members()` is `[]Model`, use the same `.Id()`/`uint32(...)` form the file already uses elsewhere.)

- [ ] **Step 5: Run the test, verify it passes**

Run: `cd services/atlas-parties/atlas.com/parties && go test ./party/ -run TestLeaderLeaveDisbandEventIncludesLeader -v`
Expected: PASS.

- [ ] **Step 6: Full module test + commit**

Run: `cd services/atlas-parties/atlas.com/parties && go test ./... && go vet ./...`
Expected: PASS / clean.

```bash
git add services/atlas-parties/atlas.com/parties/party/processor.go services/atlas-parties/atlas.com/parties/party/processor_test.go
git commit -m "fix(atlas-parties): DISBAND event carries the departing leader

Capture the full member list before RemoveMember so the leader-triggered
disband body includes the leaver; door reconciliation needs every former
member to transition each door to solo. Fixes orphaned leader door."
```

---

## Task 2: atlas-doors — `ReconcileParty` (TDD, behavior by behavior)

**Files:**
- Create: `services/atlas-doors/atlas.com/doors/door/reconcile.go`
- Create: `services/atlas-doors/atlas.com/doors/door/reconcile_test.go`

All tests use the existing seam from `processor_test.go` (`newTestProcessor`, `fakeEmit`, `NewBuilder`, `GetRegistry()`). `reconcile_test.go` lives in `package door`, so those helpers are in scope.

- [ ] **Step 1: Write the reconciler skeleton (compiles, does nothing)**

Create `services/atlas-doors/atlas.com/doors/door/reconcile.go`:

```go
package door

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/sirupsen/logrus"
)

// ReconcileParty projects every affected door's party scope from the
// authoritative post-change membership and emits the minimal status deltas.
// It replaces the per-event delta methods (Join/Leave/Disband/Show/Hide/Reslot).
//
// partyId : the party whose membership changed (never 0 here).
// members : authoritative post-change ordered list (leader index 0); empty on disband.
// joiners : ids that joined THIS event (gain visibility of existing party doors).
// leavers : ids that left THIS event (expelled/left/all former members on disband).
func ReconcileParty(
	p *ProcessorImpl,
	partyId uint32,
	members []character.Id,
	joiners []character.Id,
	leavers []character.Id,
	townPortalsByMap func(_map.Id) []TownPortal,
) error {
	inParty := make(map[character.Id]bool, len(members))
	for _, m := range members {
		inParty[m] = true
	}
	participants := dedupIds(append(append([]character.Id{}, members...), leavers...))

	for _, o := range participants {
		doors, err := GetRegistry().GetByOwner(p.ctx, p.t, o)
		if err != nil {
			p.l.WithError(err).Warnf("ReconcileParty: GetByOwner %d", uint32(o))
			continue
		}
		for _, d := range doors {
			if d.PartyId() != partyId && !inParty[o] {
				continue // door not relevant to this party
			}
			if inParty[o] {
				p.reconcileMemberDoor(partyId, members, o, d, townPortalsByMap)
			} else {
				p.dropDoorToSolo(participants, o, d, townPortalsByMap)
			}
		}
	}

	// Joiners gain visibility of every OTHER current member's party door.
	for _, j := range joiners {
		if !inParty[j] {
			continue
		}
		p.showPartyDoorsTo(partyId, members, j)
	}
	// Leavers lose visibility of every current member's party door.
	for _, o := range leavers {
		if inParty[o] {
			continue
		}
		p.hidePartyDoorsFrom(partyId, members, o)
	}
	return nil
}

func dedupIds(ids []character.Id) []character.Id {
	seen := make(map[character.Id]bool, len(ids))
	out := make([]character.Id, 0, len(ids))
	for _, id := range ids {
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}

func (p *ProcessorImpl) reconcileMemberDoor(partyId uint32, members []character.Id, owner character.Id, d Model, townPortalsByMap func(_map.Id) []TownPortal) {
	desiredSlot := ComputeSlot(partyId, members, owner)
	wireId, tx, ty, _ := ResolveTownPortal(townPortalsByMap(d.TownMapId()), desiredSlot, defaultTownX, defaultTownY)

	if d.PartyId() == partyId {
		if d.Slot() == desiredSlot {
			return
		}
		if err := p.Reslot(d.AreaDoorId(), desiredSlot, wireId, tx, ty); err != nil {
			p.l.WithError(err).Warnf("ReconcileParty: reslot door %d", d.AreaDoorId())
		}
		return
	}

	// Adopt solo/other-party door into this party.
	oldSlot := d.Slot()
	n := Clone(d).SetPartyId(partyId).SetSlot(desiredSlot).
		SetTownPortalId(wireId).SetTownX(tx).SetTownY(ty).Build()
	if err := GetRegistry().Put(p.ctx, p.t, n); err != nil {
		p.l.WithError(err).Warnf("ReconcileParty: adopt persist door %d", d.AreaDoorId())
		return
	}
	for _, m := range members {
		if m == owner {
			continue // owner already renders the area door; no re-send (no flicker)
		}
		_ = p.emit(EnvEventTopicDoorStatus, createdEventProvider(n, uint32(m)))
	}
	// Owner: town/array-only transition (clears old slot, sets new array slot).
	_ = p.emit(EnvEventTopicDoorStatus, slotChangedEventProvider(n, oldSlot))
	p.l.WithFields(logrus.Fields{
		"door_action": "reconcile_adopt", "party_id": partyId, "owner": uint32(owner),
		"area_door_id": d.AreaDoorId(), "old_slot": oldSlot, "new_slot": desiredSlot,
	}).Infof("ReconcileParty: adopted door [%d] -> party [%d] slot [%d].", d.AreaDoorId(), partyId, desiredSlot)
}

func (p *ProcessorImpl) dropDoorToSolo(participants []character.Id, owner character.Id, d Model, townPortalsByMap func(_map.Id) []TownPortal) {
	for _, m := range participants {
		if m == owner {
			continue
		}
		_ = p.emit(EnvEventTopicDoorStatus, removedEventProvider(d, RemoveReasonPartyLeft, uint32(m)))
	}
	wireId, tx, ty, _ := ResolveTownPortal(townPortalsByMap(d.TownMapId()), 0, defaultTownX, defaultTownY)
	n := Clone(d).SetPartyId(0).SetSlot(0).SetTownPortalId(wireId).SetTownX(tx).SetTownY(ty).Build()
	if err := GetRegistry().Put(p.ctx, p.t, n); err != nil {
		p.l.WithError(err).Warnf("ReconcileParty: solo persist door %d", d.AreaDoorId())
		return
	}
	_ = p.emit(EnvEventTopicDoorStatus, createdEventProvider(n, 0))
	p.l.WithFields(logrus.Fields{
		"door_action": "reconcile_solo", "owner": uint32(owner), "area_door_id": d.AreaDoorId(),
	}).Infof("ReconcileParty: door [%d] -> solo.", d.AreaDoorId())
}

func (p *ProcessorImpl) showPartyDoorsTo(partyId uint32, members []character.Id, target character.Id) {
	for _, m := range members {
		if m == target {
			continue
		}
		doors, err := GetRegistry().GetByOwner(p.ctx, p.t, m)
		if err != nil {
			continue
		}
		for _, d := range doors {
			if d.PartyId() != partyId {
				continue
			}
			_ = p.emit(EnvEventTopicDoorStatus, createdEventProvider(d, uint32(target)))
		}
	}
}

func (p *ProcessorImpl) hidePartyDoorsFrom(partyId uint32, members []character.Id, target character.Id) {
	for _, m := range members {
		doors, err := GetRegistry().GetByOwner(p.ctx, p.t, m)
		if err != nil {
			continue
		}
		for _, d := range doors {
			if d.PartyId() != partyId {
				continue
			}
			_ = p.emit(EnvEventTopicDoorStatus, removedEventProvider(d, RemoveReasonPartyLeft, uint32(target)))
		}
	}
}
```

- [ ] **Step 2: Verify it compiles**

Run: `cd services/atlas-doors/atlas.com/doors && go build ./...`
Expected: clean (uses only existing `Clone`, `ComputeSlot`, `ResolveTownPortal`, `defaultTownX/Y`, providers, `RemoveReasonPartyLeft`).

- [ ] **Step 3: Write the expel test (drop-to-solo + cross-hide)**

Add to `reconcile_test.go`:

```go
package door

import (
	"encoding/json"
	"testing"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
)

// helper: decode (type, owner, partyId, forCharacterId) for an emitted event.
func decodeEvt(b []byte) (typ string, owner, party, forCh uint32) {
	var env struct {
		Type             string `json:"type"`
		OwnerCharacterId uint32 `json:"ownerCharacterId"`
		PartyId          uint32 `json:"partyId"`
		ForCharacterId   uint32 `json:"forCharacterId"`
	}
	_ = json.Unmarshal(b, &env)
	return env.Type, env.OwnerCharacterId, env.PartyId, env.ForCharacterId
}

func twoPartyPortals() []TownPortal {
	return []TownPortal{{X: 10, Y: 20}, {X: -85, Y: 531}}
}

func TestReconcileExpelDropsLeaverToSoloAndCrossHides(t *testing.T) {
	em := &fakeEmit{}
	p, ten, ctx := newTestProcessor(t, fakeResolver{}, &counterAllocator{next: 1}, em)
	f := field.NewBuilder(1, 2, 240011000).Build()

	// Leader Chronicle (1) slot 0, Bishop (5) slot 1 — both in party 1000000008.
	chron := NewBuilder().SetAreaDoorId(1).SetTownDoorId(2).SetOwnerCharacterId(1).
		SetPartyId(1000000008).SetField(f).SetTownMapId(_map.Id(240000000)).SetSlot(0).
		SetTownPortalId(0x80).SetTownX(10).SetTownY(20).Build()
	bishop := NewBuilder().SetAreaDoorId(3).SetTownDoorId(4).SetOwnerCharacterId(5).
		SetPartyId(1000000008).SetField(f).SetTownMapId(_map.Id(240000000)).SetSlot(1).
		SetTownPortalId(0x81).SetTownX(-85).SetTownY(531).Build()
	for _, m := range []Model{chron, bishop} {
		if err := GetRegistry().Put(ctx, ten, m); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	// Chronicle expels Bishop: members=[1], leavers=[5].
	if err := ReconcileParty(p, 1000000008, []character.Id{1}, nil, []character.Id{5},
		func(_ _map.Id) []TownPortal { return twoPartyPortals() }); err != nil {
		t.Fatalf("reconcile: %v", err)
	}

	// Expect: Bishop's door REMOVED from Chronicle (forCh=1), Bishop solo CREATED (forCh=0),
	// Chronicle's door REMOVED from Bishop (forCh=5). Order: drop-to-solo first, then hide.
	gotRemovedFromChron, gotBishopSolo, gotRemovedFromBishop := false, false, false
	for _, v := range em.values {
		typ, owner, party, forCh := decodeEvt(v)
		if typ == EventDoorStatusRemoved && owner == 5 && party == 1000000008 && forCh == 1 {
			gotRemovedFromChron = true
		}
		if typ == EventDoorStatusCreated && owner == 5 && party == 0 && forCh == 0 {
			gotBishopSolo = true
		}
		if typ == EventDoorStatusRemoved && owner == 1 && party == 1000000008 && forCh == 5 {
			gotRemovedFromBishop = true
		}
	}
	if !gotRemovedFromChron || !gotBishopSolo || !gotRemovedFromBishop {
		t.Fatalf("expel deltas missing: rmFromChron=%v bishopSolo=%v rmFromBishop=%v events=%v",
			gotRemovedFromChron, gotBishopSolo, gotRemovedFromBishop, em.types)
	}
	// Bishop's door is now solo at slot 0.
	got, _ := GetRegistry().Get(ctx, ten, 3)
	if got.PartyId() != 0 || got.Slot() != 0 {
		t.Fatalf("bishop door not solo: party=%d slot=%d", got.PartyId(), got.Slot())
	}
}
```

- [ ] **Step 4: Run the expel test**

Run: `cd services/atlas-doors/atlas.com/doors && go test ./door/ -run TestReconcileExpelDropsLeaverToSoloAndCrossHides -v`
Expected: PASS.

- [ ] **Step 5: Write the disband test (leader-leave: both dropped, cross-removed)**

```go
func TestReconcileDisbandDropsAllAndCrossRemoves(t *testing.T) {
	em := &fakeEmit{}
	p, ten, ctx := newTestProcessor(t, fakeResolver{}, &counterAllocator{next: 1}, em)
	f := field.NewBuilder(1, 2, 240011000).Build()
	chron := NewBuilder().SetAreaDoorId(1).SetTownDoorId(2).SetOwnerCharacterId(1).
		SetPartyId(1000000008).SetField(f).SetTownMapId(_map.Id(240000000)).SetSlot(0).Build()
	bishop := NewBuilder().SetAreaDoorId(3).SetTownDoorId(4).SetOwnerCharacterId(5).
		SetPartyId(1000000008).SetField(f).SetTownMapId(_map.Id(240000000)).SetSlot(1).Build()
	for _, m := range []Model{chron, bishop} {
		_ = GetRegistry().Put(ctx, ten, m)
	}

	// Leader leaves -> disband: members empty, leavers=[1,5] (the atlas-parties fix supplies both).
	_ = ReconcileParty(p, 1000000008, nil, nil, []character.Id{1, 5},
		func(_ _map.Id) []TownPortal { return twoPartyPortals() })

	// Each door removed from the OTHER former member; each re-keyed solo.
	rmChronFromBishop, rmBishopFromChron := false, false
	for _, v := range em.values {
		typ, owner, party, forCh := decodeEvt(v)
		if typ == EventDoorStatusRemoved && owner == 1 && party == 1000000008 && forCh == 5 {
			rmChronFromBishop = true
		}
		if typ == EventDoorStatusRemoved && owner == 5 && party == 1000000008 && forCh == 1 {
			rmBishopFromChron = true
		}
	}
	if !rmChronFromBishop || !rmBishopFromChron {
		t.Fatalf("disband cross-removal missing: chron->bishop=%v bishop->chron=%v events=%v",
			rmChronFromBishop, rmBishopFromChron, em.types)
	}
	for _, id := range []uint32{1, 3} {
		got, _ := GetRegistry().Get(ctx, ten, id)
		if got.PartyId() != 0 || got.Slot() != 0 {
			t.Fatalf("door %d not solo after disband: party=%d slot=%d", id, got.PartyId(), got.Slot())
		}
	}
}
```

- [ ] **Step 6: Run the disband test**

Run: `cd services/atlas-doors/atlas.com/doors && go test ./door/ -run TestReconcileDisbandDropsAllAndCrossRemoves -v`
Expected: PASS.

- [ ] **Step 7: Write the reinvite/adopt test (no owner area re-send = flicker fix)**

```go
func TestReconcileReinviteAdoptsWithoutOwnerAreaResend(t *testing.T) {
	em := &fakeEmit{}
	p, ten, ctx := newTestProcessor(t, fakeResolver{}, &counterAllocator{next: 1}, em)
	f := field.NewBuilder(1, 2, 240011000).Build()
	// Chronicle (1) in party; Bishop (5) currently SOLO (post-expel).
	chron := NewBuilder().SetAreaDoorId(1).SetTownDoorId(2).SetOwnerCharacterId(1).
		SetPartyId(1000000008).SetField(f).SetTownMapId(_map.Id(240000000)).SetSlot(0).Build()
	bishopSolo := NewBuilder().SetAreaDoorId(3).SetTownDoorId(4).SetOwnerCharacterId(5).
		SetPartyId(0).SetField(f).SetTownMapId(_map.Id(240000000)).SetSlot(0).Build()
	for _, m := range []Model{chron, bishopSolo} {
		_ = GetRegistry().Put(ctx, ten, m)
	}

	// Bishop rejoins: members=[1,5], joiners=[5].
	_ = ReconcileParty(p, 1000000008, []character.Id{1, 5}, []character.Id{5}, nil,
		func(_ _map.Id) []TownPortal { return twoPartyPortals() })

	// FLICKER GUARD: Bishop (owner 5) must NOT receive a CREATED for his OWN door (owner 5).
	for _, v := range em.values {
		typ, owner, _, forCh := decodeEvt(v)
		if typ == EventDoorStatusCreated && owner == 5 && forCh == 5 {
			t.Fatalf("owner 5 got a CREATED for his own door (area re-send / platform-below flicker): %v", em.types)
		}
	}
	// Chronicle gains Bishop's door (CREATED owner 5 -> forCh 1); Bishop gains Chronicle's door (CREATED owner 1 -> forCh 5).
	chronGainsBishop, bishopGainsChron := false, false
	for _, v := range em.values {
		typ, owner, _, forCh := decodeEvt(v)
		if typ == EventDoorStatusCreated && owner == 5 && forCh == 1 {
			chronGainsBishop = true
		}
		if typ == EventDoorStatusCreated && owner == 1 && forCh == 5 {
			bishopGainsChron = true
		}
	}
	if !chronGainsBishop || !bishopGainsChron {
		t.Fatalf("reinvite visibility missing: chronGainsBishop=%v bishopGainsChron=%v events=%v",
			chronGainsBishop, bishopGainsChron, em.types)
	}
	// Bishop's door is back in the party at slot 1.
	got, _ := GetRegistry().Get(ctx, ten, 3)
	if got.PartyId() != 1000000008 || got.Slot() != 1 {
		t.Fatalf("bishop door not re-adopted at slot 1: party=%d slot=%d", got.PartyId(), got.Slot())
	}
}
```

- [ ] **Step 8: Run the reinvite test**

Run: `cd services/atlas-doors/atlas.com/doors && go test ./door/ -run TestReconcileReinviteAdoptsWithoutOwnerAreaResend -v`
Expected: PASS.

- [ ] **Step 9: Write the idempotency + orphan-heal tests**

```go
func TestReconcileIsIdempotent(t *testing.T) {
	em := &fakeEmit{}
	p, ten, ctx := newTestProcessor(t, fakeResolver{}, &counterAllocator{next: 1}, em)
	f := field.NewBuilder(1, 2, 240011000).Build()
	chron := NewBuilder().SetAreaDoorId(1).SetTownDoorId(2).SetOwnerCharacterId(1).
		SetPartyId(1000000008).SetField(f).SetTownMapId(_map.Id(240000000)).SetSlot(0).Build()
	bishop := NewBuilder().SetAreaDoorId(3).SetTownDoorId(4).SetOwnerCharacterId(5).
		SetPartyId(1000000008).SetField(f).SetTownMapId(_map.Id(240000000)).SetSlot(1).Build()
	for _, m := range []Model{chron, bishop} {
		_ = GetRegistry().Put(ctx, ten, m)
	}
	// Steady-state reconcile (no joiners/leavers, slots already correct) emits nothing.
	_ = ReconcileParty(p, 1000000008, []character.Id{1, 5}, nil, nil,
		func(_ _map.Id) []TownPortal { return twoPartyPortals() })
	if len(em.types) != 0 {
		t.Fatalf("steady-state reconcile must emit nothing, got %v", em.types)
	}
}

func TestReconcileHealsOrphanTaggedToDeadParty(t *testing.T) {
	em := &fakeEmit{}
	p, ten, ctx := newTestProcessor(t, fakeResolver{}, &counterAllocator{next: 1}, em)
	f := field.NewBuilder(1, 2, 240011000).Build()
	// Owner 1 is a current member of party 1000000009, but their door is still
	// tagged to a DEAD party 1000000008 (orphan).
	orphan := NewBuilder().SetAreaDoorId(1).SetTownDoorId(2).SetOwnerCharacterId(1).
		SetPartyId(1000000008).SetField(f).SetTownMapId(_map.Id(240000000)).SetSlot(0).Build()
	_ = GetRegistry().Put(ctx, ten, orphan)

	_ = ReconcileParty(p, 1000000009, []character.Id{1}, nil, nil,
		func(_ _map.Id) []TownPortal { return twoPartyPortals() })

	got, _ := GetRegistry().Get(ctx, ten, 1)
	if got.PartyId() != 1000000009 {
		t.Fatalf("orphan not healed into current party: party=%d", got.PartyId())
	}
}
```

- [ ] **Step 10: Run the idempotency + orphan tests**

Run: `cd services/atlas-doors/atlas.com/doors && go test ./door/ -run 'TestReconcileIsIdempotent|TestReconcileHealsOrphan' -v`
Expected: PASS.

- [ ] **Step 11: Write the slot-bound test (never emit slot > 5)**

```go
func TestReconcileNeverEmitsSlotAbove5(t *testing.T) {
	em := &fakeEmit{}
	p, ten, ctx := newTestProcessor(t, fakeResolver{}, &counterAllocator{next: 1}, em)
	f := field.NewBuilder(1, 2, 240011000).Build()
	members := []character.Id{1, 2, 3, 4, 5, 6} // full 6-cap party
	for i, owner := range members {
		_ = GetRegistry().Put(ctx, ten, NewBuilder().
			SetAreaDoorId(uint32(10+i)).SetTownDoorId(uint32(100+i)).SetOwnerCharacterId(owner).
			SetPartyId(0).SetField(f).SetTownMapId(_map.Id(240000000)).SetSlot(0).Build())
	}
	_ = ReconcileParty(p, 1000000010, members, members, nil,
		func(_ _map.Id) []TownPortal { return []TownPortal{{1, 1}, {2, 2}, {3, 3}, {4, 4}, {5, 5}, {6, 6}} })

	for _, v := range em.values {
		var env struct {
			Body struct {
				Slot    byte `json:"slot"`
				NewSlot byte `json:"newSlot"`
			} `json:"body"`
		}
		_ = json.Unmarshal(v, &env)
		if env.Body.Slot > 5 || env.Body.NewSlot > 5 {
			t.Fatalf("emitted slot > 5 (client-kill): slot=%d newSlot=%d", env.Body.Slot, env.Body.NewSlot)
		}
	}
}
```

- [ ] **Step 12: Run the slot-bound test + the full reconcile suite**

Run: `cd services/atlas-doors/atlas.com/doors && go test ./door/ -run TestReconcile -v`
Expected: all PASS.

- [ ] **Step 13: Commit**

```bash
git add services/atlas-doors/atlas.com/doors/door/reconcile.go services/atlas-doors/atlas.com/doors/door/reconcile_test.go
git commit -m "feat(atlas-doors): add convergent ReconcileParty door projection

Single membership-driven reconciler replacing the per-event delta methods:
adopt (no owner area re-send -> fixes platform-below flicker), reslot,
drop-to-solo with cross-member removal, joiner/leaver visibility, idempotent,
orphan self-heal, slot<=5 guaranteed."
```

---

## Task 3: atlas-doors — rewire party consumers to `ReconcileParty`; remove old methods

**Files:**
- Modify: `services/atlas-doors/atlas.com/doors/kafka/consumer/party/consumer.go`
- Modify: `services/atlas-doors/atlas.com/doors/door/processor.go` (delete 5 methods)
- Delete: `services/atlas-doors/atlas.com/doors/door/reslot.go`, `services/atlas-doors/atlas.com/doors/door/reslot_test.go`
- Modify: `services/atlas-doors/atlas.com/doors/door/processor_test.go` (delete 3 method tests)

- [ ] **Step 1: Rewire the five handlers**

Replace the bodies of `handleJoined/handleLeft/handleExpel/handleDisband/handleChangeLeader` in `consumer.go` so each resolves authoritative membership and calls `ReconcileParty`. The `reslotAfterMembership` helper is removed (folded in). New handler bodies:

```go
func handleJoined(l logrus.FieldLogger) message.Handler[StatusEvent[JoinedEventBody]] {
	return func(_ logrus.FieldLogger, ctx context.Context, e StatusEvent[JoinedEventBody]) {
		if e.Type != EventPartyStatusTypeJoined {
			return
		}
		pm, err := party.NewProcessor(l, ctx).GetById(e.PartyId)
		if err != nil {
			l.WithError(err).Warnf("handleJoined: party %d not found", e.PartyId)
			return
		}
		_ = enginedoor.ReconcileParty(enginedoor.NewProcessor(l, ctx), e.PartyId,
			pm.Members(), []character.Id{character.Id(e.ActorId)}, nil, townPortalsForMap(l, ctx))
	}
}

func handleLeft(l logrus.FieldLogger) message.Handler[StatusEvent[LeftEventBody]] {
	return func(_ logrus.FieldLogger, ctx context.Context, e StatusEvent[LeftEventBody]) {
		if e.Type != EventPartyStatusTypeLeft {
			return
		}
		var members []character.Id
		if pm, err := party.NewProcessor(l, ctx).GetById(e.PartyId); err == nil {
			members = pm.Members()
		}
		_ = enginedoor.ReconcileParty(enginedoor.NewProcessor(l, ctx), e.PartyId,
			members, nil, []character.Id{character.Id(e.ActorId)}, townPortalsForMap(l, ctx))
	}
}

func handleExpel(l logrus.FieldLogger) message.Handler[StatusEvent[ExpelEventBody]] {
	return func(_ logrus.FieldLogger, ctx context.Context, e StatusEvent[ExpelEventBody]) {
		if e.Type != EventPartyStatusTypeExpel {
			return
		}
		var members []character.Id
		if pm, err := party.NewProcessor(l, ctx).GetById(e.PartyId); err == nil {
			members = pm.Members()
		}
		_ = enginedoor.ReconcileParty(enginedoor.NewProcessor(l, ctx), e.PartyId,
			members, nil, []character.Id{e.Body.CharacterId}, townPortalsForMap(l, ctx))
	}
}

func handleDisband(l logrus.FieldLogger) message.Handler[StatusEvent[DisbandEventBody]] {
	return func(_ logrus.FieldLogger, ctx context.Context, e StatusEvent[DisbandEventBody]) {
		if e.Type != EventPartyStatusTypeDisband {
			return
		}
		_ = enginedoor.ReconcileParty(enginedoor.NewProcessor(l, ctx), e.PartyId,
			nil, nil, e.Body.Members, townPortalsForMap(l, ctx))
	}
}

func handleChangeLeader(l logrus.FieldLogger) message.Handler[StatusEvent[ChangeLeaderEventBody]] {
	return func(_ logrus.FieldLogger, ctx context.Context, e StatusEvent[ChangeLeaderEventBody]) {
		if e.Type != EventPartyStatusTypeChangeLeader {
			return
		}
		pm, err := party.NewProcessor(l, ctx).GetById(e.PartyId)
		if err != nil {
			l.WithError(err).Warnf("handleChangeLeader: party %d not found", e.PartyId)
			return
		}
		_ = enginedoor.ReconcileParty(enginedoor.NewProcessor(l, ctx), e.PartyId,
			pm.Members(), nil, nil, townPortalsForMap(l, ctx))
	}
}
```

Confirm the imports: `character` (`atlas-constants/character`) is needed for `character.Id(e.ActorId)`; `e.ActorId`/`e.Body.CharacterId` types — if `ActorId` is `uint32`, wrap with `character.Id(...)`; if `e.Body.CharacterId` is already `character.Id` (it is on the consumer side — see `kafka.go`), pass it directly. Remove the now-unused `reslotAfterMembership` function.

- [ ] **Step 2: Delete `ReslotParty` and the five processor methods**

Delete `services/atlas-doors/atlas.com/doors/door/reslot.go` and `reslot_test.go` entirely. In `processor.go`, delete the method bodies for `JoinPartyDoor`, `LeavePartyDoor`, `DisbandPartyDoors`, `ShowPartyDoorsToCharacter`, `HidePartyDoorsFromCharacter` (lines ~202-392). Keep `Spawn`, `RemoveByOwner`, `RemoveByOwnerIfLeftField`, `Reslot`.

```bash
git rm services/atlas-doors/atlas.com/doors/door/reslot.go services/atlas-doors/atlas.com/doors/door/reslot_test.go
```

- [ ] **Step 3: Delete the three obsolete method tests**

In `processor_test.go`, delete `TestLeavePartyDoorRemovesFromPartyThenRekeysSolo`, `TestJoinPartyDoorAdoptsSoloDoorIntoParty`, and `TestDisbandPartyDoorsRemovesFromOthersThenRekeysSolo` (the reconciler suite covers these behaviors).

- [ ] **Step 4: Build + vet to catch dangling references**

Run: `cd services/atlas-doors/atlas.com/doors && go build ./... && go vet ./...`
Expected: clean. If `go vet` flags an unused import (`character`, `_map`) in `consumer.go`, fix it.

- [ ] **Step 5: Full doors module test**

Run: `cd services/atlas-doors/atlas.com/doors && go test -race ./...`
Expected: PASS (reconcile suite + remaining processor/registry tests).

- [ ] **Step 6: Commit**

```bash
git add -A services/atlas-doors/atlas.com/doors
git commit -m "refactor(atlas-doors): route party events through ReconcileParty

Collapse the five delta handlers (Join/Leave/Disband/Show/Hide) and
ReslotParty into one membership-driven reconcile call per party event.
Removes the non-convergent delta methods and their tests."
```

---

## Task 4: atlas-channel — `handleSlotChanged` reconciles the party town-portal array

The reconciler's adopt path and in-party reslots emit `SLOT_CHANGED`. Today `handleSlotChanged` updates only the solo town portal, never the authoritative party array (`OnPartyResult case 46` / v83 `0x25`). It must clear the old slot and set the new slot so reslots/adopts render in party mode.

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/kafka/message/door/kafka.go` (add `AreaX`/`AreaY` to `SlotChangedBody`)
- Modify: `services/atlas-doors/atlas.com/doors/door/producer.go` + the doors-side `SlotChangedBody` (populate `AreaX`/`AreaY`)
- Modify: `services/atlas-channel/atlas.com/channel/kafka/consumer/door/consumer.go` (`handleSlotChanged`)

- [ ] **Step 1: Add `AreaX`/`AreaY` to both `SlotChangedBody` structs**

The party array entry encodes the door's AREA position (mirrors `handleCreated`, which passes `b.AreaX, b.AreaY`). `SlotChangedBody` lacks it. Add the fields on both sides of the contract.

In `services/atlas-channel/atlas.com/channel/kafka/message/door/kafka.go`, add to `SlotChangedBody`:
```go
	AreaX int16 `json:"areaX"`
	AreaY int16 `json:"areaY"`
```
Mirror the identical two fields in the doors-side `SlotChangedBody` (find it in `services/atlas-doors/atlas.com/doors/door/` — same package as `slotChangedEventProvider`).

- [ ] **Step 2: Populate `AreaX`/`AreaY` in `slotChangedEventProvider`**

In `services/atlas-doors/atlas.com/doors/door/producer.go`, add to the `SlotChangedBody{...}` literal in `slotChangedEventProvider`:
```go
		AreaX: m.AreaX(), AreaY: m.AreaY(),
```

- [ ] **Step 3: Write the failing channel test for the array update**

In `services/atlas-channel/atlas.com/channel/kafka/consumer/door/consumer_test.go`, add a test that drives `handleSlotChanged` with a party-scoped event and asserts a party town-portal clear (old slot) AND set (new slot) are announced. Mirror the existing door consumer test harness (it already stubs `announceTownPortalToParty` / session enumeration — reuse that stub to capture calls). Assert: one clear for `OldSlot`, one set for `NewSlot` with `TownMapId`/`MapId`/`AreaX`/`AreaY`.

```go
func TestHandleSlotChangedUpdatesPartyTownPortalArray(t *testing.T) {
	// Arrange: stub announceTownPortalToParty to record (slot, clear) tuples.
	var calls []struct {
		slot  byte
		clear bool
	}
	orig := announceTownPortalToParty
	announceTownPortalToParty = func(_ logrus.FieldLogger, _ context.Context, _ writer.Producer, _ server.Model, partyId uint32, slot byte, _, _ mapc.Id, _, _ int16, clear bool) {
		calls = append(calls, struct {
			slot  byte
			clear bool
		}{slot, clear})
	}
	defer func() { announceTownPortalToParty = orig }()

	// ... build a SLOT_CHANGED StatusEvent (partyId != 0, OldSlot=0, NewSlot=1, AreaX/AreaY set) ...
	// ... call handleSlotChanged(sc, wp)(l, ctx, e) ...

	// Assert: a clear for slot 0 and a set for slot 1 were announced.
	var sawClearOld, sawSetNew bool
	for _, c := range calls {
		if c.slot == 0 && c.clear {
			sawClearOld = true
		}
		if c.slot == 1 && !c.clear {
			sawSetNew = true
		}
	}
	if !sawClearOld || !sawSetNew {
		t.Fatalf("expected clear(old=0) and set(new=1), got %+v", calls)
	}
}
```

- [ ] **Step 4: Run it, verify it fails**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./kafka/consumer/door/ -run TestHandleSlotChangedUpdatesPartyTownPortalArray -v`
Expected: FAIL (no array calls yet).

- [ ] **Step 5: Add the array reconciliation to `handleSlotChanged`**

In `services/atlas-channel/atlas.com/channel/kafka/consumer/door/consumer.go`, after the existing town-field `RemoveTownDoor`/`SpawnPortal` broadcasts in `handleSlotChanged`, add (guarded by a real party + broadcast, mirroring `handleCreated`):

```go
	// PARTY town render path: move this member's town-portal array slot —
	// clear the old slot and set the new one so in-party viewers re-render at
	// the new slot (the v83/v95 client reads town doors from this array when in
	// a party; a stale slot ghosts, and the slot is bounded 0..5 by the client).
	if e.ForCharacterId == 0 && e.PartyId != 0 {
		announceTownPortalToParty(l, ctx, wp, sc, e.PartyId, b.OldSlot, 0, 0, 0, 0, true)
		announceTownPortalToParty(l, ctx, wp, sc, e.PartyId, b.NewSlot, b.TownMapId, e.MapId, b.AreaX, b.AreaY, false)
	}
```

(The existing `slot >= 6` guard inside `announceTownPortalToParty` at `consumer.go:139` already protects the set call.)

- [ ] **Step 6: Run it, verify it passes**

Run: `cd services/atlas-channel/atlas.com/channel && go test ./kafka/consumer/door/ -run TestHandleSlotChangedUpdatesPartyTownPortalArray -v`
Expected: PASS.

- [ ] **Step 7: Full channel + doors module tests**

Run:
```
cd services/atlas-channel/atlas.com/channel && go test -race ./... && go vet ./...
cd services/atlas-doors/atlas.com/doors && go test -race ./... && go vet ./...
```
Expected: PASS / clean.

- [ ] **Step 8: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/kafka/message/door/kafka.go services/atlas-channel/atlas.com/channel/kafka/consumer/door/consumer.go services/atlas-channel/atlas.com/channel/kafka/consumer/door/consumer_test.go services/atlas-doors/atlas.com/doors/door/producer.go services/atlas-doors/atlas.com/doors/door/kafka.go
git commit -m "fix(atlas-channel): reslot updates the party town-portal array

handleSlotChanged now clears the old slot and sets the new slot in the
authoritative party town-portal array (OnPartyResult case 46 / v83 0x25),
so in-party door reslots/adopts render correctly and leave no stale slot.
Adds AreaX/AreaY to SlotChangedBody to carry the array position."
```

---

## Task 5: Full verification gate

- [ ] **Step 1: Per-module race tests + vet**

Run, expecting clean in each:
```
cd services/atlas-doors/atlas.com/doors && go test -race ./... && go vet ./...
cd services/atlas-parties/atlas.com/parties && go test -race ./... && go vet ./...
cd services/atlas-channel/atlas.com/channel && go test -race ./... && go vet ./...
```

- [ ] **Step 2: Build all three services**

```
cd services/atlas-doors/atlas.com/doors && go build ./...
cd services/atlas-parties/atlas.com/parties && go build ./...
cd services/atlas-channel/atlas.com/channel && go build ./...
```

- [ ] **Step 3: Redis key guard**

Run from the worktree root: `GOWORK=off tools/redis-key-guard.sh`
Expected: clean (no new raw keyed go-redis calls — the reconciler uses `GetRegistry()` only).

- [ ] **Step 4: Docker bake the three touched services**

Run from the worktree root:
```
docker buildx bake atlas-doors atlas-parties atlas-channel
```
Expected: all three build (catches any missing `COPY libs/...` in the shared Dockerfile).

- [ ] **Step 5: Final commit (if any cleanup) and hand off to code review**

```bash
git status   # expect clean
```

Then run the code-review step (`superpowers:requesting-code-review`) before opening the PR — do not skip (project rule).

---

## Self-Review notes

- **Spec coverage:** disband-leader fix → Task 1; `ReconcileParty` (adopt/reslot/drop/joiner/leaver/idempotent/orphan/slot-bound) → Task 2; consumer rewire + delete delta methods → Task 3; authoritative party array on reslot → Task 4; full gate → Task 5. The design's "channel unchanged" claim is amended: Task 4 is one bounded `handleSlotChanged` addition (noted in the design file).
- **Manual end-to-end:** after deploy to `atlas-pr-769`, re-run the reported scenario (cast/cast/expel/reinvite/leader-leave/recreate-join) and confirm via Loki: no `reconcile_solo`/`reconcile_adopt` door left tagged to a dead party, no "unable to resolve party" on the tick, and no client disconnect.
- **Type consistency:** `ReconcileParty(p, partyId, members, joiners, leavers, townPortalsByMap)` is used identically in Tasks 2 and 3; `slotChangedEventProvider(n, oldSlot)` and the `AreaX/AreaY` `SlotChangedBody` fields match across producer (Task 4 Step 2) and channel consumer (Task 4 Step 5).
