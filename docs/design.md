# imperator v1alpha
design doceumtn for imperator v1beta1.

## Goal
Provide virtual resource group to applications.

## Overview

Imperator is Kubernetes Operator Pattern. This operator has two controller in the following list.

1. Machine Controller

2. MachineNodePool controller
   - This controller manage Kubernetes node to use target stateful applications.
   - This controller set virtual resource group name as node label to Kubernetes nodes defined into `nodePool` in CustomResource(CR).
   - if `nodePool.mode` is defined `maintenance` in CR, this controller is set maintenance=true as node label to Kubernetes nodes and
     maintenance to  in machine CR status.

TBD
