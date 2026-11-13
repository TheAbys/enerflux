#!/usr/bin/env bash
set -Eeuo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd -- "$SCRIPT_DIR/.." && pwd)"

IMAGE_PREFIX="${IMAGE_PREFIX:-ghcr.io/theabys}"
IMAGE_PREFIX="${IMAGE_PREFIX%/}"
IMAGE_TAG="${IMAGE_TAG:-develop}"
PLATFORMS="${PLATFORMS:-linux/amd64,linux/arm64}"

if ! command -v docker >/dev/null 2>&1; then
  echo "docker is required to build and push the EnerFlux images." >&2
  exit 1
fi

docker buildx build \
  --platform "$PLATFORMS" \
  --file "$PROJECT_ROOT/Dockerfile" \
  --target api \
  --tag "$IMAGE_PREFIX/enerflux-api:$IMAGE_TAG" \
  --push \
  "$PROJECT_ROOT"

docker buildx build \
  --platform "$PLATFORMS" \
  --file "$PROJECT_ROOT/Dockerfile" \
  --target agent \
  --tag "$IMAGE_PREFIX/enerflux-agent:$IMAGE_TAG" \
  --push \
  "$PROJECT_ROOT"

echo "Published $IMAGE_PREFIX/enerflux-api:$IMAGE_TAG and $IMAGE_PREFIX/enerflux-agent:$IMAGE_TAG"
