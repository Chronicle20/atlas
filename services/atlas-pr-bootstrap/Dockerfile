# syntax=docker/dockerfile:1.4
# Atlas PR-env bootstrap/cleanup image. Two entrypoints share this image:
#   /atlas/bootstrap.sh (PostSync) and /atlas/cleanup.sh (PostDelete).
# Single-stage Alpine. rpk (Redpanda CLI; Kafka admin protocol compatible)
# replaces the Apache Kafka tarball + JRE — single static Go binary,
# ~30 MB vs. ~235 MB and no JVM startup latency per cleanup invocation.
FROM alpine:3.23

ARG RPK_VERSION=24.3.1

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

WORKDIR /atlas
COPY scripts/lib.sh /atlas/lib.sh
COPY scripts/bootstrap.sh /atlas/bootstrap.sh
COPY scripts/cleanup.sh /atlas/cleanup.sh
COPY canonical/ /atlas/canonical/

RUN chmod +x /atlas/bootstrap.sh /atlas/cleanup.sh

ENTRYPOINT ["/atlas/bootstrap.sh"]
