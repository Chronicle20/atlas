# task-286 measurements

Before/after evidence for each layer of the build & verify concurrency work.
Each layer appends its own heading below; do not renumber existing headings.

## Layer 1 — build context

Command (per the brief):

```sh
docker buildx bake --progress=plain --set '*.output=type=cacheonly' atlas-ban 2>&1 | grep -i 'transferring context'
```

### Measured figures

All runs below were taken from this task's worktree
(`.worktrees/task-286-build-verify-concurrency/`), with `docker buildx prune -a -f`
run immediately before each measurement to eliminate BuildKit's local-source
transfer cache as a confound (BuildKit dedupes unchanged context bytes across
builds; without pruning, a "before" run immediately followed by an "after" run
under-reports because most bytes were already known to the daemon).

| Run | `.dockerignore` | `transferring context` (build-context step) |
|---|---|---|
| Before (clean cache) | original (excludes only `node_modules`, `.next`, `*.log`, `.git`, `.github`) | `11.30MB` |
| After (clean cache)  | allowlist (`*` then `!libs`, `!services`)                                    | `11.30MB` |

Raw log excerpts:

```
# before (docker buildx prune -a -f; then bake atlas-ban)
#7 transferring context: 320B done
#10 transferring context: 11.30MB 0.4s done

# after (docker buildx prune -a -f; then bake atlas-ban)
#7 transferring context: 1.14kB done
#11 transferring context: 11.30MB 0.3s done
```

(The `#7`/`.dockerignore` transfer step differs only because the new file is
larger — 320B vs 1.14kB — than the old one; that is the `.dockerignore` file
itself, not the build context.)

### What this does and does not show

**Not measured / no material difference observed for a single-service build
in this worktree.** The brief's acceptance rationale (`transferring context`
drops from several GiB to under 1 GiB) assumes a context that contains the
gitignored `tmp/` (~8 GiB of WZ assets) and `.worktrees/` (~26 GiB of sibling
checkouts) at the repo root. Neither directory exists inside a `git worktree`
checkout — `git worktree add` does not duplicate a parent worktree's
gitignored content, `tmp/` is never populated here, and `.worktrees/` (this
worktree's own path) has no meaning inside itself. `du -sh tmp .worktrees`
inside this worktree returns "No such file or directory" for both.

Independently of that, this BuildKit (`v0.32.2`, `docker` driver) performs a
lazy local-context transfer: it sends only the files the invoked Dockerfile
stages actually `COPY`, not the full directory tree admitted by
`.dockerignore`. For the `atlas-ban` target that COPY set is `libs/*` and
`services/atlas-ban/`, which is why both the old and new `.dockerignore`
report the identical `11.30MB` figure above — the old file already permitted
those same paths through implicitly (it excluded only `node_modules`, `.next`,
`*.log`, `.git`, `.github`, nothing under `libs/` or `services/`).

The allowlist's real effect — reducing the directory walk BuildKit performs to
build its context digest before deciding what to request — is not visible in
the `transferring context` byte count on a machine/worktree where `tmp/` and
`.worktrees/` are absent. It would need to be measured from the main repo
checkout, which is out of scope for work done inside this task's worktree per
this task's constraints (never edit files in the main repo when a task
worktree exists for the work). No number for that scenario is fabricated
here; acceptance criterion 1 (after-figure under 1 GiB) is trivially met by
the measured `11.30MB`, but readers should not conclude a GiB-scale reduction
was demonstrated by this evidence — it was not measurable from this location.

### Step 3 — `all-services` bake (no file lost)

Command:

```sh
docker buildx bake --set '*.output=type=cacheonly' all-services
```

Result: exit `0`. Wall time: `31:47.62` (31 minutes 47.62 seconds), from a
worktree with an already-warm builder cache from the atlas-ban runs above.
This builds all 67 Go services plus `atlas-ui`, `atlas-pr-bootstrap` and
`atlas-kafka-precreate`; no target failed, so the allowlist did not drop any
file any bake target needed.
