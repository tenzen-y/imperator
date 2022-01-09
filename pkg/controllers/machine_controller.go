/*
Copyright 2021 Yuki Iwai (@tenzen-y)

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"

	"github.com/google/go-cmp/cmp"
	"github.com/imdario/mergo"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	imperatorv1alpha1 "github.com/tenzen-y/imperator/pkg/api/v1alpha1"
	"github.com/tenzen-y/imperator/pkg/consts"
	"github.com/tenzen-y/imperator/pkg/controllers/util"
)

// MachineReconciler reconciles a Machine object
type MachineReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=imperator.tenzen-y.io,resources=machines,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=imperator.tenzen-y.io,resources=machines/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=imperator.tenzen-y.io,resources=machines/finalizers,verbs=update
// +kubebuilder:rbac:groups=imperator.tenzen-y.io,resources=machinenodepools,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods/status,verbs=get;watch
// +kubebuilder:rbac:groups=core,resources=namespaces,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Machine object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.9.2/pkg/reconcile
func (r *MachineReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	machine := &imperatorv1alpha1.Machine{}
	logger.Info("fetching Machine Resource")

	err := r.Get(ctx, req.NamespacedName, machine)
	if errors.IsNotFound(err) {
		return ctrl.Result{}, nil
	}

	return r.reconcile(ctx, machine)
}

func (r *MachineReconciler) reconcile(ctx context.Context, machine *imperatorv1alpha1.Machine) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if err := r.reconcileMachineNodePool(ctx, machine); err != nil {
		logger.Error(err, "failed to reconcile MachineNodePool", "name", machine.Name)
		return r.updateReconcileFailedStatus(ctx, machine, err)
	}
	if err := r.reconcileStatefulSet(ctx, machine); err != nil {
		logger.Error(err, "failed to reconcile StatefulSet", "name", machine.Name)
		return r.updateReconcileFailedStatus(ctx, machine, err)
	}

	if err := r.reconcileService(ctx, machine); err != nil {
		logger.Error(err, "failed to reconcile Service", "name", machine.Name)
		return r.updateReconcileFailedStatus(ctx, machine, err)
	}

	return r.updateStatus(ctx, machine)
}

func (r *MachineReconciler) updateReconcileFailedStatus(ctx context.Context, machine *imperatorv1alpha1.Machine, reconcileErr error) (ctrl.Result, error) {
	meta.SetStatusCondition(&machine.Status.Conditions, metav1.Condition{
		Type:               imperatorv1alpha1.ConditionReady,
		Status:             metav1.ConditionFalse,
		LastTransitionTime: metav1.Now(),
		Reason:             metav1.StatusFailure,
		Message:            reconcileErr.Error(),
	})
	if err := r.Status().Update(ctx, machine, &client.UpdateOptions{}); err != nil {
		return ctrl.Result{Requeue: true}, err
	}
	return ctrl.Result{}, nil
}

func (r *MachineReconciler) reconcileMachineNodePool(ctx context.Context, machine *imperatorv1alpha1.Machine) error {
	logger := log.FromContext(ctx)
	machineGroup := util.GetMachineGroup(machine.Labels)

	pool := &imperatorv1alpha1.MachineNodePool{
		TypeMeta: metav1.TypeMeta{
			Kind:       consts.KindMachineNodePool,
			APIVersion: imperatorv1alpha1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: util.GenerateMachineNodePoolName(machineGroup),
		},
	}

	origin := &imperatorv1alpha1.MachineNodePool{}
	opeResult, err := ctrl.CreateOrUpdate(ctx, r.Client, pool, func() error {
		origin = pool.DeepCopy()

		poolMachineTypeStockMap := make(map[string]bool)
		for _, mts := range pool.Spec.MachineTypeStock {
			if poolMachineTypeStockMap[mts.Name] {
				continue
			}
			poolMachineTypeStockMap[mts.Name] = true
		}

		if pool.Labels == nil {
			pool.Labels = make(map[string]string)
		}
		if _, exist := pool.Labels[consts.MachineGroupKey]; !exist {
			pool.Labels[consts.MachineGroupKey] = machineGroup
		}
		pool.Spec.MachineGroupName = machineGroup

		nodePoolSpec := machine.Spec.DeepCopy().NodePool
		pool.Spec.NodePool = nodePoolSpec
		for _, mt := range machine.Spec.MachineTypes {
			if poolMachineTypeStockMap[mt.Name] {
				continue
			}
			pool.Spec.MachineTypeStock = append(pool.Spec.MachineTypeStock, imperatorv1alpha1.NodePoolMachineTypeStock{
				Name: mt.Name,
			})
		}

		return ctrl.SetControllerReference(machine, pool, r.Scheme)
	})
	if err != nil {
		return fmt.Errorf("failed to reconcile MachineNodePool; %v", err)
	}
	if opeResult != controllerutil.OperationResultNone {
		logger.Info(fmt.Sprintf("reconciled MachineNodePool; %v", string(opeResult)))
	}
	if opeResult == controllerutil.OperationResultCreated {
		return nil
	}
	if opeResult == controllerutil.OperationResultUpdated {
		logger.Info(cmp.Diff(origin.Spec, pool.Spec, consts.CmpSliceOpts...))
	}

	return nil
}

func (r *MachineReconciler) reconcileStatefulSet(ctx context.Context, machine *imperatorv1alpha1.Machine) error {
	logger := log.FromContext(ctx)
	machineGroup := util.GetMachineGroup(machine.Labels)

	nodePoolMachineTypeMap := make(map[string]bool)
	for _, np := range machine.Spec.NodePool {
		for _, npmt := range np.MachineType {
			if nodePoolMachineTypeMap[npmt.Name] {
				continue
			}
			nodePoolMachineTypeMap[npmt.Name] = true
		}
	}

	for _, mt := range machine.Spec.MachineTypes {
		if !nodePoolMachineTypeMap[mt.Name] {
			continue
		}
		sts := &appsv1.StatefulSet{
			TypeMeta: metav1.TypeMeta{
				APIVersion: appsv1.SchemeGroupVersion.String(),
				Kind:       "StatefulSet",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      util.GenerateReservationResourceName(machineGroup, mt.Name),
				Namespace: consts.ImperatorCoreNamespace,
			},
		}
		origin := &appsv1.StatefulSet{}
		usage := util.GetMachineTypeUsage(machine.Status.AvailableMachines, mt.Name)
		if usage == nil {
			logger.Info(fmt.Sprintf("skipped to reconcile StatefuleSet for machineType, %s", mt.Name))
			continue
		}
		opeResult, err := ctrl.CreateOrUpdate(ctx, r.Client, sts, func() error {
			origin = sts.DeepCopy()
			stsReplica := usage.Reserved - (usage.Used + usage.Waiting)
			if stsReplica < 0 {
				stsReplica = 0
			} else if stsReplica == 0 && usage.Reserved == 0 {
				stsReplica = usage.Maximum
			}
			util.GenerateStatefulSet(&mt, machineGroup, stsReplica, sts)
			return ctrl.SetControllerReference(machine, sts, r.Scheme)
		})

		if err != nil {
			return fmt.Errorf("failed to reconcile StatefulSet for machineType, %s; %v", mt.Name, err)
		}
		if opeResult == controllerutil.OperationResultNone {
			logger.Info(fmt.Sprintf("reconciled StatefulSet for machineType, %s", mt.Name))
		}
		if opeResult == controllerutil.OperationResultCreated {
			continue
		}
		if opeResult == controllerutil.OperationResultUpdated {
			logger.Info(fmt.Sprintf("updated StatefulSet for machineType, %s", mt.Name))
			logger.Info(cmp.Diff(origin.Spec, sts.Spec, consts.CmpStatefulSetOpts...))
		}
	}

	return nil
}

func (r *MachineReconciler) reconcileService(ctx context.Context, machine *imperatorv1alpha1.Machine) error {
	logger := log.FromContext(ctx)
	machineGroup := util.GetMachineGroup(machine.Labels)

	for _, mt := range machine.Spec.MachineTypes {
		svc := &corev1.Service{
			TypeMeta: metav1.TypeMeta{
				APIVersion: corev1.SchemeGroupVersion.String(),
				Kind:       "Service",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      util.GenerateReservationResourceName(machineGroup, mt.Name),
				Namespace: consts.ImperatorCoreNamespace,
			},
		}

		usage := util.GetMachineTypeUsage(machine.Status.AvailableMachines, mt.Name)
		if usage == nil {
			logger.Info(fmt.Sprintf("skipped to reconcile Service for machineType, %s", mt.Name))
			continue
		}
		opeResult, err := ctrl.CreateOrUpdate(ctx, r.Client, svc, func() error {
			util.GenerateService(mt.Name, machineGroup, svc)
			return ctrl.SetControllerReference(machine, svc, r.Scheme)
		})

		if err != nil {
			return fmt.Errorf("failed to reconcile Service for machineType, %s; %v", mt.Name, err)
		}
		if opeResult == controllerutil.OperationResultNone {
			logger.Info(fmt.Sprintf("reconciled Service for machineType, %s", mt.Name))
		}
		if opeResult == controllerutil.OperationResultCreated {
			continue
		}
		if opeResult == controllerutil.OperationResultUpdated {
			logger.Info(fmt.Sprintf("machineType: %s; updated Service", mt.Name))
		}
	}
	return nil
}

func (r *MachineReconciler) updateStatus(ctx context.Context, machine *imperatorv1alpha1.Machine) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	machineGroup := util.GetMachineGroup(machine.Labels)

	desiredMachineTypeNum := make(map[string]int32)
	for _, mt := range machine.Spec.MachineTypes {
		desiredMachineTypeNum[mt.Name] = mt.Available
	}

	originAvailableMachineStatus := machine.Status.DeepCopy().AvailableMachines
	originConditions := machine.Status.DeepCopy().Conditions

	// if availableMachines is empty, create that
	availableMachinesMap := make(map[string]bool)
	for _, am := range machine.Status.AvailableMachines {
		availableMachinesMap[am.Name] = true
	}
	for _, mt := range machine.Spec.MachineTypes {
		if availableMachinesMap[mt.Name] {
			continue
		}
		machine.Status.AvailableMachines = append(machine.Status.AvailableMachines, imperatorv1alpha1.AvailableMachineCondition{
			Name: mt.Name,
			Usage: imperatorv1alpha1.UsageCondition{
				Maximum:  mt.Available,
				Reserved: 0,
				Used:     0,
				Waiting:  0,
			},
		})
	}

	for idx, statusMT := range machine.Status.AvailableMachines {

		// prepare labels
		roleCommonLabels := map[string]string{
			consts.MachineGroupKey: machineGroup,
			consts.MachineTypeKey:  statusMT.Name,
		}
		reservationRoleLabels := map[string]string{
			consts.PodRoleKey: consts.PodRoleReservation,
		}
		guestRoleLabels := map[string]string{
			consts.PodRoleKey: consts.PodRoleGuest,
		}

		if err := mergo.Map(&reservationRoleLabels, roleCommonLabels, mergo.WithOverride); err != nil {
			return ctrl.Result{}, err
		}
		if err := mergo.Map(&guestRoleLabels, roleCommonLabels, mergo.WithOverride); err != nil {
			return ctrl.Result{}, err
		}

		// looking for Pods to reserve resource
		reservationPods := &corev1.PodList{}
		if err := r.List(ctx, reservationPods, &client.ListOptions{
			LabelSelector: labels.SelectorFromSet(reservationRoleLabels),
		}); err != nil {
			return ctrl.Result{}, err
		}

		machine.Status.AvailableMachines[idx].Usage.Reserved = 0
		for _, po := range reservationPods.Items {
			podConditionTypeMap := util.GetPodConditionTypeMap(po.Status.Conditions)
			// Terminating
			if po.ObjectMeta.DeletionTimestamp != nil {
				continue
			}
			// Running
			if po.Status.Phase == corev1.PodRunning && podConditionTypeMap[corev1.ContainersReady].Status == corev1.ConditionTrue {
				machine.Status.AvailableMachines[idx].Usage.Reserved++
				// ContainerCreating
			} else if po.Status.Phase == corev1.PodPending && po.Spec.NodeName != "" {
				machine.Status.AvailableMachines[idx].Usage.Reserved++
			}
		}

		// looking for guest Pods
		guestPods := &corev1.PodList{}
		if err := r.List(ctx, guestPods, &client.ListOptions{
			LabelSelector: labels.SelectorFromSet(guestRoleLabels),
		}); err != nil {
			return ctrl.Result{}, err
		}

		machine.Status.AvailableMachines[idx].Usage.Used = 0
		machine.Status.AvailableMachines[idx].Usage.Waiting = 0
		for _, po := range guestPods.Items {

			ns := &corev1.Namespace{}
			if err := r.Get(ctx, client.ObjectKey{Name: po.Namespace}, ns); err != nil {
				return ctrl.Result{}, err
			}
			if injectionLabel := ns.Labels[consts.ImperatorResourceInjectionKey]; injectionLabel != consts.ImperatorResourceInjectionEnabled {
				continue
			}

			// Terminating
			if !po.ObjectMeta.DeletionTimestamp.IsZero() {
				continue
			}

			podConditionTypeMap := util.GetPodConditionTypeMap(po.Status.Conditions)

			// Running
			if po.Status.Phase == corev1.PodRunning && podConditionTypeMap[corev1.ContainersReady].Status == corev1.ConditionTrue {
				machine.Status.AvailableMachines[idx].Usage.Used++
			} else if po.Status.Phase == corev1.PodPending {

				// ContainerCreating
				if po.Spec.NodeName != "" {
					machine.Status.AvailableMachines[idx].Usage.Used++
				} else if _, exist := podConditionTypeMap[corev1.PodScheduled]; exist {
					scheduledCondition := podConditionTypeMap[corev1.PodScheduled]

					// Pod has not yet been scheduled on any Nodes
					if scheduledCondition.Reason == corev1.PodReasonUnschedulable &&
						scheduledCondition.Status == corev1.ConditionFalse {
						machine.Status.AvailableMachines[idx].Usage.Waiting++
					}
				}
			}
		}

		// set Usage.Maximum
		machine.Status.AvailableMachines[idx].Usage.Maximum = desiredMachineTypeNum[statusMT.Name]
	}

	meta.SetStatusCondition(&machine.Status.Conditions, metav1.Condition{
		Type:               imperatorv1alpha1.ConditionReady,
		Status:             metav1.ConditionTrue,
		LastTransitionTime: metav1.Now(),
		Reason:             metav1.StatusSuccess,
		Message:            "update status conditions",
	})

	conditionDiff := cmp.Diff(originConditions, machine.Status.Conditions, consts.CmpSliceOpts...)
	diff := cmp.Diff(originAvailableMachineStatus, machine.Status.AvailableMachines, consts.CmpSliceOpts...)
	if diff != "" || conditionDiff != "" {
		r.Recorder.Eventf(machine, corev1.EventTypeNormal, "Updated", "updated available machine status")
		if err := r.Status().Update(ctx, machine, &client.UpdateOptions{}); err != nil {
			return ctrl.Result{Requeue: true}, err
		}
		logger.Info(diff)
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MachineReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager) error {
	podHandler := handler.EnqueueRequestsFromMapFunc(func(o client.Object) []reconcile.Request {
		return r.podReconcileRequest(ctx, o)
	})

	return ctrl.NewControllerManagedBy(mgr).
		For(&imperatorv1alpha1.Machine{}).
		Owns(&imperatorv1alpha1.MachineNodePool{}).
		Owns(&appsv1.StatefulSet{}).
		Watches(&source.Kind{Type: &corev1.Pod{}}, podHandler).
		Complete(r)
}

func (r *MachineReconciler) podReconcileRequest(ctx context.Context, o client.Object) []reconcile.Request {
	// check namespace & PodRole
	ns := &corev1.Namespace{}
	if err := r.Get(ctx, client.ObjectKey{Name: o.GetNamespace()}, ns); err != nil {
		return nil
	}
	injectionLabel := ns.Labels[consts.ImperatorResourceInjectionKey]

	podLabels := o.GetLabels()
	podRole := podLabels[consts.PodRoleKey]
	if injectionLabel != consts.ImperatorResourceInjectionEnabled && podRole == consts.PodRoleGuest {
		return nil
	}

	// check MachineGroup
	machineGroupName, exist := podLabels[consts.MachineGroupKey]
	if !exist {
		return nil
	}

	machines := &imperatorv1alpha1.MachineList{}
	if err := r.List(ctx, machines, &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(map[string]string{
			consts.MachineGroupKey: machineGroupName,
		}),
	}); err != nil {
		return nil
	}
	if len(machines.Items) != 1 {
		return nil
	}

	machineType, exist := podLabels[consts.MachineTypeKey]
	if !exist {
		return nil
	}

	for _, mt := range machines.Items[0].Spec.MachineTypes {
		if mt.Name == machineType {
			return []reconcile.Request{{
				NamespacedName: client.ObjectKeyFromObject(&machines.Items[0]),
			}}
		}
	}

	return nil
}
