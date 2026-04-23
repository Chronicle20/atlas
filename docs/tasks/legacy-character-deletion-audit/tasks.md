# Character Deletion Audit - Task Checklist

**Last Updated: 2026-02-03**

## Phase 1: Critical Database Cleanup (High Priority)

### 1.1 atlas-quest Character Deletion Handler
- [x] Add `handleStatusEventDeleted` handler to `kafka/consumer/character/consumer.go`
- [x] Call existing `deleteByCharacterIdWithProgress()` on DELETED event (via `DeleteByCharacterId`)
- [x] Ensure quest status and progress records are deleted within transaction
- [ ] Add event emission for deleted quest records
- [ ] Add unit tests for deletion handler
- [ ] Add integration test verifying no orphaned records

**Acceptance Criteria:**
- Character deletion removes all quest_statuses and quest_progress records
- Deletion events emitted for audit trail
- Tests pass

### 1.2 atlas-guilds Character Deletion Handler
- [x] Create character consumer directory if not exists: `kafka/consumer/character/` (already existed)
- [x] Create `consumer.go` with status event handling (already existed, added handler)
- [x] Add `handleStatusEventDeleted` handler
- [x] Lookup character's guild via `GetByMemberId(characterId)`
- [x] If in a guild, call existing `LeaveAndEmit(guildId, characterId, true, transactionId)`
  - This already handles member removal AND emits `MEMBER_LEFT` with `Force: true`
- [x] Handle "not in guild" case gracefully (character may not be in a guild)
- [x] Register consumer in service initialization (already registered)
- [ ] Add unit tests for deletion handler
- [ ] Add integration test verifying guild cleanup

**Acceptance Criteria:**
- Character deletion removes guild membership record
- Other online guild members receive `MEMBER_LEFT` event (their clients update roster)
- Tests pass

### 1.3 atlas-families Character Deletion Handler
- [x] Create character consumer directory if not exists: `kafka/consumer/character/`
- [x] Create `consumer.go` with status event handling
- [x] Add `handleStatusEventDeleted` handler
- [x] Call existing `RemoveMemberAndEmit(transactionId, characterId, "CHARACTER_DELETED")` which already:
  - Removes character from senior's `junior_ids` list (if has senior)
  - Clears `senior_id` for all juniors (if has juniors)
  - Deletes member record
  - **Emits `LinkBrokenEvent` to notify affected family members**
- [x] Handle `ErrMemberNotFound` gracefully (character may not be in a family)
- [x] Register consumer in service initialization
- [ ] Add unit tests for deletion handler
- [ ] Add integration tests

**Acceptance Criteria:**
- Character deletion properly cascades through family relationships
- Affected family members (senior/juniors) receive `LinkBrokenEvent` notifications
- No orphaned references in `senior_id` or `junior_ids`
- Tests pass

### 1.4 atlas-fame Character Deletion Handler
- [x] Audit current character consumer implementation (no character status consumer existed)
- [x] Verify if DELETED handling exists (it did not)
- [x] If missing, add `handleStatusEventDeleted` handler
- [x] Delete fame records for deleted character (added `DeleteByCharacterId` to processor)
- [ ] Emit fame deletion events
- [ ] Add tests

**Acceptance Criteria:**
- Character deletion removes all fame records
- Tests pass

---

## Phase 2: Verification and Documentation

**Note:** atlas-rates and atlas-messengers were originally considered but are NOT needed because:
- A character must be logged out before it can be deleted
- Both services already handle LOGOUT events to clean up in-memory state
- By the time DELETED arrives, the in-memory entries are already gone

### 2.1 Verification
- [ ] Run all service tests after implementation
- [ ] Manual end-to-end test of character deletion flow
- [ ] Verify no orphaned records in all databases

### 2.2 Documentation
- [ ] Update service READMEs to document character deletion handling
- [ ] Add to service creation checklist: "Must handle character DELETED events if storing character data"
- [ ] Document the established pattern for future reference

---

## Summary

| Task | Service | Priority | Status |
|------|---------|----------|--------|
| 1.1 | atlas-quest | High | [x] |
| 1.2 | atlas-guilds | High | [x] |
| 1.3 | atlas-families | High | [x] |
| 1.4 | atlas-fame | High | [x] |
| 2.1 | Verification | Medium | [ ] |
| 2.2 | Documentation | Low | [ ] |
