#!/usr/bin/env bash
set -euo pipefail


# -------------------------------
# Configuration
# -------------------------------
SERVICES_DIR="services"
IMAGE_PREFIX="monorepo-test"
CONTINUE_ON_FAILURE=false   # set to true to keep going
DOCKER_BUILD_ARGS=()        # e.g. ("--no-cache")


# -------------------------------
# Helpers
# -------------------------------

log() {
  echo -e "\n==> $1"

}


fail() {
  echo "ERROR: $1"
  if [[ "$CONTINUE_ON_FAILURE" == "true" ]]; then

    return 1
  else

    exit 1
  fi
}

# -------------------------------
# Preconditions
# -------------------------------
if ! command -v docker >/dev/null 2>&1; then
  echo "Docker is not installed or not in PATH"
  exit 1
fi

# -------------------------------
# Discover services with Dockerfiles
# -------------------------------
mapfile -t DOCKERFILES < <(
  find "${SERVICES_DIR}" -mindepth 2 -maxdepth 2 \
    -name Dockerfile \
    -type f \
    | sort
)

if [[ ${#DOCKERFILES[@]} -eq 0 ]]; then
  echo "No Dockerfiles found under ${SERVICES_DIR}"
  exit 0
fi

log "Found ${#DOCKERFILES[@]} service Dockerfiles"

# -------------------------------
# Build loop
# -------------------------------
FAILED=()


for dockerfile in "${DOCKERFILES[@]}"; do

  service_dir="$(dirname "${dockerfile}")"
  service_name="$(basename "${service_dir}")"
  image_tag="${IMAGE_PREFIX}/${service_name}:build-test"

  log "Building ${service_name}"
  echo "Dockerfile: ${dockerfile}"
  echo "Context:    ${service_dir}"
  echo "Image:      ${image_tag}"

  if ! docker build \
      -f "${dockerfile}" \
      -t "${image_tag}" \
      "${service_dir}" \
      "${DOCKER_BUILD_ARGS[@]}"; then
    FAILED+=("${service_name}")
    fail "Build failed for ${service_name}"
  fi
done


# -------------------------------
# Summary
# -------------------------------
if [[ ${#FAILED[@]} -gt 0 ]]; then
  echo
  echo "Build failures:"

  printf ' - %s\n' "${FAILED[@]}"
  exit 1
else
  echo
  echo "All service Dockerfiles built successfully"
fi

