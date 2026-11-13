#!/usr/bin/env bash
set -Eeuo pipefail

NAMESPACE="${NAMESPACE:-enerflux}"
kubectl get pods,services,persistentvolumeclaims --namespace "$NAMESPACE" -o wide
