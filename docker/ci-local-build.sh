#!/bin/sh

set -eu

MODE="${1:-test}"
SECRETS_DIR="${2:-./_}"
CI_DOCKER_IMAGE="${CI_DOCKER_IMAGE:-docker:27-cli}"

SCRIPT_DIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
REPO_DIR="$(CDPATH= cd -- "${SCRIPT_DIR}/.." && pwd)"

DOCKER_REPO_DIR="$REPO_DIR"
if command -v cygpath >/dev/null 2>&1; then
  DOCKER_REPO_DIR="$(cygpath -w "$REPO_DIR")"
fi

MSYS_NO_PATHCONV=1 docker run --rm \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v "${DOCKER_REPO_DIR}:/workspace" \
  -w /workspace \
  -e IMAGE_NAME="${IMAGE_NAME-}" \
  -e IMAGE_REF="${IMAGE_REF-}" \
  -e TAG="${TAG-}" \
  -e LOGGING_SLACK_TOKEN="${LOGGING_SLACK_TOKEN-}" \
  -e LOGGING_SLACK_CHANNEL="${LOGGING_SLACK_CHANNEL-}" \
  -e CI_BUILD_LOG="${CI_BUILD_LOG-}" \
  -e PLATFORM="${PLATFORM-}" \
  -e RUN_IMAGE_TEST="${RUN_IMAGE_TEST-true}" \
  -e PROFILE="${PROFILE-}" \
  "$CI_DOCKER_IMAGE" \
  sh -lc 'apk add --no-cache ca-certificates python3 >/dev/null && python3 docker/ci-build.py "$@"' \
  -- "$MODE" "$SECRETS_DIR"
