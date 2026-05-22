#!/usr/bin/env bats

setup() {
    PROJECT_ROOT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
}

@test "Dockerfile copies every script under scripts/" {
    local missing=()
    for f in "$PROJECT_ROOT"/scripts/*.sh; do
        local base
        base="$(basename "$f")"
        if ! grep -qE "^COPY scripts/${base} /atlas/${base}\$" "$PROJECT_ROOT/Dockerfile"; then
            missing+=("$base")
        fi
    done
    if [ "${#missing[@]}" -ne 0 ]; then
        echo "Dockerfile missing COPY for: ${missing[*]}" >&2
        return 1
    fi
}

@test "Dockerfile chmod +x covers every executable script under scripts/" {
    # lib.sh is sourced by sibling scripts (bootstrap/cleanup/sweep-orphans),
    # not invoked directly, so it does not need the executable bit. Every
    # other *.sh is an entrypoint and must be chmod +x'd.
    local chmod_line
    chmod_line=$(grep -E '^RUN chmod \+x /atlas/' "$PROJECT_ROOT/Dockerfile" | head -1)
    [ -n "$chmod_line" ]
    local missing=()
    for f in "$PROJECT_ROOT"/scripts/*.sh; do
        local base
        base="$(basename "$f")"
        [ "$base" = "lib.sh" ] && continue
        if ! printf '%s\n' "$chmod_line" | grep -qF "/atlas/${base}"; then
            missing+=("$base")
        fi
    done
    if [ "${#missing[@]}" -ne 0 ]; then
        echo "Dockerfile chmod +x line missing entries for: ${missing[*]}" >&2
        return 1
    fi
}
