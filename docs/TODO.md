# Atlas Project TODO

This document tracks planned features and improvements for the Atlas MapleStory server project.

---

## Services

### Quest Service
- [ ] Design and implement a dedicated Quest Service
- [ ] Quest state management (NOT_STARTED, STARTED, COMPLETED)
- [ ] Quest progress tracking with step-based progress
- [ ] Quest rewards distribution via saga orchestrator

### Query Aggregator - Quest Support
- [ ] Update query-aggregator to properly support quest status lookups
- [ ] Implement quest data provider for ValidationContext
- [ ] `questStatus` condition currently returns "quest not found" without a quest service backend
- [ ] `questProgress` condition needs quest service integration for step-based progress checks

### Instance Based Transports
- [ ] Extend atlas-transports to support instance-based transport events
- [ ] Instance capacity management (e.g., "wagon is already full")
- [ ] On-demand warping (vs scheduled boarding windows)
- [ ] Use case: Kerning Square Train (NPC 1052007 selection 0)
  - Currently warps directly without capacity check
  - Original script used `em.startInstance(cm.getPlayer())` pattern

---

## NPC Conversations

### Pending Conversions
- NPCs requiring instance-based transports should be revisited after that feature is implemented

---

## Notes

- Instance-based transports differ from scheduled transports:
  - **Scheduled transports**: Have boarding windows, departure times, and use `transportAvailable` condition (e.g., ships, subway to NLC)
  - **Instance-based transports**: Available on-demand with capacity limits, no fixed schedule (e.g., Kerning Square Train)
