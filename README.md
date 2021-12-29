# Imperator

[![Go Tests](https://github.com/tenzen-y/imperator/actions/workflows/go-test.yaml/badge.svg?branch=master)](https://github.com/tenzen-y/imperator/actions/workflows/go-test.yaml?branch=master)

## Overview
Imperator is Kubernetes Custom Controller to provide virtual resource groups.

## Prerequisites
- [Kubernetes](https://kubernetes.io/) >= v1.20
- [cert-manager](https://cert-manager.io/) >= v1.0 
- [kustomize](https://kubectl.docs.kubernetes.io/installation/kustomize/) >= v4.0.5
- [NVIDIA/GPU feature discovery](https://github.com/NVIDIA/gpu-feature-discovery) >= v0.3.0
(optional: If you are using some NVIDIA GPUs on your Kubernetes Cluster, you must install this.)

## Getting Started
[Here](https://github.com/tenzen-y/imperator/tree/master/examples) you will find some examples.

## Contribution
Any contributions are welcome! Please take a look [CONTRIBUTING.md](./CONTRIBUTING.md) for developers.
