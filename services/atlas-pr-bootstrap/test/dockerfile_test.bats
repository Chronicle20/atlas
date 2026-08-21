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
        [ "$base" = "version-ports.sh" ] && continue
        [ "$base" = "service-config.sh" ] && continue
        [ "$base" = "env-record.sh" ] && continue
        if ! printf '%s\n' "$chmod_line" | grep -qF "/atlas/${base}"; then
            missing+=("$base")
        fi
    done
    if [ "${#missing[@]}" -ne 0 ]; then
        echo "Dockerfile chmod +x line missing entries for: ${missing[*]}" >&2
        return 1
    fi
}

@test "Dockerfile copies tenant-tables.txt (sweep_tenant's data dependency)" {
    # sweep-orphans.sh's sweep_tenant reads "$(dirname "$0")/tenant-tables.txt"
    # at runtime — a missing COPY means --sweep-tenant dies in-cluster with
    # "missing /atlas/tenant-tables.txt" the first time the documented
    # recovery path (sweep-orphans.sh --apply) is actually run, six hours
    # after the image shipped. Fail the build instead.
    grep -qE '^COPY scripts/tenant-tables\.txt /atlas/tenant-tables\.txt$' "$PROJECT_ROOT/Dockerfile"
}

@test "Dockerfile installs util-linux (provides uuidgen for sparse service rows)" {
    # sparse mode's create_service_config mints a fresh services-row id with
    # uuidgen. It was absent from the apk list, so the id came out empty and
    # atlas-channel/atlas-login crash-looped on uuid.MustParse(""). new_uuid
    # now fails loudly instead, but the package must still be here or every
    # sparse environment falls back to /proc — assert the intended source.
    grep -qE '^[[:space:]]*util-linux[[:space:]]*\\?$' "$PROJECT_ROOT/Dockerfile"
}
