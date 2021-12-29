#!/usr/bin/env bash

# Copyright 2021 Yuki Iwai (@tenzen-y)
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -eo pipefail
cd "$(dirname "$0")"

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
