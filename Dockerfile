# syntax=docker/dockerfile:1.24
#
# Shared Atlas Dockerfile. One file builds every Go service in
# .github/config/services.json (.services[] | select(.type=="go-service")).
#
# Usage:
#   docker build -f Dockerfile --build-arg SERVICE=atlas-<name> .
# Preferred:
#   docker buildx bake atlas-<name>
# Build everything:
#   docker buildx bake all-go-services
#
# Adding a new shared lib requires appending two COPY lines to this
# file (one to the mod-only block, one to the source block) AND adding
# the lib name to the synthesized go.work `for L in ...` loop below.
# That's it — no per-service edits.
ARG GO_VERSION=1.25.5
ARG ALPINE_VERSION=3.21

FROM golang:${GO_VERSION}-alpine${ALPINE_VERSION} AS build-env

ARG SERVICE
RUN test -n "${SERVICE}" || (echo "ERROR: build arg SERVICE is required (e.g., atlas-account)" >&2 && exit 1)

RUN apk add --no-cache git

WORKDIR /app

# Layer: all 17 atlas libs' go.mod (and go.sum where present — atlas-retry and
# atlas-service have no external deps so no go.sum exists). Lib-mod-only layer;
# shared across every target.
COPY libs/atlas-constants/go.mod   libs/atlas-constants/go.sum   libs/atlas-constants/
COPY libs/atlas-database/go.mod    libs/atlas-database/go.sum    libs/atlas-database/
COPY libs/atlas-kafka/go.mod       libs/atlas-kafka/go.sum       libs/atlas-kafka/
COPY libs/atlas-lock/go.mod        libs/atlas-lock/go.sum        libs/atlas-lock/
COPY libs/atlas-model/go.mod       libs/atlas-model/go.sum       libs/atlas-model/
COPY libs/atlas-object-id/go.mod   libs/atlas-object-id/go.sum   libs/atlas-object-id/
COPY libs/atlas-opcodes/go.mod     libs/atlas-opcodes/go.sum     libs/atlas-opcodes/
COPY libs/atlas-packet/go.mod      libs/atlas-packet/go.sum      libs/atlas-packet/
COPY libs/atlas-redis/go.mod       libs/atlas-redis/go.sum       libs/atlas-redis/
COPY libs/atlas-rest/go.mod        libs/atlas-rest/go.sum        libs/atlas-rest/
COPY libs/atlas-retry/go.mod       libs/atlas-retry/
COPY libs/atlas-saga/go.mod        libs/atlas-saga/go.sum        libs/atlas-saga/
COPY libs/atlas-script-core/go.mod libs/atlas-script-core/go.sum libs/atlas-script-core/
COPY libs/atlas-service/go.mod     libs/atlas-service/
COPY libs/atlas-socket/go.mod      libs/atlas-socket/go.sum      libs/atlas-socket/
COPY libs/atlas-tenant/go.mod      libs/atlas-tenant/go.sum      libs/atlas-tenant/
COPY libs/atlas-tracing/go.mod     libs/atlas-tracing/go.sum     libs/atlas-tracing/
COPY libs/atlas-wz/go.mod          libs/atlas-wz/go.sum          libs/atlas-wz/

# Layer: this service's tree (per-target; brings in its go.mod and source).
COPY services/${SERVICE}/atlas.com/ services/${SERVICE}/atlas.com/

# Layer: all 17 atlas libs' source trees (shared across every target; invalidates
# when any lib source changes — same invalidation profile as today).
COPY libs/atlas-constants   libs/atlas-constants
COPY libs/atlas-database    libs/atlas-database
COPY libs/atlas-kafka       libs/atlas-kafka
COPY libs/atlas-lock        libs/atlas-lock
COPY libs/atlas-model       libs/atlas-model
COPY libs/atlas-object-id   libs/atlas-object-id
COPY libs/atlas-opcodes     libs/atlas-opcodes
COPY libs/atlas-packet      libs/atlas-packet
COPY libs/atlas-redis       libs/atlas-redis
COPY libs/atlas-rest        libs/atlas-rest
COPY libs/atlas-retry       libs/atlas-retry
COPY libs/atlas-saga        libs/atlas-saga
COPY libs/atlas-script-core libs/atlas-script-core
COPY libs/atlas-service     libs/atlas-service
COPY libs/atlas-socket      libs/atlas-socket
COPY libs/atlas-tenant      libs/atlas-tenant
COPY libs/atlas-tracing     libs/atlas-tracing
COPY libs/atlas-wz          libs/atlas-wz

# Synthesize a minimal go.work containing only the 17 libs + the target service.
# The repo-root go.work also lists ~50 sibling services + 2 tools/* modules
# whose go.mod files are not in the build context; copying it directly fails
# workspace-load. Equivalent to what the legacy per-service Dockerfiles did.
RUN MOD_DIR=$(ls -d services/${SERVICE}/atlas.com/*/ | head -1 | sed 's:/$::') \
    && test -n "$MOD_DIR" || (echo "ERROR: no module dir under services/${SERVICE}/atlas.com/" >&2 && exit 1) \
    && test -f "${MOD_DIR}/go.mod" || (echo "ERROR: ${MOD_DIR}/go.mod missing" >&2 && exit 1) \
    && { \
         printf 'go 1.25.5\n\nuse (\n'; \
         for L in atlas-constants atlas-database atlas-kafka atlas-lock atlas-model \
                  atlas-object-id atlas-opcodes atlas-packet atlas-redis atlas-rest \
                  atlas-retry atlas-saga atlas-script-core atlas-service atlas-socket \
                  atlas-tenant atlas-tracing atlas-wz; do \
           printf '    ./libs/%s\n' "$L"; \
         done; \
         printf '    ./%s\n)\n' "$MOD_DIR"; \
       } > go.work

# Discover the inner module dir (services/${SERVICE}/atlas.com/<inner>) and build.
# Atlas convention: exactly one inner directory per service.
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    MOD_DIR=$(ls -d services/${SERVICE}/atlas.com/*/ | head -1) \
    && go build -C "$MOD_DIR" -o /server

# Stash this service's config.yaml in a known location for the runtime stage to COPY.
# Not every service ships a config.yaml (most don't); when absent, emit an empty
# placeholder so the runtime-stage COPY has a stable source path. The legacy
# per-service Dockerfiles for services without a config simply omit the COPY;
# emitting an empty /config.yaml is behaviorally equivalent (no service reads
# from /config.yaml unless it explicitly opens it, and the ones that do — e.g.
# atlas-account, atlas-ban — get the real file).
RUN MOD_DIR=$(ls -d services/${SERVICE}/atlas.com/*/ | head -1) \
    && if [ -f "${MOD_DIR}config.yaml" ]; then \
         cp "${MOD_DIR}config.yaml" /app/config.yaml; \
       else \
         : > /app/config.yaml; \
       fi

FROM alpine:3.23

EXPOSE 8080

RUN apk add --no-cache libc6-compat

WORKDIR /

COPY --from=build-env /server /
COPY --from=build-env /app/config.yaml /

CMD ["/server"]
