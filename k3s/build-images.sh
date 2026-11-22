#!/usr/bin/env bash
set -Eeuo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd -- "$SCRIPT_DIR/.." && pwd)"

IMAGE_PREFIX="${IMAGE_PREFIX:-ghcr.io/theabys}"
IMAGE_PREFIX="${IMAGE_PREFIX%/}"
IMAGE_TAG="${IMAGE_TAG:-develop}"
PLATFORMS="${PLATFORMS:-linux/amd64,linux/arm64}"
BUILDER_NAME="${BUILDER_NAME:-enerflux-multiarch}"

if ! command -v docker >/dev/null 2>&1; then
  echo "docker is required to build and push the EnerFlux images." >&2
  exit 1
fi

if docker buildx inspect "$BUILDER_NAME" >/dev/null 2>&1; then
  builder_driver="$(docker buildx inspect "$BUILDER_NAME" | sed -n 's/^Driver: //p')"
  if [[ "$builder_driver" != "docker-container" ]]; then
    echo "Buildx builder '$BUILDER_NAME' uses '$builder_driver'; choose a docker-container builder with BUILDER_NAME." >&2
    exit 1
  fi
else
  docker buildx create --name "$BUILDER_NAME" --driver docker-container >/dev/null
fi

docker buildx inspect "$BUILDER_NAME" --bootstrap >/dev/null

docker buildx build \
  --builder "$BUILDER_NAME" \
  --pull \
  --platform "$PLATFORMS" \
  --file "$PROJECT_ROOT/Dockerfile" \
  --target api \
  --tag "$IMAGE_PREFIX/enerflux-api:$IMAGE_TAG" \
  --push \
  "$PROJECT_ROOT"

docker buildx build \
  --builder "$BUILDER_NAME" \
  --pull \
  --platform "$PLATFORMS" \
  --file "$PROJECT_ROOT/Dockerfile" \
  --target agent \
  --tag "$IMAGE_PREFIX/enerflux-agent:$IMAGE_TAG" \
  --push \
  "$PROJECT_ROOT"

echo "Published $IMAGE_PREFIX/enerflux-api:$IMAGE_TAG and $IMAGE_PREFIX/enerflux-agent:$IMAGE_TAG"
