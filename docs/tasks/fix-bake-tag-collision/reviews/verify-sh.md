# Review: fix-bake-tag-collision — `tools/verify.sh`

## Scope

`git diff` against `main` (335ad1a7b) touches exactly one file,
`tools/verify.sh` (+19/-1, one hunk: a new comment block plus
`BAKE_OUTPUT='*.output=type=cacheonly'` at `tools/verify.sh:282`, and
`docker buildx bake --set "$BAKE_OUTPUT" "$t"` at `tools/verify.sh:334`,
replacing the old `docker buildx bake "$t"`). `git status --porcelain` at
review time shows only `M tools/verify.sh` — no other tracked or untracked
changes in the worktree. Scope matches the brief.

Read for contract dependency (not part of the diff, but the change's
correctness rests on their semantics): `docker-bake.hcl`, the root
`Dockerfile`, `deploy/compose/*.yml`, `docs/verification.md`.

## 1. Shell correctness

- `BAKE_OUTPUT='*.output=type=cacheonly'` (`tools/verify.sh:282`) is a
  top-level script variable, defined once, before `bake_targets()` is
  declared and before the `if [ "$NO_DOCKER" -eq 1 ]` block. It is in scope
  at the one and only use site, `tools/verify.sh:334`, which sits inside the
  `else` branch of that same `if`, inside the `for t in "${TARGETS[@]}"`
  loop. That loop runs identically whether `TARGETS` was populated by the
  `--all` branch (`TARGETS=(all-go-services)`, `tools/verify.sh:313`) or by
  the resolved-`bake_targets` branch — same loop, same `step` call, so both
  paths pick it up. PASS.
- The `BAKE_RESOLVE_FAILED` path does not reach the bake loop at all (the
  `elif`/`else` at `tools/verify.sh:328-336` short-circuits to `:` and never
  populates or iterates `TARGETS`), so `BAKE_OUTPUT` is simply unused on
  that path — correct, since no bake runs there. PASS.
- Quoting: the assignment is single-quoted (`'*.output=type=cacheonly'`,
  `tools/verify.sh:282`) and the use is `"$BAKE_OUTPUT"`
  (`tools/verify.sh:334`) — double-quoted variable expansion. Bash only
  glob-expands an *unquoted* word containing an unescaped `*`; a quoted
  `"$var"` expansion is never subject to pathname expansion regardless of
  the variable's contents. `set -euo pipefail` (`tools/verify.sh:21`) is the
  only shell-option line in the file — no `set -f`/`noglob` toggle either
  way, and none is needed here since quoting already forecloses the glob.
  Confirmed by reading the file directly (not assumed): the `--set` argument
  is one shell word, `*.output=type=cacheonly`, passed intact to
  `docker buildx bake`, which does its own wildcard target-matching
  internally (a `docker buildx bake` feature, not a shell glob) — this
  matches the evidence log's "wildcard form... exit 0, tag NOT created."
  PASS.

## 2. Does cacheonly weaken the guard the negative controls didn't cover?

This is the highest-risk question and the one worth digging into rather
than trusting the comment.

- `docker-bake.hcl`'s `target "go-service"` (`docker-bake.hcl:104-119`) sets
  no `target =` (build-target-stage) field, so bake builds to the
  Dockerfile's *last* stage by default.
- The root `Dockerfile` has exactly two `FROM` stages: `build-env`
  (`golang:...` — compiles the Go binary) and the final, unnamed
  `alpine:3.24` stage, which does
  `COPY --from=build-env /server /`,
  `COPY --from=build-env /app/config.yaml /`, and six more
  `COPY --from=build-env` lines for the per-service data-dir stash
  (`Dockerfile:143-152`).
- Because the *target* stage is the final `alpine` stage, and that stage's
  own instructions (`COPY --from=build-env ...`) are part of the solve
  graph BuildKit must execute to produce that stage's output, `type=cacheonly`
  does not change *what gets solved* — it only changes what happens to the
  solved result afterward (write it to the image store vs. discard it). A
  missing `COPY libs/...` line upstream in `build-env` would still fail
  `go build -C "$MOD_DIR" -o /server` (`Dockerfile:107-109`) regardless of
  exporter, and a broken `COPY --from=build-env` in the final stage itself
  would still fail to resolve during the solve, before any export step runs.
- This reasoning is independently corroborated by the evidence log:
  Negative control A ("bad COPY path... under cacheonly -> exit 1") is
  exactly a broken-COPY scenario, and it still fails under `cacheonly`. That
  is direct evidence against the "cacheonly might lazily skip the final
  COPY layer" hypothesis, not just an inference from reading the Dockerfile.
  Negative control B (a real Go compile error) also still fails, confirming
  the `build-env` stage's `RUN go build` line is not skipped either.
- Conclusion: `type=cacheonly` does not weaken the guard relative to the
  prior "bake with default (docker) exporter" behavior. Every instruction in
  both stages is still part of the BuildKit solve graph under the default
  (unnamed) target; only the final `type=image`/`type=docker` load into the
  local image store is dropped. PASS — verified against the actual
  Dockerfile stage structure, not asserted from the comment alone.

## 3. Anything else depend on verify.sh leaving `<svc>:local` images behind?

- `grep -rn "ATLAS_IMAGE_TAG\|:local" tools/debug-start.sh tools/db-bootstrap.sh tools/build-services.sh`
  returns nothing — none of these reference the tag or consume a
  verify.sh-produced image.
- `deploy/compose/*.yml` reference `${ATLAS_IMAGE_TAG:-local}` (e.g.
  `deploy/compose/docker-compose.core.yml:30`,
  `deploy/compose/docker-compose.socket.yml:23`), which is exactly the
  collision this task fixes — compose is a *consumer* of the `:local` tag,
  and the stated intent is that `tools/build-services.sh` (unchanged, per
  the brief and confirmed by its absence from the diff) remains the sole
  producer for that consumption path. verify.sh's bake was never meant to
  feed compose; nothing in `deploy/` calls `docker buildx bake` directly to
  confirm that boundary is otherwise unchanged.
- `.github/workflows/pr-validation.yml` is the only workflow invoking bake;
  it is untouched by this diff and, per `docker-bake.hcl:18` and
  `docs/verification.md:114-124`, CI already overrides `<target>.tags=` per
  shard rather than relying on the `local` default — a separate mechanism
  from verify.sh's local run, unaffected by this change.
- PASS — nothing else in the repo depends on verify.sh's bake output landing
  in the image store.

## 4. Accuracy of the new comment block (`tools/verify.sh:262-281`)

Checked claim by claim against the diff and the evidence log:

- "This bake is a BUILD CHECK, not an image build: nothing downstream in
  this script consumes its output." — True of the diff itself: `bake_out`
  is used only to select targets (`tools/verify.sh:319-326`), and no later
  step in the script references an image. Matches the stated brief.
- "output=cacheonly... never writes to the docker image store." — Matches
  the evidence log's cacheonly probes (exit 0, tag not created) and the
  end-to-end run's absence of "exporting to image"/"naming to" log lines
  plus the unchanged image ID before/after.
- "docker-bake.hcl tags targets `<svc>:${ATLAS_IMAGE_TAG}` (default `local`)"
  — matches `docker-bake.hcl:20-23` (`variable "ATLAS_IMAGE_TAG" { default = "local" }`)
  and `docker-bake.hcl:118` (`tags = ["${svc}:${ATLAS_IMAGE_TAG}"]`).
- "the same tag deploy/compose/docker-compose.*.yml runs" — matches
  `deploy/compose/docker-compose.core.yml:30` and
  `deploy/compose/docker-compose.socket.yml:23` (`${ATLAS_IMAGE_TAG:-local}`).
- "A broken build still FAILS under cacheonly (every stage runs; only the
  export is dropped) — verified against both a bad COPY path and a Go type
  error." — matches negative controls A and B in the evidence log, and is
  independently supported by the Dockerfile stage analysis in §2 above
  (single unnamed final target, both stages in the solve graph regardless
  of exporter).
- "the shared buildkit cache still makes a later real build fast." — this
  is the one claim in the comment not directly exercised by the evidence
  log (no before/after timing of a `tools/build-services.sh` run was
  captured). It is consistent with standard BuildKit behavior — the solve
  cache is populated independently of the exporter, so a `cacheonly` bake
  does warm the same local cache a subsequent `type=docker` build would hit
  — but it is an inference, not a measurement. Flagged as non-blocking:
  the claim is plausible and does not affect the fix's correctness, but it
  is technically unverified by the evidence bundle handed to this review.
- "To actually PRODUCE runnable `<svc>:local` images, use
  tools/build-services.sh." — consistent with the brief ("`tools/build-services.sh`
  is the intended image-producing path and is unchanged") and with
  `docs/verification.md:124` ("`docker buildx bake all-go-services` (or
  `tools/build-services.sh`, a thin wrapper)").

No invented values, opcodes, or unverified numeric claims found in the
comment. One soft, plausible-but-unmeasured claim noted above
(non-blocking).

## 5. Is `docs/verification.md` / `CLAUDE.md` now stale?

- `docs/verification.md`'s "## The docker layer" section
  (`docs/verification.md:112-132`) describes *why* the bake step exists and
  *that* it is mandatory; it never claims the bake step leaves behind or is
  meant to leave behind a usable `<svc>:local` image, so nothing there is
  factually contradicted by this change. It is silent on output/export
  behavior entirely.
- That silence is itself a minor gap this change creates: a reader of
  `docs/verification.md` who knows the old behavior (bake loads into the
  image store) has no signal that verify.sh's bake no longer does that, and
  might reach for a `verify.sh`-built image expecting it to exist (as the
  bug report shows someone implicitly did, worktree-to-worktree). A
  one-line addition — e.g. "verify.sh's bake runs with
  `output=cacheonly`; it does not produce a runnable `<svc>:local` image,
  use `tools/build-services.sh` for that" — would close this gap. This is
  non-blocking (the code change is correct and self-documented via the
  script comment), but the fix as submitted does not touch
  `docs/verification.md`, and the brief's checklist item 5 explicitly asked
  whether it should be updated in the same change. Recommend adding it
  before merge, or in the same PR, since it is a two-line, zero-risk
  addition to a doc this exact diff's topic sentence already lives in.
- `CLAUDE.md` does not mention the bake step's output/export behavior at
  all (only `docker-bake.hcl` and `docs/verification.md` do); nothing there
  is stale.

## Findings summary

No blocking defects found. Shell scoping/quoting is correct, the guard's
strength is unchanged (verified against the Dockerfile stage graph, not
just trusted from the comment), and no other code path depends on
verify.sh's bake output. One non-blocking documentation gap
(`docs/verification.md` doesn't mention the export-behavior change) and one
non-blocking soft claim in the new comment (cache-reuse assertion is
plausible but unmeasured).
