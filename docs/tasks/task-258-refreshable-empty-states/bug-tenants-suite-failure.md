# Bug: nondeterministic `atlas-ui` full-suite failure in `TenantsPage.test.tsx`

## Symptom

`tools/verify.sh` (flagless) failed at the `atlas-ui tests + build` gate. Only the
full-suite run failed; `TenantsPage.test.tsx` run alone always passes (12/12).

Two runs of the gate (`.superpowers/sdd/plan/gates/verify-final.log`,
`verify-final-2.log`) showed different failure sets:

- Run 1 (loaded machine): 4 tests failed, across `TenantsPage.test.tsx` **and**
  `src/components/features/reward-pools/__tests__/PoolItemDialog.test.tsx`.
- Run 2 (clean machine): 2 tests failed, both in `TenantsPage.test.tsx`:
  1. `rejects name longer than 100 chars with inline error` →
     `Error: Test timed out in 5000ms.`
  2. `submits valid new name, closes dialog, shows success toast, calls refreshTenants`
     → `updateTenant` called with `{ name: "xxAcme Renamed" }` instead of
     `{ name: "Acme Renamed" }`.

## Investigation

`PoolItemDialog.tsx` and its test file are **not touched by this branch at all**
(`git diff 1461bfc96 fbb9736f7 -- .../PoolItemDialog.tsx .../PoolItemDialog.test.tsx`
is empty), yet run 1 shows it failing the same way (a `Test timed out in 5000ms`
under `user.type(...)`). This rules out anything specific to
`TenantsPage.tsx`'s new refresh wiring (the `tenantsSource` object literal,
the `if (loading && !isRefreshingTenants)` guard, `useGridRefresh`) as the
mechanism — `useGridRefresh` was independently confirmed to be a pure function
(no `useEffect`/`useMemo`/state), so it cannot be driving a re-render loop.

Locally reproduced the same class of failure with `npx vitest run` (full
suite) on this branch, without the fix:

```
❯ src/components/features/npc/__tests__/NpcShopCommodityDialog.test.tsx (4 tests | 1 failed) 7677ms
     × create mode submits the exact CommodityAttributes shape a picker-chosen id produces 5239ms
❯ src/pages/__tests__/TenantsPage.test.tsx (12 tests | 2 failed) 25555ms
     × rejects name longer than 100 chars with inline error 5299ms
     × submits valid new name, closes dialog, shows success toast, calls refreshTenants 5004ms
 Test Files  2 failed | 257 passed (259)
      Tests  3 failed | 2128 passed (2131)
```

A *third*, still-different, unmodified file (`NpcShopCommodityDialog.test.tsx`)
timed out the same way. Three different failure sets across three runs, all
sharing the same shape (`Test timed out in 5000ms` inside a `user.type(...)`
call), and hitting files this branch never touched, is strong evidence the
mechanism is **general CPU contention across the full parallel test run**,
not a defect specific to `TenantsPage.tsx`'s production code.

### Root cause

`userEvent.setup()` (the `@testing-library/user-event` v14.6.1 default) uses
`delay: 0` (a *number*, not `null`). Per
`node_modules/@testing-library/user-event/dist/cjs/utils/misc/wait.js`, any
numeric `delay` — including `0` — goes through a real
`globalThis.setTimeout(resolve, delay)` **once per keystroke**:

```js
const delay = config.delay;
if (typeof delay !== 'number') { ... }
Promise.race([
  new Promise((resolve) => globalThis.setTimeout(() => resolve(), delay)),
  config.advanceTimers(delay),
])
```

`rejects name longer than 100 chars with inline error` types 101 characters,
i.e. 101 real `setTimeout(…, 0)` round-trips through the event loop, each of
which re-renders `TenantsPage` (the form is `mode: "onChange"`, and
`useWatch` forces a re-render on every keystroke) and re-runs the Zod
validator. Under normal load a `setTimeout(0)` resolves in a fraction of a
millisecond; under the CPU contention produced by Vitest's default
multi-worker full-suite run (259 files, several hundred concurrent React
renders across worker threads), each of those 101 timer round-trips can cost
tens of milliseconds, and the cumulative wall-clock time occasionally exceeds
the file's 5000ms `testTimeout`. This is a pre-existing latency budget that
was already marginal at the merge base (the 101-char test is unchanged from
`1461bfc96`) — this branch's 35 newly added tests add enough additional
parallel work across the suite to tip an already-marginal test over the edge
more often. `PoolItemDialog.test.tsx` and `NpcShopCommodityDialog.test.tsx`
have their own multi-keystroke `user.type()` calls and are marginal for the
exact same reason; they are not modified by this branch, just squeezed by the
same shared CPU budget.

### Failure 2 is a cascade of failure 1 — confirmed

When `rejects name longer than 100 chars` times out, Vitest stops *waiting*
for the test but does not cancel the in-flight `user.type()` promise chain —
it is still walking through its remaining real `setTimeout` calls, each of
which dispatches a keydown/keyup/input event at whatever element is
`document.activeElement` at the time the timer fires. The test framework
moves on to the next test in the file, which calls `renderPage()` again and
focuses a *new* rename dialog's `Name` input. Two of the still-pending
keystrokes from the abandoned `"x".repeat(101)` typing land on that new
input before the next test's own `user.clear()` + `user.type(input, "Acme
Renamed")` runs, producing the observed `"xxAcme Renamed"` — exactly two
stray `"x"` characters, consistent with two timers still in flight when the
next test's render committed. This confirms failure 2 is not an independent
bug; it disappears once failure 1's underlying timeout is fixed.

## Fix

Pass `{ delay: null }` to every `userEvent.setup()` call in
`TenantsPage.test.tsx`. Per
`node_modules/@testing-library/user-event/dist/types/options.d.ts`
(`delay?: number | null`), `null` is the documented "no artificial per-key
delay" mode: `wait.js`'s `typeof delay !== 'number'` branch short-circuits
and the interaction proceeds through microtasks instead of scheduling a real
timer for each keystroke. This removes the 101 real-timer round-trips that
were the timing-marginal cost, and removes the possibility of stray timers
surviving into the next test — both symptoms are eliminated by the same
one-line change, which is further evidence the diagnosis is correct rather
than coincidental.

This is **not** a timeout increase: `testTimeout` is untouched. It removes
the actual per-keystroke real-timer cost that was consuming the budget, and
no behavior asserted by any test changes (`user.type`/`user.click` semantics
are otherwise identical).

### Files changed

- `services/atlas-ui/src/pages/__tests__/TenantsPage.test.tsx` — all 5
  `userEvent.setup()` call sites changed to `userEvent.setup({ delay: null })`.

## Verification

```
cd services/atlas-ui && npx vitest run src/pages/__tests__/TenantsPage.test.tsx
```
```
 Test Files  1 passed (1)
      Tests  12 passed (12)
```

Full suite, run twice from a otherwise-idle shell (per the task's
verification bar):

Run 1:
```
 Test Files  259 passed (259)
      Tests  2131 passed (2131)
   Duration  41.96s
```

Run 2:
```
 Test Files  259 passed (259)
      Tests  2131 passed (2131)
   Duration  39.93s
```

`npx tsc -b` also passes with no output (no type errors introduced).

## Scope note

`PoolItemDialog.test.tsx` and `NpcShopCommodityDialog.test.tsx` were not
touched — they are unmodified by this branch and their occasional
susceptibility to the same shared-CPU-budget timing issue is a pre-existing,
repo-wide characteristic of tests that `user.type()` long strings with the
user-event default `delay: 0`, not something task-258 introduced. Both full
suite runs above passed cleanly at branch HEAD with only the
`TenantsPage.test.tsx` fix applied, so no further change was needed to clear
the gate. If this class of flake resurfaces in unrelated files under CI load,
the same `{ delay: null }` treatment is the applicable fix there too, but
that is outside this branch's scope.

## Round 3: `delay: null` alone was insufficient — remaining timeout root cause

Round 2's `{ delay: null }` fix was correct and necessary (it eliminated the
cascade failure and removed all real per-keystroke `setTimeout`s), but the
branch-end gate still failed once more, per
`.superpowers/sdd/plan/gates/verify-final-3.log`:

```
     × rejects name longer than 100 chars with inline error 5030ms
 FAIL  src/pages/__tests__/TenantsPage.test.tsx > TenantsPage rename flow > rejects name longer than 100 chars with inline error
Error: Test timed out in 5000ms.
 Test Files  1 failed | 258 passed (259)
      Tests  1 failed | 2130 passed (2131)
```

The cascade (`"xxAcme Renamed"`) is confirmed gone in this log — only the
timeout remains, exactly as round 2 predicted would be the residual risk.

### Root cause (confirmed)

`delay: null` removes the real-timer cost per keystroke, but
`user.type(input, "x".repeat(101))` still dispatches 101 discrete
keydown/keypress/input/keyup event sequences through `user-event`'s internal
microtask queue. Each of the 101 `input` events fires react-hook-form's
`mode: "onChange"` validation (the field is registered under
`tenantNameSchema`, `.max(100, ...)`), which re-runs the Zod validator and
triggers a React re-render of the rename dialog stacked on top of the
already-rendered tenants table. Under the CPU contention produced by
Vitest's default multi-worker full-suite run (259 files, hundreds of
concurrent renders across worker threads), 101 synchronous
validate-then-render cycles is enough wall-clock cost on its own — even with
zero artificial timer delay — to occasionally brush the file's 5000ms
`testTimeout`. `delay: null` fixed the timer-multiplication half of the
mechanism; the render-multiplication half (101 renders vs. 1) was the
remaining, still-marginal cost.

### Fix

Replaced the 101-keystroke `user.type()` call with a single
`user.click(input)` (to focus/select, mirroring the other tests in the file)
followed by one `user.paste("x".repeat(101))` call. `user-event`'s `paste()`
API dispatches a single `paste` event plus a single synthetic `input` event
carrying the full pasted value — exactly one react-hook-form `onChange`
validation cycle and one re-render, instead of 101. The other `user.type()`
calls in this file (`"   Acme   "`, `"   "`, `"Acme Renamed"`,
`"Something New"`) are all ≤13 characters and were left untouched — they are
not a meaningful render-count cost and are not part of the observed
timeout/cascade pattern.

Test-honesty check performed: temporarily changed
`tenantNameSchema`'s `.max(100, ...)` to `.max(200, ...)` in
`services/atlas-ui/src/lib/schemas/tenant.schema.ts` and reran the single
test in isolation — it failed as expected (`within(dialog).getByText(/100
characters or less/i)` never appeared, because a 101-char paste no longer
tripped the loosened schema), confirming the test still asserts the real
validation path rather than passing for an unrelated reason. The schema
change was then reverted (`git diff` on that file is empty).

### Files changed (round 3)

- `services/atlas-ui/src/pages/__tests__/TenantsPage.test.tsx` — the
  `rejects name longer than 100 chars with inline error` test now uses
  `user.click(input)` + `user.paste("x".repeat(101))` instead of
  `user.type(input, "x".repeat(101))`. Assertions unchanged.

### Verification (round 3)

Single file:
```
cd services/atlas-ui && npx vitest run src/pages/__tests__/TenantsPage.test.tsx
 Test Files  1 passed (1)
      Tests  12 passed (12)
```

Full suite, two consecutive runs (`--reporter=verbose` to capture per-test
durations):

Run 1:
```
 Test Files  259 passed (259)
      Tests  2131 passed (2131)
   Duration  43.19s
```
Previously-failing test: `rejects name longer than 100 chars with inline
error 740ms` (vs. the 5000ms budget it was previously timing out against,
and the 5030ms it hit in `verify-final-3.log`).

Run 2:
```
 Test Files  259 passed (259)
      Tests  2131 passed (2131)
   Duration  37.28s
```
Previously-failing test: `rejects name longer than 100 chars with inline
error 584ms`.

`TenantsPage.test.tsx`'s 12-test summed durations across the full-suite runs
were 7029ms (run 1) and 5998ms (run 2) — no per-file summary line is emitted
by the verbose reporter for a fast, passing file, so the sum of the file's
own 12 test durations is reported here as the closest available proxy for
the file's total wall-clock cost.

`npx tsc -b` — clean, no output.

### Scope note (round 3)

`PoolItemDialog.test.tsx` and `NpcShopCommodityDialog.test.tsx`, noted in
round 2 as sharing the same shared-CPU-budget class of flake, were not
touched — same reasoning as round 2: they are unmodified by this branch.
