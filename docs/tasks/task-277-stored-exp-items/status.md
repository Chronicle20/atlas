# task-277 status — implementation complete, verified, awaiting code review

**As of commit `24f1b7b54` on branch `task-277-stored-exp-items`. Tree clean.**

## Where this stands

All 14 plan tasks are implemented, reviewed and committed. The flagless
`tools/verify.sh` completed green (`EXIT=0`, "All checks passed") at HEAD
`648b103ca` — 91 modules under `go build/vet/test -race` plus every guard.
Per CLAUDE.md that is the run that counts; the three earlier flagless attempts
do not (one caught a real defect, two died on a full `/tmp` tmpfs). All six
packet-audit gates pass.

The branch has NOT been code-reviewed as a whole and NO PR exists.

## The one real defect the gate caught

Task 12 added 16 seed-template bindings (2 handlers x 8 in-scope templates) and
never bumped `services/atlas-configurations/atlas.com/configurations/socket/corpus_test.go`,
whose guard asserts the total corpus size. Fixed in `7b197cd46` (3387 -> 3403).
It passed a literal-by-literal opcode check and an APPROVED per-task review
first: the guard lives in a different package from the edited JSON, so a
diff-scoped review cannot reach it, and every earlier gate was `--quick`, which
does not run that test. **Generalization worth keeping: a task that adds N
entries to a corpus an invariant guard counts over must name that guard in its
own file inventory.**

## Two traps for the next session

1. `plan.md` Task 14 Step 2 prescribes `packet-audit <cmd> --check` with a
   DOUBLE dash. That form exits 3 with "missing required flags
   --csv-clientbound, ...", which reads like a gate failure but is an
   invocation error. Correct form is a SINGLE dash `-check`, and
   `doc-freshness` takes no check flag at all.
2. A flagless `verify.sh` (`-race` x 91 modules) needs several GB of `/tmp`
   scratch on a 16G tmpfs that accumulates per-task gocache dirs. Check
   `df -h /tmp` before launching it. An ENOSPC there masquerades as a lint or
   typecheck failure in whichever service happened to be compiling.

## Next unit

Pre-PR code review — a separate gate a green `verify.sh` cannot substitute for.
Run `superpowers:requesting-code-review`: `plan-adherence-reviewer` (14 tasks,
small enough to run unsharded) and `backend-guidelines-reviewer`. NO frontend
reviewer — no TypeScript changed on this branch. The change crosses
atlas-channel / atlas-character / atlas-configurations / atlas-consumables /
atlas-data plus `libs/atlas-constants` and `libs/atlas-packet`, so trace each
event into its consumers by hand per CLAUDE.md.

Full per-task history, rulings and gate forensics: `.superpowers/sdd/plan/progress.md`.
