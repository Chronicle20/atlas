# task-206 — Final review outcomes

Five reviewers ran against the whole branch (46 commits, 146 files, ~22k lines) after all 30
plan tasks were implemented and individually reviewed. **Merge verdict: READY.**

| Review | Verdict | Findings file |
|---|---|---|
| Whole-branch (cross-cutting) | **READY** — 0 Critical, 0 Important | this file |
| Plan adherence | 30/30 implemented | [`audit-plan-adherence.md`](audit-plan-adherence.md) |
| Packet completeness critic | Clean on the primary claim, 0 scope holes | [`completeness-critic.md`](completeness-critic.md) |
| Frontend guidelines | PASS, 0 blocking FE-* | [`audit-frontend.md`](audit-frontend.md) |
| Backend guidelines | Structural findings only, 0 correctness/security | [`audit-backend.md`](audit-backend.md) |

## The highest-risk change, and why it stands

`dispatcher-lint`'s FAM-CAP check was scoped to skip `direction: serverbound` files. A guard
narrowed to make a branch pass deserves adversarial scrutiny, so it got it. The verdict is
**correct scoping, not a weakened guard**, established empirically:

1. The 17-cell demotion is real and was **reproduced** — `matrix` run twice into scratch
   out-dirs, once with a `families.yaml` carrying the two fnames: 8010 cells, 17 changed,
   all `verified → family` (10 `COUPON_CODE` + 7 `CASHSHOP_OPERATION`, all serverbound).
   `grade.go` applies `in.Families[baseFName(ref.FName)]` with **no direction check**.
2. FAM-CAP's other remedy is **structurally unreachable** for serverbound — the discrete-struct
   resolver searches only `libs/atlas-packet/<pkg>/clientbound/`. Both dispositions the check
   offers were inapplicable: it was mis-scoped, not merely inconvenient.
3. **Clientbound still trips**, and a file that *omits* `direction:` still trips. Only an
   explicit `serverbound` opts out — the narrowing is fail-safe.
4. Precedent predates this branch: `docs/packets/evidence/families.yaml` already documents
   that serverbound `RPS_ACTION` is deliberately unlisted for the same reason.

Now recorded in [`docs/packets/DISPATCHER_FAMILY.md`](../../packets/DISPATCHER_FAMILY.md)
rather than living only in a code comment.

## Backend findings — controller adjudication

The backend audit raised structural items. Each was checked against **actual repo practice**
before being accepted:

| Finding | Ruling | Evidence |
|---|---|---|
| `Model.ToEntity()` missing (Important ×3) | **Rejected** | Only 5 files repo-wide define it — not an established convention |
| `TransformSlice` missing (Minor ×3) | **Rejected — backwards** | 4 files repo-wide define it; **44** use `model.SliceMap(Transform)`, which is what this branch does |
| `Builder` in `model.go` not `builder.go` (Important ×3) | **Deferred** | 118 files repo-wide use `builder.go`, but `atlas-cashshop` has **zero** — no precedent in the service these packages mirror. Pure file organization, no behaviour change |
| `PatchRestModel` in `patch.go` (Minor) | **Deferred** | Better cohesion with `Nullable[T]` |
| Missing `degrade.Observe` (Minor) | **Deferred** | Used in 2 files across all of atlas-channel, and in neither this consumer nor its sibling `handleStatusEventPurchase` |

## Fix wave (two items, both verified)

- **`4067ff646`** — documented the FAM-CAP serverbound scoping.
- **`7df6c8c6f`** — stopped the `UNKNOWN_ERROR` coupon path logging a phantom
  "will likely cause a client crash" on ordinary operational paths (missing locker row,
  transaction failure). Short-circuit confined to `CashShopUseCouponFailedBody`; `ResolveCode`
  and every other `*Body` untouched; **the resolved wire byte is still 99**, unchanged. Both
  callers enumerated repo-wide. The new no-ERROR-log test was falsified (removing the
  short-circuit makes it fail) and the pre-existing `reason == 99` assertion still passes.

## Known limits, stated plainly

These are accepted, not hidden. The honesty check found no place claiming more than was shown.

- **No test on this branch exercises write concurrency.** `databasetest.NewInMemoryTenantDB`
  sets `SetMaxOpenConns(1)` — mandatory, because gorm's SQLite driver gives each pooled
  connection its own empty in-memory database. The Postgres unique-index path that resolves
  the same-account race is therefore unexercised. The falsification was run and reported as a
  **disconfirmation**: the banned read-then-write `reserveUse` passes the concurrency test
  10/10.
- **Most error key *names* are ordinal alignment**, not decompiled text. Every *byte* traces
  to an address in `derivation.md` and no reserved byte is mapped, but a one-position slip
  inside an `aligned` block would show an adjacent-but-wrong notice and pass every static
  check. Only the live client test can settle it.
- **`COUPON_USAGE_LIMIT` is absent** from the `gms_v48` and `jms_v185` error tables — both
  correct, not defects. v48's switch has no such case at all; on jms it falls among the 23
  keys that need a JMS `String.wz` dump. Effect: a maxed-out coupon shows the generic notice
  on those two versions.
- **Live client end-to-end has not been run** — see the Step 7 section of
  [`verification.md`](verification.md).
