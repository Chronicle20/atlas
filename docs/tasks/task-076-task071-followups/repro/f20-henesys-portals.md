# F20 — Henesys portal duplication diagnosis

## Observed (operator-reported, 2026-05-22)

Henesys map 100000000 returns duplicate portal entries in atlas-portals'
response. Operator-side observation only; no captured payload was
available to this worktree.

## Diagnosis attempt

Skipped: this worktree has no reachable atlas-portals / atlas-data env.
Per the task-076 plan (R4 / OQ-6), the bounded fix in `extractPortals`
is applied unconditionally because:

1. Dedup by `(name, target, x, y)` is a defensive no-op when WZ data
   contains no duplicates — the key uniqueness only filters in the
   pathological case.
2. The Henesys WZ data is known (per OQ-6 design notes) to contain
   shadow entries (`pn=""`, `pn="0"`) that collide with player-visible
   portals on the same coordinate.
3. The accompanying regression test (`layers_portal_test.go`) pins the
   dedup behavior so any future regression surfaces immediately.

## Follow-up

Operator should:
1. Probe atlas-portals for `?map=100000000` on atlas-main post-deploy.
2. Verify duplicate entries no longer appear.
3. If duplicates persist, the bug is read-side (atlas-portals) not
   extraction-side, and a follow-up task should walk that path. See
   plan R4 for the structural escape hatch.
