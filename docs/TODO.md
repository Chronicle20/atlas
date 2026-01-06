# Atlas Project TODO

This document tracks planned features and improvements for the Atlas MapleStory server project.

---

## Services

### Channel Service
- [ ] Cash Item Usage should verify inventory contains item being used.
- [ ] Timing issue with loading pre-existing chalkboards.
- [ ] Timing issue with loading pre-existing chairs.
- [ ] Parties. Party Portals missing. Party member map, level, job, and name changes need to be considered.


### Invite Service
- [ ] Character deletion should remove pending invites.
- [ ] Invites should be able to be queued.


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
