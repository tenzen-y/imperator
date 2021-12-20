#!/usr/bin/env bash
cd $(dirname $0)
set -o pipefail

kind create cluster --config ./kind-config.yaml
kubectl config use-context kind-kind

# Wait for Start Kind Node
TIMEOUT=5m
kubectl wait --for condition=ready --timeout=${TIMEOUT} node kind-control-plane
kubectl wait --for condition=ready --timeout=${TIMEOUT} node kind-worker
kubectl taint nodes --all node-role.kubernetes.io/master- || true
kubectl taint nodes --all  node-role.kubernetes.io/control-plane- || true

# Deploy CertManager
kubectl apply --validate=false -f https://github.com/jetstack/cert-manager/releases/latest/download/cert-manager.yaml
kubectl wait pods -n cert-manager --for condition=ready --timeout=${TIMEOUT} -l "app.kubernetes.io/name in (webhook,cainjector,cert-manager)"
