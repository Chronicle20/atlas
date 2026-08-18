#!/usr/bin/env bash
# plan-lint.sh — check an implementation plan against the rules
# .claude/commands/plan-task.md Step 5a already states, before Phase 4 turns
# each task section into an implementer brief.
#
# Why this exists
# ---------------
# On the task-231 execute chain the controller appended 18 `## CONTROLLER
# RULING` blocks to task briefs, each patching a defect in the plan, and most
# preceded by an investigation it had to run first. The rulings name what was
# wrong:
#
#   "R33-3 — SetEnabled MUST NOT schedule. The brief's Files line 12 and
#    Step 3 would not compile..."
#   "R39-1 — Step 2's command matches NOTHING. Fix..."
#   "...the four 'not implemented' methods. CLAUDE.md says plainly: No TODO,
#    stubbed handlers, or 501s in landed commits."
#
# Every one of those was discoverable at plan time with a grep. Instead they
# were discovered at dispatch time, by a controller sitting at 150-250k
# context, which is the most expensive place in the whole workflow to learn
# anything. Roughly a third of one session's tool calls were this cycle.
#
# Step 5a already says "Paths must be repo-relative and must exist (or be
# explicitly marked `new file`)" and "a task touching more than ~6 files ...
# gets split". Both rules were stated and neither was checked. This checks them.
#
# Usage: tools/plan-lint.sh PLAN [--no-commands] [--no-symbols] [--quiet]
#
#   --no-commands   skip F3 (do not execute the plan's read-only commands)
#   --no-symbols    skip F5 (do not index the repo's Go symbols)
#   --quiet         findings only, no per-check progress
#
# Exit codes:
#   0  no findings
#   1  findings reported
#   2  usage error / no such plan

set -u

plan=""; run_commands=1; run_symbols=1; quiet=0
for a in "$@"; do
    case "$a" in
        --no-commands) run_commands=0 ;;
        --no-symbols)  run_symbols=0 ;;
        --quiet)       quiet=1 ;;
        -h|--help)     sed -n '28,33p' "$0"; exit 0 ;;
        -*) printf 'plan-lint.sh: unknown flag %s\n' "$a" >&2; exit 2 ;;
        *)  plan="$a" ;;
    esac
done

[ -n "$plan" ] || { printf 'usage: tools/plan-lint.sh PLAN [--no-commands]\n' >&2; exit 2; }
[ -f "$plan" ] || { printf 'plan-lint.sh: no such plan: %s\n' "$plan" >&2; exit 2; }

root="$(git rev-parse --show-toplevel 2>/dev/null || pwd)"
findings=0
warnings=0
say()     { [ "$quiet" -eq 1 ] || printf '%s\n' "$*"; }
finding() { findings=$((findings + 1)); printf '  %s\n' "$*"; }
# F4 is advisory: Step 5a permits a deliberately large task provided context.md
# says why. Reporting it as an error would drown the hard failures in noise.
warn()    { warnings=$((warnings + 1)); printf '  %s\n' "$*"; }
# F4 and F5 advise different fixes, so each footer prints only if its own check
# fired. The counters are set by the checks; `warn` stays generic.
f4warns=0; f5warns=0

# A repo-relative path starts with a real top-level directory. Anything else on
# a Files bullet is a Go symbol (`monster.Model.SpawnSourceType`), a REST route
# (`/api/events`), a JSX fragment, or a brace/glob expansion — none of which are
# files to stat. Deriving the list from the tree keeps this honest as the repo
# changes.
top_dirs="$(git -C "$root" ls-tree --name-only -d HEAD 2>/dev/null | tr '\n' '|' | sed 's/|$//')"
[ -n "$top_dirs" ] || top_dirs='deploy|docs|libs|services|tools|dev|.claude|.github'

is_repo_path() {
    case "$1" in
        *'{'*|*'}'*|*'*'*|*' '*|*'<'*|*'"'*|*'('*) return 1 ;;  # expansion/markup, not a path
        /*) return 1 ;;                                          # absolute -> a URL route
    esac
    printf '%s' "$1" | grep -qE "^($top_dirs)/" || return 1
    return 0
}

# ---------------------------------------------------------------- F1 + F4 ---
# Walk each "## Task N" section, collect its "### Files" bullets, and check
# path existence (F1) and task size (F4).
say "F1/F4  file references and task size"

awk '
    /^## +Task +[0-9]+/ { task=$0; sub(/^## +/, "", task); infiles=0; next }
    /^### +Files/       { infiles=1; next }
    /^### /             { infiles=0; next }
    infiles && /^[-*] / { print task "\t" $0 }
' "$plan" > /tmp/plan-lint-files.$$ 2>/dev/null

current=""; count=0; services=""
emit_size() {
    [ -n "$current" ] || return 0
    # `grep -c` prints 0 AND exits 1 on no match, so `|| echo 0` would append a
    # second line and make this "0\n0" — an integer test error, not a count.
    nsvc="$(printf '%s\n' "$services" | grep -c . || true)"
    case "${nsvc:-0}" in ''|*[!0-9]*) nsvc=0 ;; esac
    if [ "$count" -gt 6 ]; then
        f4warns=$((f4warns + 1)); warn "F4 oversized: \"$current\" lists $count files (Step 5a splits at ~6)"
    fi
    if [ "$nsvc" -gt 1 ]; then
        f4warns=$((f4warns + 1)); warn "F4 multi-service: \"$current\" spans $nsvc services — Step 5a splits these"
    fi
}

while IFS="$(printf '\t')" read -r task bullet; do
    [ -n "$bullet" ] || continue
    if [ "$task" != "$current" ]; then
        emit_size
        current="$task"; count=0; services=""
    fi

    # First backticked token on the bullet is the path (drop any :line suffix).
    path="$(printf '%s' "$bullet" | sed -n 's/.*`\([^`]*\)`.*/\1/p' | head -1 | sed 's/:[0-9]*$//')"
    [ -n "$path" ] || continue
    is_repo_path "$path" || continue

    count=$((count + 1))
    svc="$(printf '%s' "$path" | sed -n 's|^\(services/[^/]*\).*|\1|p')"
    [ -n "$svc" ] && services="$(printf '%s\n%s' "$services" "$svc" | sort -u | grep -v '^$')"

    # "new file" / "new" marks a path that is expected not to exist yet.
    if printf '%s' "$bullet" | grep -qiE 'new file|\bnew\b|create'; then
        continue
    fi
    if [ ! -e "$root/$path" ]; then
        finding "F1 missing: $path  (in \"$task\") — does not exist and is not marked \`new file\`"
    fi
done < /tmp/plan-lint-files.$$
emit_size
rm -f /tmp/plan-lint-files.$$

# --------------------------------------------------------------------- F2 ---
# CLAUDE.md: "No // TODO, stubbed handlers, or 501s in landed commits."
# A plan that *specifies* one is a defect at plan time, not review time.
say "F2     planned stubs"
# A good plan mentions stub language in order to FORBID it — a sweep command, a
# RED-phase test expectation, an explicit "remove these before the phase ends".
# Flagging those is noise, and noise is what gets a linter ignored. Only a plan
# that specifies a stub with no adjacent guard is a finding.
grep -nEi 'not implemented|//[[:space:]]*TODO|\btodo!\(|StatusNotImplemented|\b501\b' "$plan" 2>/dev/null \
  | grep -viE '(grep|rg|ripgrep)[[:space:]]' \
  | grep -viE 'expected:|confirm|remove|remains|no output|must not|may land|forbid|no stub' \
  > /tmp/plan-lint-stubs.$$ 2>/dev/null || true
# Redirect, not a pipe: a `while` in a pipeline runs in a subshell and its
# increments to $findings are discarded.
while IFS= read -r hit; do
    [ -n "$hit" ] || continue
    finding "F2 planned stub: ${hit}"
done < /tmp/plan-lint-stubs.$$
rm -f /tmp/plan-lint-stubs.$$

# --------------------------------------------------------------------- F3 ---
# Commands the plan tells the implementer to run, that match nothing.
if [ "$run_commands" -eq 1 ]; then
    say "F3     read-only commands in the plan"
    grep -oE '^[[:space:]]*(grep|rg|ls|find)[[:space:]][^;&`$><]*$' "$plan" 2>/dev/null \
    | sed 's/^[[:space:]]*//' | sort -u > /tmp/plan-lint-cmds.$$ 2>/dev/null || true

    while IFS= read -r cmd; do
        [ -n "$cmd" ] || continue
        # Allowlist only, and refuse anything with shell metacharacters.
        case "$cmd" in
            grep\ *|rg\ *|ls\ *|find\ *) ;;
            *) continue ;;
        esac
        # `\|` is grep alternation, not a shell pipe — the commonest shape in
        # these plans. Mask it, then reject any pipe that survives.
        probe="$(printf '%s' "$cmd" | sed 's/\\|/@@ALT@@/g')"
        case "$probe" in
            *'|'*|*';'*|*'&'*|*'$('*|*'`'*|*'>'*|*'<'*) continue ;;
        esac
        # A trailing backslash means the command continues on the next line;
        # what we extracted is a fragment and would "match nothing" trivially.
        case "$cmd" in *\\) continue ;; esac
        # `find` is only read-only if you exclude the primaries that act. The
        # metacharacter test above does not cover these: `-exec ... +` needs no
        # `;` to terminate, and `-delete` needs no subcommand at all. Both
        # would run with cwd inside the repo.
        case " $cmd " in
            *' -delete '*|*' -exec '*|*' -execdir '*|*' -ok '*|*' -okdir '*|\
            *' -fls '*|*' -fprint '*|*' -fprint0 '*|*' -fprintf '*)
                warn "F3 skipped (acting primary, not read-only): $cmd"
                continue ;;
        esac
        if ! ( cd "$root" && timeout 20 sh -c "$cmd" ) >/dev/null 2>&1; then
            finding "F3 matches nothing: $cmd"
        fi
    done < /tmp/plan-lint-cmds.$$
    rm -f /tmp/plan-lint-cmds.$$
fi

# --------------------------------------------------------------------- F5 ---
# Symbols the plan's Go blocks call that resolve nowhere.
#
# F1 checks the paths a plan names. Nothing checked the *symbols* it names, and
# those are written from memory at 250k context by a planner with no compiler.
# On the task-238 plan chain roughly thirty tool calls in the session's most
# expensive stretch were the planner hand-verifying its own snippets — grepping
# for `newCtxTenant`, `SetCashScene`, `NewModelBuilder` to find out whether the
# API it had just written into the plan exists. That is this check, run by hand.
#
# What counts as "resolves": the name is defined anywhere in the repo's Go
# source, OR called anywhere in it, OR appears anywhere in the plan's own Go
# blocks outside a call site. The second clause is what keeps this quiet without
# a hand-curated allowlist — if the repo already calls `require.NoError` or
# `db.Where`, those names are part of this codebase's vocabulary and a plan may
# use them freely. The third clause lets a plan introduce its own vocabulary:
# a helper it defines in Task 2 and calls in Task 6 resolves against itself.
#
# Scope is deliberately narrow — selector calls (`.Name(`) only. The obvious
# companion rule, "unqualified lowercase calls must be package-local helpers",
# was measured against all 263 plans in docs/tasks/ and fired 216 times, almost
# all of them benign: helpers the plan legitimately introduces in prose, plus
# SQL, Lua and IDA pseudocode sitting inside ```go fences. A check that noisy
# gets ignored, which costs more than it saves. The selector rule fires 107
# times across 38 of those 263 plans — a rate a planner can actually act on.
#
# Advisory, like F4. A plan may legitimately be the first thing in the repo to
# touch an external API (task-071's `multipart.CreateFormFile` is a true
# example), and that is a "confirm the signature" prompt, not a build break.
if [ "$run_symbols" -eq 1 ]; then
    say "F5     symbols named in the plan's Go blocks"

    awk '
        /^[[:space:]]*```go[[:space:]]*$/ { ing=1; next }
        /^[[:space:]]*```/                { ing=0; next }
        ing                               { print }
    ' "$plan" > /tmp/plan-lint-go.$$ 2>/dev/null || true

    if [ -s /tmp/plan-lint-go.$$ ]; then
        # The repo's whole Go vocabulary: everything it defines, everything it
        # calls. One pass; ~28k names on this tree.
        { grep -rhoE '^func (\([^)]*\) )?[A-Za-z_][A-Za-z0-9_]*' --include='*.go' \
              --exclude-dir=.worktrees --exclude-dir=node_modules "$root" 2>/dev/null \
            | sed -E 's/^func (\([^)]*\) )?//'
          grep -rhoE '\.[A-Za-z_][A-Za-z0-9_]*\(' --include='*.go' \
              --exclude-dir=.worktrees --exclude-dir=node_modules "$root" 2>/dev/null \
            | tr -d '.('
        } | sort -u > /tmp/plan-lint-idx.$$ 2>/dev/null || true

        # The plan's own vocabulary. Blanking call sites first is what makes
        # this "declared, not merely called" — then add back the definitions,
        # whose names are followed by `(` and would have been blanked with them.
        { sed -E 's/[A-Za-z_][A-Za-z0-9_]*\(/ /g' /tmp/plan-lint-go.$$ \
            | grep -oE '[A-Za-z_][A-Za-z0-9_]*'
          grep -hoE '^[[:space:]]*func (\([^)]*\) )?[A-Za-z_][A-Za-z0-9_]*' /tmp/plan-lint-go.$$ \
            | sed -E 's/^[[:space:]]*func (\([^)]*\) )?//'
        } | sort -u > /tmp/plan-lint-vocab.$$ 2>/dev/null || true

        sort -u /tmp/plan-lint-idx.$$ /tmp/plan-lint-vocab.$$ > /tmp/plan-lint-known.$$
        grep -hoE '\.[A-Za-z_][A-Za-z0-9_]*\(' /tmp/plan-lint-go.$$ \
          | tr -d '.(' | sort -u > /tmp/plan-lint-sel.$$ 2>/dev/null || true

        comm -23 /tmp/plan-lint-sel.$$ /tmp/plan-lint-known.$$ > /tmp/plan-lint-unk.$$ 2>/dev/null || true
        # Redirect, not a pipe — same subshell trap as F2's loop.
        while IFS= read -r sym; do
            [ -n "$sym" ] || continue
            f5warns=$((f5warns + 1)); warn "F5 unknown symbol: ${sym} — nothing in the repo defines or calls it, and the plan does not declare it"
        done < /tmp/plan-lint-unk.$$
        rm -f /tmp/plan-lint-idx.$$ /tmp/plan-lint-vocab.$$ /tmp/plan-lint-known.$$ \
              /tmp/plan-lint-sel.$$ /tmp/plan-lint-unk.$$
    fi
    rm -f /tmp/plan-lint-go.$$
fi

# ------------------------------------------------------------------ verdict --
printf '\n'
if [ "$findings" -eq 0 ] && [ "$warnings" -eq 0 ]; then
    printf 'plan-lint: clean — %s\n' "$plan"
    exit 0
fi
printf 'plan-lint: %d error(s), %d warning(s) in %s\n' "$findings" "$warnings" "$plan"
if [ "$f4warns" -gt 0 ]; then
    printf '  F4 warnings are advisory — Step 5a allows a deliberately large task\n'
    printf '  provided context.md records why. Oversized tasks are what produce\n'
    printf '  PARTIAL hand-backs, so split them unless you have a reason.\n'
fi
if [ "$f5warns" -gt 0 ]; then
    printf '  F5 warnings are advisory too, but grep each one before you commit:\n'
    printf '  either the symbol exists and this is the first use in the repo, or\n'
    printf '  you invented it and an implementer will hit it at dispatch time.\n'
fi
if [ "$findings" -gt 0 ]; then
    printf '  Fix the errors before /execute-task. Each one otherwise costs a\n'
    printf '  CONTROLLER RULING at dispatch time, discovered at 150-250k context\n'
    printf '  instead of here.\n'
    exit 1
fi
exit 0
