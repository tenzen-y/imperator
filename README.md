# Imperator

[![GitHub license](https://img.shields.io/github/license/tenzen-y/imperator)](https://github.com/tenzen-y/imperator/blob/master/LICENSE)
[![Go Tests](https://github.com/tenzen-y/imperator/actions/workflows/go-test.yaml/badge.svg?branch=master)](https://github.com/tenzen-y/imperator/actions/workflows/go-test.yaml?branch=master)
[![Codecov](https://codecov.io/gh/tenzen-y/imperator/branch/master/graph/badge.svg?token=34S3WJXV40)](https://codecov.io/gh/tenzen-y/imperator)
![GitHub release (latest SemVer)](https://img.shields.io/github/v/release/tenzen-y/imperator)

## Overview
Imperator is Kubernetes Operator to provide virtual resource groups.

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
