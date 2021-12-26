# Guide for developers

## Requirements

- [kubectl](https://kubectl.docs.kubernetes.io/installation/kubectl/) >= v1.20
- [kustomize](https://kubectl.docs.kubernetes.io/installation/kustomize/) >= v4.0.5
- [Docker](https://www.docker.com/) >=  v20.10
- [kubebuilder](https://github.com/kubernetes-sigs/kubebuilder) >= 3.0.0
- [Go](https://go.dev/) >= 1.17
- [KIND](https://kind.sigs.k8s.io/) >= v0.11

## Design documents

- [design_v1alpha1.md in English](./docs/design_v1alpha1.md)
- [design_v1alpha1_ja.md in Japanese](./docs/design_v1alpha1_ja.md)

## Build Container Image from source code

1. Verify formats and Generate codes in the following command.

```shell
$ make check
```

2. Run envtest in the following command.

```shell
$ make test
```

3. Build Container Image

```shell
$ make docker-build IMAGE_TAG_BASE=<IMAGE_NAME> VERSION=<IMAGE_TAG>
```

## Run integration test with KIND Cluster

1. Start KIND Cluster and deploy cert-manager in the following command.

```shell
$ make kind-start
```

2. Build image from source code, Load build image to KIND cluster and Start integration test in the following command.

```shell
$ make integration-test
```
