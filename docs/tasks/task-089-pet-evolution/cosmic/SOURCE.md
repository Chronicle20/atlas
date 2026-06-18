# Cosmic quest-script provenance

The Garnox pet-evolution quest conversations (8185/8189/4659) were authored from the
upstream Cosmic quest scripts:

- Source repo: `Cosmic` (local: `~/source/Cosmic`), `scripts/quest/{8185,8189,4659}.js`
  (Author "Blue"/"Moogra", "Garnox", New Leaf City : Town Center).

Faithfulness notes (deviations approved by the user):
- Dialogue text is verbatim from the scripts.
- Quest 8184 (Pet's Evolution1) is a plain item turn-in — no script, no conversation.
- No-op `startStateMachine` (the scripts have no `start()`).
- 8185 leaves the quest STARTED (script never calls `completeQuest()`); 8189/4659 `complete_quest`.
- atlas's `evolve_pet` rolls the adult outcome internally from atlas-data WZ `evolProb`
  (the scripts hand-roll it); the conversation can't know the result, so the success line's
  dynamic "now it's a #i<after>#" clause was replaced with a generic phrase.
- Pet targeting: `enumerate_evolvable_pets` exposes the first eligible summoned pet id
  (`firstEvolvablePet`), matching the scripts' "first matching summoned pet" loop.
- 4659 consumes only the Rock (5380000); the 50× Cheap Battery (4000111) is a quest
  requirement (enforced by atlas-quest) but the script does not destroy it.
