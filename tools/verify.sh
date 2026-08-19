#!/usr/bin/env bash
# tools/verify.sh — the pre-PR verification gate.
#
# One entry point for everything that must be clean before a branch is called
# "done". It mirrors the jobs in .github/workflows/pr-validation.yml so a green
# run here means a green run in CI; when the two drift, CI is the authority and
# this script is the bug.
#
# Rationale, per-guard detail and the escape hatches live in
# docs/verification.md. This script is the executable form of that document —
# do not restate its contents in CLAUDE.md.
#
# Change detection: guards whose CI job is path-gated only run when the
# relevant paths changed against the merge base. `--all` forces everything;
# `--base <rev>` narrows the change set to an increment, which is what a
# per-task iteration gate wants — see docs/verification.md, "Iteration gate".
#
# Every check runs even after an earlier one fails, so one pass gives the
# complete picture. Exit status is non-zero if any check failed.

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

BASE=""
ALL=0
NO_DOCKER=0
NO_UI=0
QUICK=0
FACTS=0

usage() {
    cat <<'EOF'
usage: tools/verify.sh [options]

  --base <rev>   diff base for change detection (default: merge-base with
                 origin/main, falling back to main). For a per-task iteration
                 gate pass the last commit you already gated — the default
                 whole-branch diff makes one libs/ commit fan every later run
                 out to all modules.
  --all          run every check regardless of what changed
  --no-docker    skip `docker buildx bake` (fast inner loop; NOT sufficient
                 before a PR when a go.mod was touched)
  --no-ui        skip the atlas-ui lint/test layer
  --quick        skip docker + `go test -race` (syntax/vet/guards only)
  --facts        print WHAT this invocation would select — change base, changed
                 services/libs, fan-out reason, module count, guard suites,
                 bake targets, gates — and exit 0 without running any check.
                 Combine with the same flags you would really run
                 (`--facts --quick --base <sha>`); the answer reflects them.
  -h, --help     this message

Exit status is 0 only when every check that ran passed.
EOF
}

while [ $# -gt 0 ]; do
    case "$1" in
        --base) BASE="$2"; shift 2 ;;
        --all) ALL=1; shift ;;
        --no-docker) NO_DOCKER=1; shift ;;
        --no-ui) NO_UI=1; shift ;;
        --quick) QUICK=1; NO_DOCKER=1; shift ;;
        --facts) FACTS=1; shift ;;
        -h|--help) usage; exit 0 ;;
        *) echo "verify.sh: unknown option $1" >&2; usage >&2; exit 2 ;;
    esac
done

PASSED=()
FAILED=()
SKIPPED=()
SELECTED=()

# Informational chatter. Under --facts it must not pollute the fact block on
# stdout, but it must not be lost either — a suppressed warning is how a fan-out
# surprise becomes a rediscovery.
info() {
    if [ "$FACTS" -eq 1 ]; then
        echo "$@" >&2
    else
        echo "$@"
    fi
}

# step <label> <command...>
#
# Under --facts this is the whole mechanism: the script's real body runs, its
# real selection logic decides which steps are reached, and each one records its
# label instead of executing. That is why --facts cannot drift from a real run —
# it IS the real run, with the work removed. Never reimplement the selection
# predicates in the fact printer.
step() {
    local label="$1"; shift
    if [ "$FACTS" -eq 1 ]; then
        SELECTED+=("$label")
        return 0
    fi
    printf '\n\033[1m── %s\033[0m\n' "$label"
    if "$@"; then
        PASSED+=("$label")
    else
        FAILED+=("$label")
        printf '\033[31m✗ %s FAILED\033[0m\n' "$label"
    fi
}

skip() {
    SKIPPED+=("$1")
}

resolve_base() {
    if [ -n "$BASE" ]; then
        echo "$BASE"
        return 0
    fi
    git merge-base HEAD origin/main 2>/dev/null && return 0
    git merge-base HEAD main 2>/dev/null && return 0
    return 1
}

# ---------------------------------------------------------------- change set

CHANGED=""
BASE_SHA="all"
if [ "$ALL" -eq 1 ]; then
    info "verify.sh: --all — running every check"
else
    if base="$(resolve_base)"; then
        # `git diff --name-only` with no rev compares worktree-vs-INDEX, so a
        # file that has been `git add`ed but not committed shows up in none of
        # these three sources — not the committed range, not the unstaged diff,
        # and not --others (staging makes it tracked). Running the gate after
        # `git add -p` would then skip that file's module, bake target and
        # guards and still exit 0. Diff against HEAD to cover staged + unstaged.
        CHANGED="$(git diff --name-only "$base"...HEAD; git diff --name-only HEAD; git ls-files --others --exclude-standard)"
        CHANGED="$(printf '%s\n' "$CHANGED" | sort -u | sed '/^$/d')"
        BASE_SHA="$(git rev-parse --short "$base")"
        info "verify.sh: change base $BASE_SHA — $(printf '%s\n' "$CHANGED" | sed '/^$/d' | wc -l) changed path(s)"
    else
        echo "verify.sh: WARNING — no merge base with origin/main or main; falling back to --all" >&2
        ALL=1
    fi
fi

touched() {
    # touched <grep-ere> — true when the change set matches, or when --all
    [ "$ALL" -eq 1 ] && return 0
    # NOT `grep -q`. Under `set -o pipefail`, grep -q exits on the first match
    # and the still-writing `printf` takes SIGPIPE (141), which pipefail then
    # reports as the pipeline's status — so a MATCH reads as "no match" and the
    # guard SKIPS while the gate still exits 0. It only bites once $CHANGED
    # exceeds the 64KB pipe buffer, i.e. on exactly the large sweep branches
    # that most need the guards. Letting grep drain its input removes the race.
    printf '%s\n' "$CHANGED" | grep -E "$1" >/dev/null
}

changed_tool_suites() {
    # Test suites to run for the changed tools/ scripts, one path per line.
    # tools/foo.sh -> tools/foo_test.sh (when it exists); a changed
    # tools/foo_test.sh runs itself.
    if [ "$ALL" -eq 1 ]; then
        find tools -name '*_test.sh' -type f | sort
        return 0
    fi
    printf '%s\n' "$CHANGED" \
        | grep -E '^tools/.*\.sh$' \
        | while IFS= read -r f; do
            case "$f" in
                *_test.sh) suite="$f" ;;
                *)         suite="${f%.sh}_test.sh" ;;
            esac
            [ -f "$suite" ] && printf '%s\n' "$suite"
        done \
        | sort -u || true
}

# --------------------------------------------------------------- go modules

all_modules() {
    find "$ROOT/services" "$ROOT/libs" -name go.mod -not -path '*/node_modules/*' -print0 \
        | xargs -0 -r -n1 dirname | sort -u
}

# The one predicate that decides whether this run fans out to every module.
# Extracted so `--facts` can name the reason without restating the rule.
fanout_paths() {
    printf '%s\n' "$CHANGED" | grep -E '^(go\.work|libs/)' || true
}

changed_modules() {
    if [ "$ALL" -eq 1 ]; then
        all_modules
        return
    fi
    # go.work, or ANY shared lib, reaches every module: services consume libs
    # through the workspace, so a lib edit can break a service that has no
    # changed file of its own. Conservative on purpose — use --quick to skip
    # the -race pass while iterating.
    #
    # The trap this fan-out sets: CHANGED is the whole branch against its merge
    # base, so ONE libs/ commit makes every later run on that branch a full
    # 86-module build, forever. On a long branch that is ~10 minutes per run
    # instead of ~1. Say so out loud, and name the remedy — an iteration gate
    # should pass --base <last-gated-commit> so the change set is the increment
    # under test, not the accumulated branch.
    local fanout
    fanout="$(fanout_paths)"
    if [ -n "$fanout" ]; then
        if [ -z "$BASE" ]; then
            printf '\033[33mverify.sh: shared-lib change fans out to ALL modules (%s path(s) under go.work/libs/).\n' \
                "$(printf '%s\n' "$fanout" | wc -l | tr -d ' ')" >&2
            printf '           first: %s\n' "$(printf '%s\n' "$fanout" | head -1)" >&2
            printf '           This is the whole-branch diff. For a per-task iteration gate pass\n' >&2
            printf '           --base <last-gated-commit> to scope it to the increment under test.\033[0m\n' >&2
        else
            # stderr, NOT stdout: this function's stdout IS the module list the
            # caller cd's into, so a chatty line here becomes a phantom module
            # ("cd: verify.sh: shared-lib change...: No such file or directory")
            # and fails the gate. Matches the no-BASE branch above.
            echo "verify.sh: shared-lib change in this increment — fanning out to all modules" >&2
        fi
        all_modules
        return
    fi
    local mods=() m rel
    while IFS= read -r m; do mods+=("$m"); done < <(all_modules)
    for m in "${mods[@]}"; do
        rel="${m#"$ROOT"/}"
        # See touched(): no `-q` under pipefail.
        if printf '%s\n' "$CHANGED" | grep -E "^${rel}/" >/dev/null; then
            echo "$m"
        fi
    done
}

MODULES=()
while IFS= read -r m; do [ -n "$m" ] && MODULES+=("$m"); done < <(changed_modules | sort -u)

go_layer() {
    local mod="$1" rel="${1#"$ROOT"/}"
    (
        cd "$mod"
        go build ./... && go vet ./... || exit 1
        if [ "$QUICK" -eq 0 ]; then
            go test -race ./... || exit 1
        fi
    )
}

if [ "${#MODULES[@]}" -eq 0 ]; then
    skip "go build/vet/test (no Go module changed)"
else
    info "verify.sh: ${#MODULES[@]} changed Go module(s)"
    for mod in "${MODULES[@]}"; do
        step "go build/vet$([ "$QUICK" -eq 0 ] && echo '/test -race')  ${mod#"$ROOT"/}" go_layer "$mod"
    done
fi

# ------------------------------------------------------------------- docker
#
# `go build` against the workspace go.work does NOT catch a missing
# `COPY libs/...` in the shared root Dockerfile. Only the bake does.
#
# This bake is a BUILD CHECK, not an image build: nothing downstream in this
# script consumes its output. It is therefore run with output=cacheonly, so it
# never writes to the docker image store.
#
# That matters because the store is machine-global while worktrees are not.
# docker-bake.hcl tags targets `<svc>:${ATLAS_IMAGE_TAG}` (default `local`),
# the same tag deploy/compose/docker-compose.*.yml runs — so a verify.sh in one
# worktree used to silently replace the `<svc>:local` image built from another
# tree's code, and two trees verifying the same service raced for the tag.
# cacheonly removes the write entirely rather than renaming it: the collision
# is gone and the images stop accumulating. The buildkit solve cache is still
# populated, so a later real build reuses this one's work — measured on
# atlas-ban with a source edit: the cacheonly run compiled it (27 CACHED
# steps), the normal bake that followed reported the go-build step itself
# CACHED (64 CACHED steps).
#
# A broken build still FAILS under cacheonly (every stage runs; only the export
# is dropped) — verified against both a bad COPY path and a Go type error.
# To actually PRODUCE runnable `<svc>:local` images, use tools/build-services.sh.
BAKE_OUTPUT='*.output=type=cacheonly'

bake_targets() {
    # services whose go.mod (or the shared Dockerfile / bake file) changed
    # See touched(): no `-q` under pipefail.
    if printf '%s\n' "$CHANGED" | grep -E '^(Dockerfile|docker-bake\.hcl|go\.work)$' >/dev/null; then
        python3 -c "
import json
d=json.load(open('.github/config/services.json'))
for s in d.get('services',[]):
    if s.get('type')=='go-service': print(s['name'])
"
        return
    fi
    python3 -c "
import json,sys
changed=set(sys.stdin.read().split())
d=json.load(open('.github/config/services.json'))
for s in d.get('services',[]):
    if s.get('type')!='go-service': continue
    p=s.get('path','')
    if any(c.startswith(p+'/') and c.endswith('go.mod') for c in changed):
        print(s['name'])
" <<<"$CHANGED"
}

if [ "$NO_DOCKER" -eq 1 ]; then
    skip "docker buildx bake (--no-docker/--quick)"
else
    TARGETS=()
    if [ "$ALL" -eq 1 ]; then
        TARGETS=(all-go-services)
    else
        # Fail CLOSED. Consuming bake_targets through `< <(...)` discards its
        # exit status, so a missing python3 or an unparseable services.json
        # would yield zero targets and read as "no go.mod touched" — silently
        # downgrading the one mandatory check into a green pass.
        bake_out=""
        if ! bake_out="$(bake_targets)"; then
            BAKE_RESOLVE_FAILED=1
            FAILED+=("docker bake target resolution (services.json/python3)")
            printf '\033[31m✗ could not resolve bake targets — refusing to report a skip\033[0m\n'
            bake_out=""
        fi
        while IFS= read -r t; do [ -n "$t" ] && TARGETS+=("$t"); done <<<"$bake_out"
    fi
    if [ "${BAKE_RESOLVE_FAILED:-0}" -eq 1 ]; then
        : # already recorded as a failure above; never report this as a skip
    elif [ "${#TARGETS[@]}" -eq 0 ]; then
        skip "docker buildx bake (no go.mod touched)"
    else
        for t in "${TARGETS[@]}"; do
            step "docker buildx bake $t" docker buildx bake --set "$BAKE_OUTPUT" "$t"
        done
    fi
fi

# ------------------------------------------------------------------- guards
#
# Gate the analyzer guards on "a .go file actually changed": a Go analyzer
# cannot find a new violation in a diff that contains no Go. CI is gated the
# same way now (on its affected-module matrices, plus a tools/-changed signal),
# and still runs them on every PR that touches Go.
#
# All four analyzers run in ONE sweep. Each used to rebuild its analyzer into a
# temp dir and walk ~60 modules with a standalone go/analysis driver that
# type-checks from source every time (measured: 46.4s then 47.5s back to back).
# They now share a single `go vet -vettool=` binary, so the tree is type-checked
# once and Go's per-package fact cache does the rest. Run one on its own with
# tools/<name>-guard.sh when iterating on that analyzer.

go_analyzer_guards() {
    # Scope the sweep to the modules this run already identified as changed.
    # An empty list means "sweep that root entirely", which is what we want in
    # the two cases that produce one: --all, and a .go change that lives
    # outside services/ and libs/ (an analyzer's own source under tools/,
    # where the verdict on unchanged code can move).
    local svc="" lib="" m
    if [ "$ALL" -eq 0 ]; then
        for m in ${MODULES[@]+"${MODULES[@]}"}; do
            case "$m" in
                "$ROOT"/services/*) svc="$svc$m"$'\n' ;;
                "$ROOT"/libs/*)     lib="$lib$m"$'\n' ;;
            esac
        done
    fi
    GUARD_SERVICE_MODULES="$svc" GUARD_LIB_MODULES="$lib" ./tools/go-analyzer-guards.sh
}

if [ "$ALL" -eq 1 ] || touched '\.go$'; then
    step "go analyzer guards"     go_analyzer_guards
    step "skill/job id guard"     ./tools/skill-job-id-guard.sh
else
    skip "Go analyzer guards (no .go file changed)"
fi

# FR-8.5 (task-232): keeps the query-scope audit from rotting while
# Phases B-F are still in flight — a newly-added unscoped entity.go struct
# or call site must fail CI. Gated separately from go_analyzer_guards above
# (rather than folded into the shared vettool) so a scopeguard-only change
# doesn't force a rebuild/re-run of the other three analyzers, and vice versa.
#
# The predicate is fleet-wide over ANY changed .go file, not just entity.go:
# Rule 2 (the call-site check) fires on any GORM call site in services/ or
# libs/, not only ones that live in an entity.go file — a scoped-out
# `s.db.Model(...)` in a brand-new scheduler.go would otherwise merge clean
# on a diff that touches no entity.go (fix round 2, BLOCKING finding). The
# old `entity\.go$` half stays as an explicit alternative purely for
# readability at the call site; it is already implied by `\.go$`.
if [ "$ALL" -eq 1 ] || touched '\.go$|^tools/scopeguard/'; then
    step "scope guard" ./tools/scope-guard.sh
else
    skip "scope guard (no Go file changed)"
fi

# task-232 FR-4.1: bans new direct producer.Produce calls under services/ that
# bypass producer.ProviderImpl's composed header decorators (span + tenant +
# environment). Gated on ANY services/ Go change — not just files matching
# *producer*.go — because the violation this guards against can appear in any
# services/ file (a processor.go, a handler, a saga step), not only ones named
# for the seam they call into. This repo's own allowlist proves the narrower
# predicate was wrong: two of the four pre-existing call sites
# (reactor/processor.go, party_quest/processor.go) contain no "producer"
# substring in their path and would have skipped the gated (non --all) run
# entirely. Also gated on libs/atlas-kafka/ (the seam itself) and the guard's
# own source, mirroring the scope guard's self-inclusion above.
if [ "$ALL" -eq 1 ] || touched '^services/.*\.go$|^libs/atlas-kafka/|^tools/producerseamguard/|^tools/producer-seam-guard\.sh$'; then
    step "producer seam guard" ./tools/producer-seam-guard.sh
else
    skip "producer seam guard (no services/ or atlas-kafka Go file changed)"
fi

# Path-gated in CI.

if touched '^(\.github/config/services\.json|deploy/k8s/|docker-bake\.hcl|go\.work|tools/db-bootstrap\.sh)'; then
    step "service registration guard" ./tools/service-registration-guard.sh
else
    skip "service registration guard (no registration list changed)"
fi

# task-232 Task 29A: asserts every service Deployment/StatefulSet/DaemonSet
# carries SERVICE_NAME sourced via the downward API from its own `app` pod
# label — libs/atlas-env/registry.go MapRegistry.IsOwner keys ownership on
# it, so a missing or wrong-form value is a silent traffic misroute, not a
# build failure. Gated on any deploy/k8s/ manifest change plus the guard's
# own source, mirroring the service registration guard predicate above.
if touched '^deploy/k8s/|^tools/service-name-guard'; then
    step "service name guard" ./tools/service-name-guard.sh
else
    skip "service name guard (no deploy/k8s manifest changed)"
fi

if touched '^services/atlas-configurations/seed-data/templates/'; then
    step "template opcode order guard"       ./tools/template-opcode-order-guard.sh
    step "template duplicate binding guard"  ./tools/template-duplicate-binding-guard.sh
    step "template movement types guard"     ./tools/template-movement-types-guard.sh
else
    skip "template guards (no tenant socket-config template changed)"
fi

if touched '^services/atlas-channel/atlas\.com/channel/socket/handler/.*\.go$' \
    || touched '^services/atlas-configurations/seed-data/templates/'; then
    step "operator cancel path guard" ./tools/operator-cancel-path-guard.sh
else
    skip "operator cancel path guard (no socket handler or tenant template changed)"
fi

if touched 'kafka/message/trade/kafka\.go'; then
    step "trade contract mirror guard" ./tools/trade-contract-mirror-guard.sh
else
    skip "trade contract mirror guard (contract unchanged)"
fi

if touched 'kafka/message/mist/kafka\.go'; then
    step "mist contract mirror guard" ./tools/mist-contract-mirror-guard.sh
else
    skip "mist contract mirror guard (contract unchanged)"
fi

if touched 'kafka/message/shops/kafka\.go'; then
    step "npc-shop contract mirror guard" ./tools/npc-shop-contract-mirror-guard.sh
else
    skip "npc-shop contract mirror guard (contract unchanged)"
fi

if touched '^services/.*\.go$|^tools/envguard/|^tools/env-domain-guard\.sh$'; then
    step "env domain guard" ./tools/env-domain-guard.sh
else
    skip "env domain guard (no service Go file changed)"
fi

if touched '^services/.*/main\.go$|^tools/envguard/'; then
    step "env bootstrap guard" ./tools/env-bootstrap-guard.sh
else
    skip "env bootstrap guard (no service main.go changed)"
fi

if touched 'kafka/message/npc/kafka\.go'; then
    step "npc-conversation contract mirror guard" ./tools/npc-conversation-contract-mirror-guard.sh
else
    skip "npc-conversation contract mirror guard (contract unchanged)"
fi

if touched '^tools/.*\.sh$'; then
    step "shell tooling guard" ./tools/shell-guard.sh --require-shellcheck

    # Run the test suite belonging to each changed tools/ script — whether the
    # script changed or its own _test.sh did. This replaces a rule hardcoded to
    # task-resolve/task-brief, under which every other script in tools/ was
    # ungated: a branch adding three tools/ scripts saw all 14 checks skip and
    # still exited 0.
    suites="$(changed_tool_suites)"
    if [ -n "$suites" ]; then
        while IFS= read -r suite; do
            [ -n "$suite" ] || continue
            step "$(basename "$suite")" "./$suite"
        done <<EOF
$suites
EOF
    else
        skip "tools test suites (no changed script has one)"
    fi
else
    skip "shell tooling guard (no tools/ script changed)"
    skip "tools test suites (no tools/ script changed)"
fi

# atlas-pr-bootstrap's shell is the PR environment's whole control plane —
# bootstrap.sh, cleanup.sh and their helpers create and reclaim every
# ephemeral env. It ships a substantial bats suite, but nothing ran it: the
# tools/ gate above only reaches tools/, only matches *_test.sh (these are
# *.bats), and `bats` appears nowhere in .github/workflows. So the suite was
# advisory, and the sparse SERVICE_ID defect (empty uuidgen output silently
# becoming an empty env var, crash-looping atlas-channel/atlas-login) shipped
# past a test file that could have expressed it. Gate it like any other suite.
if touched '^services/atlas-pr-bootstrap/'; then
    if command -v bats >/dev/null 2>&1; then
        step "atlas-pr-bootstrap bats suite" bats services/atlas-pr-bootstrap/test
    else
        # Deliberately a hard failure, not a skip. A silent skip is how this
        # suite went unrun in the first place; `bats` is already part of the
        # toolchain tools/task-facts.sh probes for.
        step "atlas-pr-bootstrap bats suite" \
            sh -c 'echo "bats is not installed — install it to verify services/atlas-pr-bootstrap (see tools/task-facts.sh toolchain probe)" >&2; exit 1'
    fi
else
    skip "atlas-pr-bootstrap bats suite (service unchanged)"
fi

# The PreToolUse/PostToolUse hooks are shell too, and they gate every tool call
# in every session — a broken one is not a lint problem, it is a workflow
# outage. Their suites live beside them rather than under tools/, so the
# tools/-changed gate above does not reach them.
if touched '^\.claude/hooks/'; then
    hook_suites="$(find .claude/hooks -name '*_test.sh' -type f | sort)"
    if [ -n "$hook_suites" ]; then
        while IFS= read -r suite; do
            [ -n "$suite" ] || continue
            step "$(basename "$suite")" "./$suite"
        done <<EOF
$hook_suites
EOF
    else
        skip "hook test suites (none exist)"
    fi
else
    skip "hook test suites (no .claude/hooks/ change)"
fi

if touched '^(deploy/|tools/gen-lb-ports\.sh|.*versions\.json)'; then
    step "LB port drift"       ./tools/gen-lb-ports.sh --check
    step "routes drift"        ./tools/gen-routes.sh --check
    step "version coverage"    ./tools/check-version-coverage.sh
else
    skip "LB port / version coverage (no deploy or versions.json change)"
fi

if touched '^(deploy/k8s/base/atlas-.*\.yaml|docs/tasks/task-232-sparse-ephemeral-environments/query-scope-audit\.md|tools/gen-tenant-tables(_test)?\.sh|services/atlas-pr-bootstrap/scripts/tenant-tables\.txt)'; then
    step "tenant tables drift"  ./tools/gen-tenant-tables.sh --check
    step "tenant tables generator tests" ./tools/gen-tenant-tables_test.sh
else
    skip "tenant tables drift (no audit, DB_NAME manifest, or generator change)"
fi

if touched '^(deploy/k8s/overlays/pr/|deploy/k8s/overlays/pr-sparse/|tools/pr-sparse-mirror-guard\.sh)'; then
    step "pr-sparse mirror drift" ./tools/pr-sparse-mirror-guard.sh
else
    skip "pr-sparse mirror drift (neither overlay changed)"
fi

if touched '^(tools/mode-select(_test)?\.sh|\.github/actions/detect-changes/action\.yml)'; then
    step "mode select decision table" ./tools/mode-select_test.sh
else
    skip "mode select decision table (mode-select.sh / detect-changes unchanged)"
fi

# ------------------------------------------------------------- lint & format
#
# Scope the Go layer to the CHANGED modules, exactly as the lint-go CI job does
# (it feeds lint.sh the detect-changes module matrix). Unscoped, lint.sh walks
# all 86 modules and invokes golangci-lint 172 times — two per module, one
# process at a time — which is minutes of wall clock for a diff that touches
# none of them.

UI_CHANGED=0
if [ "$ALL" -eq 1 ] || touched '^services/atlas-ui/'; then
    UI_CHANGED=1
fi

if [ "${#MODULES[@]}" -gt 0 ]; then
    LINT_ARGS=(--check --go)
    if base="$(resolve_base)"; then
        LINT_ARGS+=(--base "$base")
    fi
    step "lint & format guard (${#MODULES[@]} module(s))" \
        ./tools/lint.sh "${LINT_ARGS[@]}" "${MODULES[@]}"
else
    skip "lint & format guard, Go layer (no Go module changed)"
fi

if [ "$NO_UI" -eq 1 ]; then
    skip "lint & format guard, UI layer (--no-ui)"
elif [ "$UI_CHANGED" -eq 1 ]; then
    step "lint & format guard (atlas-ui)" ./tools/lint.sh --check --ui
else
    skip "lint & format guard, UI layer (atlas-ui unchanged)"
fi

# ----------------------------------------------------------------- ui tests
#
# CI's test-ui job runs .github/actions/node-test, which is lint + `npm test` +
# `npm run build`. lint.sh --ui covers only the first of the three, and the
# build is what type-checks the test files — so a UI change with a failing
# vitest or a broken build passed this gate green and then failed CI.

ui_test_layer() {
    (
        # Same shim tools/lint.sh uses; no-op when node is already correct.
        # shellcheck source=lib/node-env.sh
        . "$ROOT/tools/lib/node-env.sh"
        cd "$ROOT/services/atlas-ui"
        # Same two commands node-test runs. `npm test` is already `vitest run`.
        npm test && npm run build
    )
}

if [ "$NO_UI" -eq 1 ] || [ "$QUICK" -eq 1 ]; then
    skip "atlas-ui tests + build (--no-ui/--quick)"
elif [ "$UI_CHANGED" -eq 1 ]; then
    step "atlas-ui tests + build" ui_test_layer
else
    skip "atlas-ui tests + build (atlas-ui unchanged)"
fi

# -------------------------------------------------------------------- facts
#
# Everything above has run its real selection logic; under --facts nothing has
# executed. Print what was selected and stop.
#
# Contract: `key=value`, one per line, stdout only, always exit 0. Callers grep
# it; adding a key is safe, renaming one is not.

if [ "$FACTS" -eq 1 ]; then
    csv() {
        # csv <newline-list> — collapse to a comma list, or "none"
        local v; v="$(printf '%s\n' "$1" | sed '/^$/d' | paste -sd, -)"
        printf '%s' "${v:-none}"
    }

    fanout="$(fanout_paths)"
    changed_services="$(printf '%s\n' "$CHANGED" | sed -n 's|^services/\([^/]*\)/.*|\1|p' | sort -u)"
    changed_libs="$(printf '%s\n' "$CHANGED" | sed -n 's|^libs/\([^/]*\)/.*|\1|p' | sort -u)"
    module_rel=""
    for m in ${MODULES[@]+"${MODULES[@]}"}; do
        module_rel="$module_rel${m#"$ROOT"/}"$'\n'
    done

    echo "base=$BASE_SHA"
    if [ "$ALL" -eq 1 ]; then
        echo "changed_paths=all"
    else
        echo "changed_paths=$(printf '%s\n' "$CHANGED" | sed '/^$/d' | wc -l | tr -d ' ')"
    fi
    echo "changed_services=$(csv "$changed_services")"
    echo "changed_libs=$(csv "$changed_libs")"
    echo "go_changed=$(touched '\.go$' && echo true || echo false)"
    echo "ui_changed=$([ "$UI_CHANGED" -eq 1 ] && echo true || echo false)"
    if [ -n "$fanout" ]; then
        echo "fanout_reason=shared-lib:$(printf '%s\n' "$fanout" | head -1)"
    elif [ "$ALL" -eq 1 ]; then
        echo "fanout_reason=--all"
    else
        echo "fanout_reason=none"
    fi
    echo "modules_selected=${#MODULES[@]}"
    # A full fan-out lists 80+ modules; the count above is the fact, the list is
    # only useful when it is short enough to act on.
    if [ "${#MODULES[@]}" -le 12 ]; then
        echo "modules=$(csv "$module_rel")"
    else
        echo "modules=(${#MODULES[@]} modules — fan-out, list suppressed)"
    fi
    echo "guard_suites=$(csv "$(changed_tool_suites)")"
    echo "bake_targets=$(csv "$(printf '%s\n' ${TARGETS[@]+"${TARGETS[@]}"})")"

    # Gates that WOULD run. Per-module build steps collapse to one line — the
    # count is already `modules_selected`.
    gates=""
    module_gate=""
    for label in ${SELECTED[@]+"${SELECTED[@]}"}; do
        case "$label" in
            # `|| true` is load-bearing: a bare assignment takes the exit status
            # of its last command substitution, so on --quick the false test
            # would abort the script under `set -e`.
            "go build/vet"*) module_gate="go build/vet$([ "$QUICK" -eq 0 ] && echo '/test -race' || true) (${#MODULES[@]} modules)" ;;
            *) gates="$gates$label"$'\n' ;;
        esac
    done
    [ -n "$module_gate" ] && gates="$module_gate"$'\n'"$gates"
    echo "gates_selected=$(printf '%s\n' "$gates" | sed '/^$/d' | wc -l | tr -d ' ')"
    printf '%s\n' "$gates" | sed '/^$/d' | while IFS= read -r g; do echo "gate=$g"; done
    echo "gates_skipped=${#SKIPPED[@]}"
    exit 0
fi

# ------------------------------------------------------------------ summary

printf '\n\033[1m════ verify.sh summary ════\033[0m\n'
for s in "${SKIPPED[@]}"; do printf '  \033[2m− %s\033[0m\n' "$s"; done
for s in "${PASSED[@]}";  do printf '  \033[32m✓\033[0m %s\n' "$s"; done
for s in "${FAILED[@]}";  do printf '  \033[31m✗ %s\033[0m\n' "$s"; done

if [ "${#FAILED[@]}" -gt 0 ]; then
    printf '\n\033[31m%d check(s) FAILED — the branch is not ready.\033[0m\n' "${#FAILED[@]}"
    exit 1
fi

if [ "$NO_DOCKER" -eq 1 ] || [ "$QUICK" -eq 1 ]; then
    printf '\n\033[33mAll checks passed, but docker bake was skipped — not a pre-PR pass.\033[0m\n'
    exit 0
fi

printf '\n\033[32mAll checks passed.\033[0m\n'
