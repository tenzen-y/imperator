# Getting Started

## KIND Cluster Example 
Getting Started with KIND cluster.

### Prerequisites
- [KIND](https://kind.sigs.k8s.io/) >= v0.11
- [Docker](https://www.docker.com/) >=  v20.10
- [kubectl](https://kubectl.docs.kubernetes.io/installation/kubectl/) >= v1.20
- [kustomize](https://kubectl.docs.kubernetes.io/installation/kustomize/) >= v4.0.5

1. Create KIND Cluster and Deploy cert-manager.

```shell
$ make kind-start
```

2. Deploy imperator

```shell
$ kubectl apply -f https://raw.githubusercontent.com/tenzen-y/imperator/master/deploy/imperator.yaml
namespace/imperator-system created
customresourcedefinition.apiextensions.k8s.io/machinenodepools.imperator.tenzen-y.io configured
customresourcedefinition.apiextensions.k8s.io/machines.imperator.tenzen-y.io configured
serviceaccount/imperator-controller created
role.rbac.authorization.k8s.io/imperator-leader-election-role created
clusterrole.rbac.authorization.k8s.io/imperator-manager-role created
clusterrole.rbac.authorization.k8s.io/imperator-metrics-reader created
clusterrole.rbac.authorization.k8s.io/imperator-proxy-role created
rolebinding.rbac.authorization.k8s.io/imperator-leader-election-rolebinding created
clusterrolebinding.rbac.authorization.k8s.io/imperator-proxy-rolebinding created
clusterrolebinding.rbac.authorization.k8s.io/imperator-rolebinding created
service/imperator-metrics-service created
service/imperator-webhook-service created
deployment.apps/imperator-controller created
certificate.cert-manager.io/imperator-serving-cert created
issuer.cert-manager.io/imperator-selfsigned-issuer created
mutatingwebhookconfiguration.admissionregistration.k8s.io/imperator-mutating-webhook-configuration created
validatingwebhookconfiguration.admissionregistration.k8s.io/imperator-validating-webhook-configuration created
```

3. Wait for starting imperator-controller

```shell
$ kubectl wait pods -n imperator-system --for condition=ready --timeout=5m -l app.kubernetes.io/name=imperator
pod/imperator-controller-8559f57db8-qzbgt condition met
```

4. Deploy Sample Machine

```shell
$ kubectl apply -f https://raw.githubusercontent.com/tenzen-y/imperator/master/examples/machine/general-machine.yaml
$ kubectl get machines.imperator.tenzen-y.io,machinenodepools.imperator.tenzen-y.io
NAME                                            AGE
machine.imperator.tenzen-y.io/general-machine   2m34s

NAME                                                              READY   GROUP
machinenodepool.imperator.tenzen-y.io/general-machine-node-pool           
$ ###
$ # you will find the `imperator.tenzen-y.io/compute-small=general-machine` and `imperator.tenzen-y.io/node-pool=ready`. 
$ kubectl get nodes --show-labels kind-control-plane
NAME                 STATUS   ROLES                  AGE     VERSION   LABELS
kind-control-plane   Ready    control-plane,master   9m57s   v1.21.1   beta.kubernetes.io/arch=amd64,beta.kubernetes.io/os=linux,imperator.tenzen-y.io/compute-small=general-machine,imperator.tenzen-y.io/node-pool=ready,kubernetes.io/arch=amd64,kubernetes.io/hostname=kind-control-plane,kubernetes.io/os=linux,node-role.kubernetes.io/control-plane=,node-role.kubernetes.io/master=,node.kubernetes.io/exclude-from-external-load-balancers=
$ ###
$ # you will find a container to reserve resource for `compute-small`. 
$ kubectl get pods -n imperator-system
NAME                                   READY   STATUS    RESTARTS   AGE
general-machine-compute-small-0        1/1     Running   0          94s
imperator-controller-b6679f876-w4flg   2/2     Running   0          2m1s
$ kubectl describe machines.imperator.tenzen-y.io general-machine
Name:         general-machine
Namespace:    
Labels:       imperator.tenzen-y.io/machine-group=general-machine
Annotations:  <none>
API Version:  imperator.tenzen-y.io/v1alpha1
Kind:         Machine
Metadata:
  Creation Timestamp:  2021-12-23T19:24:50Z
  Generation:          1
  Managed Fields:
    API Version:  imperator.tenzen-y.io/v1alpha1
    Fields Type:  FieldsV1
    fieldsV1:
      f:status:
        .:
        f:availableMachines:
        f:condition:
    Manager:      imperator-controller
    Operation:    Update
    Time:         2021-12-23T19:24:50Z
    API Version:  imperator.tenzen-y.io/v1alpha1
    Fields Type:  FieldsV1
    fieldsV1:
      f:metadata:
        f:annotations:
          .:
          f:kubectl.kubernetes.io/last-applied-configuration:
        f:labels:
          .:
          f:imperator.tenzen-y.io/machine-group:
      f:spec:
        .:
        f:machineTypes:
        f:nodePool:
    Manager:         kubectl-client-side-apply
    Operation:       Update
    Time:            2021-12-23T19:24:50Z
  Resource Version:  1345
  UID:               99e87506-9add-42ab-8952-aceda3275022
Spec:
  Machine Types:
    Available:  1
    Name:       compute-small
    Spec:
      Cpu:     600m
      Memory:  1Gi
  Node Pool:
    Machine Type:
      Name:  compute-small
    Mode:    ready
    Name:    kind-control-plane
    Taint:   false
Status:
  Available Machines:
    Name:  compute-small
    Usage:
      Maximum:   1
      Reserved:  0
      Used:      0
      Waiting:   0
  Condition:
    Last Transition Time:  2021-12-23T19:24:50Z
    Message:               update status conditions
    Reason:                Success
    Status:                True
    Type:                  Ready
Events:
  Type    Reason   Age   From       Message
  ----    ------   ----  ----       -------
  Normal  Updated  0s    imperator  updated available machine status
```

5. Deploy sample namespace and deployment

```shell
$ kustomize build https://github.com/tenzen-y/imperator.git/examples/guest?=master | kubectl apply -f - 
$ ###
$ # you will find injected `resources.requests` and `resources.limits`. 
$ kubectl describe pods 
Name:         guest-deployment-bbd8c8754-hsqhk
Namespace:    guest-ns
Priority:     0
Node:         kind-control-plane/172.18.0.3
Start Time:   Thu, 23 Dec 2021 19:29:08 +0000
Labels:       imperator.tenzen-y.io/inject-resource=guest-container
              imperator.tenzen-y.io/machine-group=general-machine
              imperator.tenzen-y.io/machine-type=compute-small
              imperator.tenzen-y.io/pod-role=guest
              pod-template-hash=bbd8c8754
Annotations:  <none>
Status:       Running
IP:           10.244.0.7
IPs:
  IP:           10.244.0.7
Controlled By:  ReplicaSet/guest-deployment-bbd8c8754
Containers:
  guest-container:
    Container ID:  containerd://1602f3299e3a83014ac0a74deff1ee63a6f31549f4b68e320264fda5fc58f011
    Image:         alpine:3.15.0
    Image ID:      docker.io/library/alpine@sha256:21a3deaa0d32a8057914f36584b5288d2e5ecc984380bc0118285c70fa8c9300
    Port:          <none>
    Host Port:     <none>
    Command:
      sleep
    Args:
      300s
    State:          Running
      Started:      Thu, 23 Dec 2021 19:34:09 +0000
    Last State:     Terminated
      Reason:       Completed
      Exit Code:    0
      Started:      Thu, 23 Dec 2021 19:29:08 +0000
      Finished:     Thu, 23 Dec 2021 19:34:08 +0000
    Ready:          True
    Restart Count:  1
    Limits:
      cpu:     600m
      memory:  1Gi
    Requests:
      cpu:        600m
      memory:     1Gi
    Environment:  <none>
    Mounts:
      /var/run/secrets/kubernetes.io/serviceaccount from kube-api-access-xdbch (ro)
Conditions:
  Type              Status
  Initialized       True 
  Ready             True 
  ContainersReady   True 
  PodScheduled      True 
Volumes:
  kube-api-access-xdbch:
    Type:                    Projected (a volume that contains injected data from multiple sources)
    TokenExpirationSeconds:  3607
    ConfigMapName:           kube-root-ca.crt
    ConfigMapOptional:       <nil>
    DownwardAPI:             true
QoS Class:                   Guaranteed
Node-Selectors:              <none>
Tolerations:                 imperator.tenzen-y.io/compute-small=general-machine:NoSchedule
                             imperator.tenzen-y.io/node-pool=ready:NoSchedule
                             node.kubernetes.io/not-ready:NoExecute op=Exists for 300s
                             node.kubernetes.io/unreachable:NoExecute op=Exists for 300s
Events:
  Type    Reason     Age                  From               Message
  ----    ------     ----                 ----               -------
  Normal  Scheduled  5m54s                default-scheduler  Successfully assigned guest-ns/guest-deployment-bbd8c8754-hsqhk to kind-control-plane
  Normal  Pulled     53s (x2 over 5m54s)  kubelet            Container image "alpine:3.15.0" already present on machine
  Normal  Created    53s (x2 over 5m54s)  kubelet            Created container guest-container
  Normal  Started    53s (x2 over 5m54s)  kubelet            Started container guest-container
```

6. Tear Down

```shell
$ kind delete cluster
```
