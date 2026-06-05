# FR-10: Login-Draw Monster Book Cover — Decision Record

## Decision

The login-draw `CharacterData` cover field stays in **card-id space**. No changes are made to:

- `libs/atlas-packet/character/data.go` (`encodeMonsterBook`)
- `services/atlas-channel/atlas.com/channel/socket/writer/character_data.go`

## Behavioral Evidence

The live v83 client crash was observed **only** when opening the Character Info panel (`OnCharacterInfo`), never at login or map-entry, despite a cover being set. The monster-book window resolves card→mob client-side from the cover card id. This strongly indicates the login `CharacterData` path does not call `CMobTemplate::GetMobTemplate` on the cover field.

## Regression Guard

Test: `services/atlas-channel/atlas.com/channel/socket/writer/character_data_test.go`
Function: `TestBuildCharacterData_MonsterBook`

The test constructs a monster book with `CoverCardId: item.Id(2380001)` (a card id, not a mob id) and asserts:

```go
if cd.MonsterBook.CoverCardId != item.Id(2380001) {
    t.Errorf("cover = %d, want 2380001", cd.MonsterBook.CoverCardId)
}
```

Value `2380001` is in item-id space (monster card items are in the 2380xxx range), confirming the login-draw cover is the card id, unchanged.

**Result: PASS** (verified in this session).

## IDA Check

IDA-MCP tools were unavailable this session (all method calls returned "Method not found"). The IDB could not be loaded for verification.

Conclusion: **IDA unavailable — not verified this session.** The no-change decision rests on behavioral evidence only (crash exclusively on Character Info, never at login/map-entry).

## Contrast: Character-Info Path

The Character Info handler (`CWvsContext::OnCharacterInfo` → `sub_684798`) was confirmed by prior IDA analysis (task-082 design phase) to call `CMobTemplate::GetMobTemplate(cover)` on the cover field. This is why Task 7 patched the Character-Info packet writer to send the mob id instead of the card id. The login-draw path does not share this code path.
