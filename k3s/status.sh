#!/usr/bin/env bash
set -Eeuo pipefail

NAMESPACE="${NAMESPACE:-enerflux}"
helm list --namespace "$NAMESPACE"
kubectl get pods,services,persistentvolumeclaims --namespace "$NAMESPACE" -o wide
