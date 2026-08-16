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
        -h|--help) usage; exit 0 ;;
        *) echo "verify.sh: unknown option $1" >&2; usage >&2; exit 2 ;;
    esac
done

PASSED=()
FAILED=()
SKIPPED=()

step() {
    # step <label> <command...>
    local label="$1"; shift
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
if [ "$ALL" -eq 1 ]; then
    echo "verify.sh: --all — running every check"
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
        echo "verify.sh: change base $(git rev-parse --short "$base") — $(printf '%s\n' "$CHANGED" | sed '/^$/d' | wc -l) changed path(s)"
    else
        echo "verify.sh: WARNING — no merge base with origin/main or main; falling back to --all" >&2
        ALL=1
    fi
fi

touched() {
    # touched <grep-ere> — true when the change set matches, or when --all
    [ "$ALL" -eq 1 ] && return 0
    printf '%s\n' "$CHANGED" | grep -qE "$1"
}

# --------------------------------------------------------------- go modules

all_modules() {
    find "$ROOT/services" "$ROOT/libs" -name go.mod -not -path '*/node_modules/*' -print0 \
        | xargs -0 -r -n1 dirname | sort -u
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
    fanout="$(printf '%s\n' "$CHANGED" | grep -E '^(go\.work|libs/)' || true)"
    if [ -n "$fanout" ]; then
        if [ -z "$BASE" ]; then
            printf '\033[33mverify.sh: shared-lib change fans out to ALL modules (%s path(s) under go.work/libs/).\n' \
                "$(printf '%s\n' "$fanout" | wc -l | tr -d ' ')" >&2
            printf '           first: %s\n' "$(printf '%s\n' "$fanout" | head -1)" >&2
            printf '           This is the whole-branch diff. For a per-task iteration gate pass\n' >&2
            printf '           --base <last-gated-commit> to scope it to the increment under test.\033[0m\n' >&2
        else
            echo "verify.sh: shared-lib change in this increment — fanning out to all modules" >&2
        fi
        all_modules
        return
    fi
    local mods=() m rel
    while IFS= read -r m; do mods+=("$m"); done < <(all_modules)
    for m in "${mods[@]}"; do
        rel="${m#"$ROOT"/}"
        if printf '%s\n' "$CHANGED" | grep -qE "^${rel}/"; then
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
    echo "verify.sh: ${#MODULES[@]} changed Go module(s)"
    for mod in "${MODULES[@]}"; do
        step "go build/vet$([ "$QUICK" -eq 0 ] && echo '/test -race')  ${mod#"$ROOT"/}" go_layer "$mod"
    done
fi

# ------------------------------------------------------------------- docker
#
# `go build` against the workspace go.work does NOT catch a missing
# `COPY libs/...` in the shared root Dockerfile. Only the bake does.

bake_targets() {
    # services whose go.mod (or the shared Dockerfile / bake file) changed
    if printf '%s\n' "$CHANGED" | grep -qE '^(Dockerfile|docker-bake\.hcl|go\.work)$'; then
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
            step "docker buildx bake $t" docker buildx bake "$t"
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

if touched '^services/atlas-configurations/seed-data/templates/'; then
    step "template opcode order guard"       ./tools/template-opcode-order-guard.sh
    step "template duplicate binding guard"  ./tools/template-duplicate-binding-guard.sh
    step "template movement types guard"     ./tools/template-movement-types-guard.sh
else
    skip "template guards (no tenant socket-config template changed)"
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

if touched '^tools/task-(resolve|brief)(_test)?\.sh$'; then
    step "task resolve/brief tests" ./tools/task-resolve_test.sh
else
    skip "task resolve/brief tests (task tooling unchanged)"
fi

if touched '^(deploy/|tools/gen-lb-ports\.sh|.*versions\.json)'; then
    step "LB port drift"       ./tools/gen-lb-ports.sh --check
    step "version coverage"    ./tools/check-version-coverage.sh
else
    skip "LB port / version coverage (no deploy or versions.json change)"
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
