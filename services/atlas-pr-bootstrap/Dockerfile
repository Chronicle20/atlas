# syntax=docker/dockerfile:1.24
# Atlas PR-env bootstrap/cleanup image. Two entrypoints share this image:
#   /atlas/bootstrap.sh (PostSync) and /atlas/cleanup.sh (PostDelete).
# Single-stage Alpine. rpk (Redpanda CLI; Kafka admin protocol compatible)
# replaces the Apache Kafka tarball + JRE — single static Go binary,
# ~30 MB vs. ~235 MB and no JVM startup latency per cleanup invocation.
FROM alpine:3.23

# Bumping RPK_VERSION invalidates test/fixtures/rpk-*.json — rpk's
# --format json schema is a stability boundary, not a CLI flag. After
# bumping, regenerate per test/fixtures/README.md and re-run
# `bats services/atlas-pr-bootstrap/test/`.
ARG RPK_VERSION=24.3.1
# mc (MinIO Client) is fetched from MinIO's release CDN; the binary at
# `RELEASE.<date>` is the pinned version. Used by `sweep-orphans.sh`'s
# `sweep_minio` phase to enumerate and delete orphan per-tenant
# prefixes under `atlas-wz`, `atlas-assets`, `atlas-renders` (issue
# #596 — MinIO storage leak across PR teardowns).
#
# Bump path: `curl -sL https://dl.min.io/client/mc/release/linux-amd64/mc -o mc && chmod +x mc && ./mc --version`
# prints the latest release tag MinIO is serving; copy that here.
# Anything older than ~2 years is still served from the archive path
# below, so picking a stale tag won't break the build until MinIO
# rotates retention.
ARG MC_VERSION=RELEASE.2025-08-13T08-35-41Z

RUN --mount=type=cache,target=/var/cache/apk,sharing=locked \
    apk add \
        bash \
        curl \
        jq \
        postgresql-client \
        redis \
        ca-certificates \
        github-cli \
        kubectl \
        unzip

# Fetch rpk static binary. Pin version so JSON output schema and admin
# RPC semantics are deterministic; bumping is a deliberate Dockerfile
# change.
RUN curl -fsSL --retry 3 --retry-delay 5 \
        "https://github.com/redpanda-data/redpanda/releases/download/v${RPK_VERSION}/rpk-linux-amd64.zip" \
        -o /tmp/rpk.zip && \
    unzip /tmp/rpk.zip -d /usr/local/bin && \
    chmod +x /usr/local/bin/rpk && \
    rm /tmp/rpk.zip

# Fetch mc (MinIO Client) static binary.
RUN curl -fsSL --retry 3 --retry-delay 5 \
        "https://dl.min.io/client/mc/release/linux-amd64/archive/mc.${MC_VERSION}" \
        -o /usr/local/bin/mc && \
    chmod +x /usr/local/bin/mc

WORKDIR /atlas
COPY scripts/lib.sh /atlas/lib.sh
COPY scripts/bootstrap.sh /atlas/bootstrap.sh
COPY scripts/cleanup.sh /atlas/cleanup.sh
COPY scripts/sweep-orphans.sh /atlas/sweep-orphans.sh
COPY scripts/reclaim-main-bare-keys.sh /atlas/reclaim-main-bare-keys.sh
COPY canonical/ /atlas/canonical/

RUN chmod +x /atlas/bootstrap.sh /atlas/cleanup.sh /atlas/sweep-orphans.sh /atlas/reclaim-main-bare-keys.sh

ENTRYPOINT ["/atlas/bootstrap.sh"]
