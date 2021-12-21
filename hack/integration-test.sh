#!/usr/bin/env bash
cd "$(dirname "$0")"
set -eox pipefail

CLUSTER_TYPE=${1:-KinD}
IMPERATOR_CORE_NAMESPACE=imperator-system
GUEST_NAMESPACE=guest-ns
GUEST_POD_LABELS="imperator.tenzen-y.io/machine-group=general-machine,imperator.tenzen-y.io/machine-type=compute-small,imperator.tenzen-y.io/pod-role=guest,imperator.tenzen-y.io/inject-resource=guest-container"
TIMEOUT=5m

function setup() {
  # Setup impelator
  echo "Setup imperator"

  yq eval -i '.spec.template.spec.containers[0].imagePullPolicy|="IfNotPresent"' ../config/manager/manager.yaml
  kustomize build ../config/crd | kubectl apply -f -
  kustomize build ../config/default | kubectl apply -f -
  kubectl wait pods -n ${IMPERATOR_CORE_NAMESPACE} --for condition=ready --timeout=${TIMEOUT} -l app.kubernetes.io/name=imperator
  kubectl get pods -n ${IMPERATOR_CORE_NAMESPACE}
  sleep 5

  # Deploy Machine
  echo "Deploy Machine"
  if [ "${CLUSTER_TYPE}" = 'minikube' ]; then \
    yq eval -i '.spec.nodePool[0].name|="minikube"' ../examples/general-machine.yaml
  fi;
  kubectl apply -f ../examples/general-machine.yaml
  kubectl get machines
  kubectl get pods -n ${IMPERATOR_CORE_NAMESPACE}
  kubectl describe machines general-machine
}

function _deploy_guest_deployment() {
  injection=$1
  echo "Deploy Guest Deployment"
  if ! $injection; then \
    yq eval -i 'del(.metadata.labels)' ../examples/guest/namespace.yaml
  fi;
  kustomize build ../examples/guest | kubectl apply -f -

  count=0
  WAIT_LIMIT=5
  while [ ${count} -lt ${WAIT_LIMIT} ]; do
    POD_NUM=$(kubectl get pods -n ${GUEST_NAMESPACE} 2>/dev/null | wc -l)
    if [ "${POD_NUM}" = "0" ]; then \
      count=$(( "${count}" + 1 ));
      sleep 2;
    else \
      count=5;
    fi;
  done;
  
  kubectl wait pods -n ${GUEST_NAMESPACE} --for condition=ready --timeout ${TIMEOUT} -l ${GUEST_POD_LABELS}
  kubectl get pods -n ${GUEST_NAMESPACE}
  kubectl describe machines general-machine
}

function _get_pod_yaml() {
  POD_NAME="$(kubectl get pods -n ${GUEST_NAMESPACE} -l ${GUEST_POD_LABELS} -o name | cut -d/ -f2)"
  kubectl get pods -n ${GUEST_NAMESPACE} "${POD_NAME}" -o yaml
}

function _get_actual_resources() {
  resource_type=$1
  if [ "$resource_type" = "cpu" ]; then \
    _get_pod_yaml | yq eval '.spec.containers[0].resources.requests.cpu' -;
  elif [ "$resource_type" = "memory" ]; then \
    _get_pod_yaml | yq eval '.spec.containers[0].resources.requests.memory' -;
  else \
    echo "can not get actual resource, <$resource_type>"
    exit 1
  fi;
}

function _get_desired_resources() {
  resource_type=$1
  if [ "$resource_type" = "cpu" ]; then \
    yq eval '.spec.machineTypes[0].spec.cpu' ../examples/general-machine.yaml;
  elif [ "$resource_type" = "memory" ]; then \
    yq eval '.spec.machineTypes[0].spec.memory' ../examples/general-machine.yaml;
  else \
    echo "can not get desired resources, <$resource_type>"
    exit 1
  fi;
}

function _teardown() {
  kustomize build ../examples/guest | kubectl delete -f -
  kubectl wait pods -n ${GUEST_NAMESPACE} -l ${GUEST_POD_LABELS} --for=delete --timeout ${TIMEOUT}
  git reset --hard
}

function integration_test() {
  injection=$1
  _deploy_guest_deployment "$injection"

  desired_cpu=$(_get_desired_resources cpu)
  desired_memory=$(_get_desired_resources memory)
  if [ -z "${desired_cpu}" ]; then \
    echo "desired cpu is empty."
    exit 1
  elif [ -z "${desired_memory}" ]; then \
    echo "desired memory is empty."
    exit 1
  fi

  actual_cpu=$(_get_actual_resources cpu)
  actual_memory=$(_get_actual_resources memory)
  if [ -z "${actual_cpu}" ]; then \
    echo "actual cpu is empty."
    exit 1
  elif [ -z "${actual_memory}" ]; then \
    echo "actual memory is empty."
    exit 1
  fi

  if $injection; then \
    echo "imperator inject resources, affinity, and toleration to Pod"
    if [ ! "${desired_cpu}" = "${actual_cpu}" ]; then \
      echo "desired cpu: <$desired_cpu> and actual cpu: <$actual_cpu> are different.";
      exit 1;
    elif [ ! "${desired_memory}" = "${actual_memory}" ]; then \
      echo "desired memory: <$desired_cpu> and actual memory: <$actual_cpu> are different.";
      exit 1;
    fi;
  else
    echo "imperator does not inject resources, affinity, and toleration to Pod"
    if [ ! "${actual_cpu}" = "null" ]; then \
      echo "imperator injected cpu; <$actual_cpu>";
      exit 1;
    elif [ ! "${actual_memory}" = "null" ]; then \
      echo "imperator injected memory; <$actual_memory>";
      exit 1;
    fi;
  fi;

  _teardown
}

# setup testenv
setup

# imperator inject resources, affinity, and toleration to Pod
integration_test true

# imperator does not inject resources, affinity, and toleration to Pod
integration_test false
