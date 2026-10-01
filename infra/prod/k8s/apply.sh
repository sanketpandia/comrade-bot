#!/usr/bin/env bash
# Apply prod overlay (sources ConfigMaps from infra/prod/observability).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
OVERLAY="${ROOT}/overlays/prod"

build() {
  if command -v kustomize >/dev/null 2>&1; then
    kustomize build --load-restrictor LoadRestrictionsNone "$OVERLAY"
  else
    kubectl kustomize --load-restrictor=LoadRestrictionsNone "$OVERLAY"
  fi
}

build | kubectl apply -f -
echo "Applied kustomize overlay: ${OVERLAY}"
