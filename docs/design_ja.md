# imperator v1alpha
design document for imperator v1alpha1.

## Goal
Provide virtual resource group to applications.

## Overview

Imperator は Kubernetes Operator Pattern の controller で，2 つの controller が動作しています．

1. Machine Controller
    - Machine リソースで定義されたリソースで StatefulSet で reserved コンテナを作成する．
    - StatefulSet の命名規則は，<Machine Group>-<Machine Type>

2. MachineNodePool controller

Note: v1alpha1 では 1 つの Node を複数の Machine グループに参加させることはできない．

## controller の設計
Machine CR の作成後， MachineLearning CR 作成後までのシーケンス図は以下のようになっている．

![シーケンス図](./v1alpha1-sequence.png)

### Machine controller

machine の数量管理では，Pod リソースを監視する．
- Pod: label に `imperator.io/machine` がついている物をスクレイプする．
  また，スクレイプしてきた中で 以下の条件に合致する物を稼働中としてカウントする．
    - Running: `.status.containerStatuses.state.Running` が nil ではない場合．
    - ContainerCreating: `.status.containerStatuses.state.waiting` が nil ではないかつ，
      `.status.containerStatuses.state.waiting.reason` が `Error` ではない場合．
    - Terminating: `.metadata.deletionTimestamp` が nil ではない場合．

### NodePool controller
- Node の Annotation　に `imperator.io/machine-group=グループ名` を付与する．
- nodePool の mode が ready のノードに `imperator/nodePool=ready` のラベルをつける．
  nodePool に無いノードもしくは， mode が `ready` ではなくなったノードや status が `not-ready` では無くなったノードからはラベルを削除する．
- status の nodePool 欄 condition は，定期的に node を監視し，健康状態に応じて変更する．

## Custom Resource Schema

### Machine リソース

- spec.machineTypes[*].spec.hostLimit は，対象のマシン 1 つでホストリソースの何割まで消費することができるかの制限をつける．
- spec.machineTypes[*].spec.dependence は，親リソースを指定し，その親リソースの何割のリソースを使用するかを .availableRatio に記述する．

```yaml
---
apiVersion: imperator.tenzen-y.io/v1alpha1
kind: Machine
metadata:
  name: general-machine
  labels:
    imperator.io/machine-group: general-machine
spec:
  nodePool:
    - name: michiru
      mode: ready
      assignmentType: taint # omitempty;default=label
      machineType:
        name: compute-xlarge
        scheduleChildren: true # omitempty;default=false
    - name: utaha
      mode: maintenance
      assignmentType: label # omitempty;default=label
      machineType:
        name: compute-medium
        scheduleChildren: false # omitempty;default=false
    - name: eriri
      mode: ready
      assignmentType: label # omitempty;default=label
      machineType:
        name: compute-medium
        scheduleChildren: false # omitempty;default=false
  machineTypes:
    - name: compute-medium
      spec:
        cpu: 6000m
        memory: 48Gi
        gpu: #omitempty
          type: nvidia.com/gpu
          num: 1
          generation: turing
      available: 4
    - name: compute-xlarge
      spec:
        cpu: 40000m
        memory: 128Gi
        gpu: #omitempty
          type: nvidia.com/gpu
          num: 2
          generation: ampere
      available: 1
    - name: compute-large
      spec:
        cpu: 20000m
        memory: 64Gi
        gpu: #omitempty
          type: nvidia.com/gpu
          num: 1
          generation: ampere
      dependence: #omitempty
        parent: vram-large1
        availableRatio: 0.5
      available: 2
status:
  condition:
    - lastTransitionTime: "2021-07-24T09:08:39Z"
      status: "True"
      type: Ready
  availableMachines:
    - name: compute-medium
      usage:
        maximum: 4
        used: 1
    - name: compute-xlarge
      usage:
        maximum: 1
        used: 0.5
    - name: compute-large
      usage:
        maximum: 2
        used: 1
```

### MachineNodePool リソース

- .metadata.name は ownerReference を参照し，`.metadata.name-node-pool` にする．
- spec　は machine リソースから持ってくる．

```yaml
---
apiVersion: imperator.io/v1alpha1
kind: MachineNodePool
metadata:
  name: general-machine-node-pool
  labels:
    imperator.tenzen-y.io/machine-group: general-machine
spec:
  machineGroup: general-machine
  nodePool:
    - name: michiru
      mode: ready
      assignmentType: taint # omitempty;default=label
      machineType:
        name: compute-xlarge
        scheduleChildren: true # omitempty;default=false
    - name: utaha
      mode: maintenance
      assignmentType: label # omitempty;default=label
      machineType:
        name: compute-medium
        scheduleChildren: false # omitempty;default=false
    - name: eriri
      mode: ready
      assignmentType: label # omitempty;default=label
      machineType:
        name: compute-medium
        scheduleChildren: false # omitempty;default=false
  machineTypeStock:
    - name: compute-xlarge
    - name: compute-large
      parent: compute-xlarge # omitempty;*string
    - name: compute-medium
status:
  condition:
    - lastTransitionTime: "2021-07-24T09:08:39Z"
      status: "True"
      type: Ready
  nodePool:
    - name: michiru
      condition: Ready
    - name: utaha
      condition: Maintenance
    - name: eriri
      condition: NotReady
```
