#!/usr/bin/env bash
# tools/lib/module-graph.sh — the workspace require graph (Layer 5).
#
# `tools/verify.sh`'s `changed_modules` used to treat ANY changed `libs/` path
# as a reason to build every module in the workspace: services consume libs
# through go.work, so a lib edit can in principle break a service with no
# changed file of its own. Correct, but far wider than necessary — a lib deep
# in the graph with few consumers still triggered an 80+ module build.
#
# This library computes the actual blast radius: the transitive
# reverse-dependency closure of the changed module(s) over the workspace
# `require` graph. A module is in scope only if it changed directly, or if it
# (transitively) requires a module that did.
#
# Sourceable, not a standalone script — `tools/verify.sh` sets its own
# `set -euo pipefail`; this file must not fight that when sourced, so it does
# not set its own shell options at file scope.

# module_path_of <dir> — the `module` directive from <dir>/go.mod, or empty
# (and no error) when <dir> is not a Go module.
module_path_of() {
    local dir="$1"
    [ -f "$dir/go.mod" ] || return 0
    sed -n 's/^module[[:space:]]\+\([^[:space:]]\+\).*/\1/p' "$dir/go.mod" | head -1
}

# _module_graph_requires <go.mod-file> — every module path named in a
# `require` directive in the given file, one per line. Covers both the
# single-line form (`require example.com/foo v1.2.3`) and the block form
# (`require ( ... )`); an `// indirect` comment does not exclude the entry —
# it is still a real edge in the workspace graph.
_module_graph_requires() {
    awk '
        /^require[ \t]*\([ \t]*$/ { inblock = 1; next }
        inblock && /^\)[ \t]*$/ { inblock = 0; next }
        inblock {
            line = $0
            sub(/\/\/.*/, "", line)
            gsub(/^[ \t]+|[ \t]+$/, "", line)
            if (line != "") { split(line, a, /[ \t]+/); print a[1] }
            next
        }
        /^require[ \t]+[^( \t]/ {
            line = $0
            sub(/^require[ \t]+/, "", line)
            sub(/\/\/.*/, "", line)
            gsub(/^[ \t]+|[ \t]+$/, "", line)
            split(line, a, /[ \t]+/)
            print a[1]
        }
    ' "$1"
}

# module_consumers <root> <dir>... — the transitive reverse-dependency
# closure of the given module directories over the workspace require graph,
# unioned with the given directories themselves. Prints one absolute module
# directory per line, sorted and unique.
#
# A given directory that is not a Go module (no go.mod, or no `module`
# directive) is silently dropped — not an error, just not part of the result.
#
# Edges are restricted to module paths actually present under <root>;
# external (non-workspace) requires are irrelevant to this closure and are
# ignored. BFS with a visited set makes a require cycle terminate.
module_consumers() {
    local root="$1"
    shift
    local -a changed_dirs=("$@")

    local -A path_of_dir=()
    local -A dir_of_path=()
    local modfile dir path

    while IFS= read -r -d '' modfile; do
        dir="$(dirname "$modfile")"
        path="$(module_path_of "$dir")"
        [ -n "$path" ] || continue
        path_of_dir["$dir"]="$path"
        dir_of_path["$path"]="$dir"
    done < <(find "$root" -name go.mod -not -path '*/node_modules/*' -print0)

    # Reverse edges: for every workspace module, which OTHER workspace
    # modules require it. consumers_of[<required-path>] is a newline-joined
    # list of the paths that require it.
    local -A consumers_of=()
    local req
    for dir in "${!path_of_dir[@]}"; do
        path="${path_of_dir[$dir]}"
        while IFS= read -r req; do
            [ -n "$req" ] || continue
            [ -n "${dir_of_path[$req]:-}" ] || continue
            consumers_of["$req"]="${consumers_of[$req]:-}"$'\n'"$path"
        done < <(_module_graph_requires "$dir/go.mod")
    done

    local -A result_dirs=()
    local -A visited=()
    local -a queue=()
    local cdir raw

    for raw in "${changed_dirs[@]}"; do
        path="$(module_path_of "$raw")"
        [ -n "$path" ] || continue
        result_dirs["$raw"]=1
        queue+=("$path")
    done

    local qi=0 cpath
    while [ "$qi" -lt "${#queue[@]}" ]; do
        path="${queue[$qi]}"
        qi=$((qi + 1))
        [ -n "${visited[$path]:-}" ] && continue
        visited["$path"]=1
        [ -n "${consumers_of[$path]:-}" ] || continue
        while IFS= read -r cpath; do
            [ -n "$cpath" ] || continue
            cdir="${dir_of_path[$cpath]:-}"
            [ -n "$cdir" ] || continue
            result_dirs["$cdir"]=1
            queue+=("$cpath")
        done <<<"${consumers_of[$path]}"
    done

    printf '%s\n' "${!result_dirs[@]}" | sort -u
}
