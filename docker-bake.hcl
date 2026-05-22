# Atlas docker bake file. Drives all Go-service image builds.
#
# Canonical source of truth for the service list: .github/config/services.json
# (entries where type == "go-service"). The list below MUST mirror that file.
# A future CI check will assert parity; until then, treat the JSON as canonical
# and update both together.
#
# (We can't read services.json from HCL: docker buildx bake's HCL evaluator
# has no `file()` function and no `locals {}` block, so jsondecode-from-disk
# isn't available the way Terraform exposes it. Top-level identifiers are
# the supported pattern.)
#
#   docker buildx bake                                  # all-go-services (default group)
#   docker buildx bake all-go-services                  # explicit: every Go service
#   docker buildx bake atlas-account                    # one
#   docker buildx bake atlas-account atlas-ban          # subset
#
# CI overrides tags per-target via --set "<target>.tags=<image>:<tag>".

variable "ATLAS_IMAGE_TAG" {
  # Used by local builds (matches the deploy/compose ${ATLAS_IMAGE_TAG:-local} pattern).
  default = "local"
}

variable "GO_VERSION" {
  default = "1.25.5"
}

variable "ALPINE_VERSION" {
  default = "3.21"
}

# Mirror of .github/config/services.json .services[] | select(.type=="go-service") | .name.
# Keep alphabetized; keep in sync with services.json.
go_services = [
  "atlas-account",
  "atlas-asset-expiration",
  "atlas-ban",
  "atlas-buddies",
  "atlas-buffs",
  "atlas-cashshop",
  "atlas-chairs",
  "atlas-chalkboards",
  "atlas-channel",
  "atlas-character",
  "atlas-character-factory",
  "atlas-configurations",
  "atlas-consumables",
  "atlas-data",
  "atlas-drop-information",
  "atlas-drops",
  "atlas-effective-stats",
  "atlas-expressions",
  "atlas-fame",
  "atlas-families",
  "atlas-gachapons",
  "atlas-guilds",
  "atlas-inventory",
  "atlas-invites",
  "atlas-keys",
  "atlas-login",
  "atlas-map-actions",
  "atlas-maps",
  "atlas-marriages",
  "atlas-merchant",
  "atlas-messages",
  "atlas-messengers",
  "atlas-monster-book",
  "atlas-monster-death",
  "atlas-monsters",
  "atlas-notes",
  "atlas-npc-conversations",
  "atlas-npc-shops",
  "atlas-parties",
  "atlas-party-quests",
  "atlas-pets",
  "atlas-portal-actions",
  "atlas-portals",
  "atlas-query-aggregator",
  "atlas-quest",
  "atlas-rates",
  "atlas-reactor-actions",
  "atlas-reactors",
  "atlas-renders",
  "atlas-saga-orchestrator",
  "atlas-skills",
  "atlas-storage",
  "atlas-tenants",
  "atlas-transports",
  "atlas-world",
]

# One target per Go service, expanded from the list above at parse time.
target "go-service" {
  matrix = {
    svc = go_services
  }
  name       = svc
  context    = "."
  dockerfile = "Dockerfile"
  args = {
    SERVICE        = svc
    GO_VERSION     = "${GO_VERSION}"
    ALPINE_VERSION = "${ALPINE_VERSION}"
  }
  # Default local tag. CI overrides per-target via --set.
  tags = ["${svc}:${ATLAS_IMAGE_TAG}"]
}

# Non-Go services with their own Dockerfiles. cideps EnrichDockerServices
# includes any services.json entry with `docker_image` set regardless of
# type (see tools/cideps/config.go:116-141 + the
# TestEnrichDockerServices_IncludesNonGoServices guard), so these must be
# bake targets too — otherwise CI's per-shard `bake <names…>` errors with
# "could not find any target matching '<name>'" the first time a PR touches
# atlas-ui or atlas-pr-bootstrap.
target "atlas-ui" {
  context    = "."
  dockerfile = "services/atlas-ui/Dockerfile"
  tags       = ["atlas-ui:${ATLAS_IMAGE_TAG}"]
}

target "atlas-pr-bootstrap" {
  # Its Dockerfile uses relative COPYs (scripts/, canonical/), so the
  # context is the service directory rather than the repo root.
  context    = "services/atlas-pr-bootstrap"
  dockerfile = "Dockerfile"
  tags       = ["atlas-pr-bootstrap:${ATLAS_IMAGE_TAG}"]
}

group "all-go-services" {
  targets = go_services
}

group "all-services" {
  targets = concat(go_services, ["atlas-ui", "atlas-pr-bootstrap"])
}

# Default group: build everything (Go + non-Go) so a bare `docker buildx bake`
# matches the implicit "build all images" intent.
group "default" {
  targets = ["all-services"]
}
