package controllers

import (
	"context"
	"fmt"
	"strings"

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
	"github.com/tenzen-y/imperator/pkg/controllers/utils"
)

// MachineReconciler reconciles a Machine object
type MachineReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

//+kubebuilder:rbac:groups=imperator.tenzen-y.io,resources=machines,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=imperator.tenzen-y.io,resources=machines/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=imperator.tenzen-y.io,resources=machines/finalizers,verbs=update

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
		logger.Error(err, "unable to reconcile MachineNodePool", "name", machine.Name)
		r.Recorder.Eventf(machine, corev1.EventTypeWarning, "Failed", "Machine, %s: failed to reconcile MachineNodePool: %s", machine.Name, err.Error())
		meta.SetStatusCondition(&machine.Status.Conditions, metav1.Condition{
			Type:               imperatorv1alpha1.ConditionReady,
			Status:             metav1.ConditionFalse,
			LastTransitionTime: metav1.Now(),
			Reason:             metav1.StatusFailure,
			Message:            err.Error(),
		})
		return ctrl.Result{Requeue: true}, err
	}
	if err := r.reconcileStatefulSet(ctx, machine); err != nil {
		logger.Error(err, "unable to reconcile StateFulSet", "name", machine.Name)
		r.Recorder.Eventf(machine, corev1.EventTypeWarning, "Failed", "Machine, %s: failed to reconcile StatefulSet: %s", machine.Name, err.Error())
		meta.SetStatusCondition(&machine.Status.Conditions, metav1.Condition{
			Type:               imperatorv1alpha1.ConditionReady,
			Status:             metav1.ConditionFalse,
			LastTransitionTime: metav1.Now(),
			Reason:             metav1.StatusFailure,
			Message:            err.Error(),
		})
		return ctrl.Result{Requeue: true}, err
	}

	if err := r.reconcileService(ctx, machine); err != nil {
		logger.Error(err, "unable to reconcile Service", "name", machine.Name)
		r.Recorder.Eventf(machine, corev1.EventTypeWarning, "Failed", "Machine, %s: failed to reconcile Service: %s", machine.Name, err.Error())
		meta.SetStatusCondition(&machine.Status.Conditions, metav1.Condition{
			Type:               imperatorv1alpha1.ConditionReady,
			Status:             metav1.ConditionFalse,
			LastTransitionTime: metav1.Now(),
			Reason:             metav1.StatusFailure,
			Message:            err.Error(),
		})
		return ctrl.Result{Requeue: true}, err
	}

	return r.updateStatus(ctx, machine)
}

func (r *MachineReconciler) reconcileMachineNodePool(ctx context.Context, machine *imperatorv1alpha1.Machine) error {
	logger := log.FromContext(ctx)

	pool := &imperatorv1alpha1.MachineNodePool{
		TypeMeta: metav1.TypeMeta{
			Kind:       consts.KindMachine,
			APIVersion: imperatorv1alpha1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: strings.Join([]string{
				machine.Labels[consts.MachineGroupKey],
				"node-pool",
			}, "-"),
		},
	}

	origin := &imperatorv1alpha1.MachineNodePoolSpec{}
	opeResult, err := ctrl.CreateOrUpdate(ctx, r.Client, pool, func() error {
		origin = pool.Spec.DeepCopy()

		if pool.Labels == nil {
			pool.Labels = map[string]string{}
		}
		if _, exist := pool.Labels[consts.MachineGroupKey]; !exist {
			pool.Labels[consts.MachineGroupKey] = machine.Labels[consts.MachineGroupKey]
		}

		nodePoolSpec := machine.Spec.DeepCopy().NodePool
		pool.Spec.NodePool = nodePoolSpec
		return ctrl.SetControllerReference(machine, pool, r.Scheme)
	})
	if err != nil {
		return fmt.Errorf("failed to reconcile MachineNodePool; %v", err)
	}
	if opeResult != controllerutil.OperationResultNone {
		logger.Info("reconciled MachineNodePool", string(opeResult))
	}
	if opeResult == controllerutil.OperationResultUpdated {
		logger.Info(cmp.Diff(origin, pool))
	}

	return nil
}

func (r *MachineReconciler) reconcileStatefulSet(ctx context.Context, machine *imperatorv1alpha1.Machine) error {
	logger := log.FromContext(ctx)
	machineGroup := utils.GetMachineGroup(machine.Labels)

	for _, mt := range machine.Spec.MachineTypes {
		sts := &appsv1.StatefulSet{
			TypeMeta: metav1.TypeMeta{
				APIVersion: appsv1.SchemeGroupVersion.String(),
				Kind:       "StatefulSet",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      utils.GetReservationResourceName(machineGroup, mt.Name),
				Namespace: consts.ImperatorCoreNamespace,
			},
		}
		origin := &appsv1.StatefulSet{}

		opeResult, err := controllerutil.CreateOrUpdate(ctx, r.Client, sts, func() error {
			usage := utils.GetMachineTypeUsage(machine.Status.AvailableMachines, mt.Name)
			if usage == nil {
				logger.Info(fmt.Sprintf("skipped to reconcile StatefuleSet for machineType, %s", mt.Name))
				return nil
			}
			origin = sts
			stsReplica := usage.Ready - usage.Used
			sts = utils.GenerateStatefulSet(&mt, machineGroup, stsReplica)
			return controllerutil.SetOwnerReference(machine, sts, r.Scheme)
		})

		if err != nil {
			return fmt.Errorf("failed to reconcile StatefulSet for machineType, %s; %v", mt.Name, err)
		}
		if opeResult == controllerutil.OperationResultNone {
			logger.Info(fmt.Sprintf("reconciled StatefulSet for machineType, %s", mt.Name))
		}
		if opeResult == controllerutil.OperationResultUpdated {
			logger.Info(fmt.Sprintf("updated StatefulSet for machineType, %s", mt.Name))
			logger.Info(cmp.Diff(origin, sts))
		}
	}

	return nil
}

func (r *MachineReconciler) reconcileService(ctx context.Context, machine *imperatorv1alpha1.Machine) error {
	logger := log.FromContext(ctx)
	machineGroup := utils.GetMachineGroup(machine.Labels)

	for _, mt := range machine.Spec.MachineTypes {
		svc := &corev1.Service{
			TypeMeta: metav1.TypeMeta{
				APIVersion: corev1.SchemeGroupVersion.String(),
				Kind:       "Service",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      utils.GetReservationResourceName(machineGroup, mt.Name),
				Namespace: consts.ImperatorCoreNamespace,
			},
		}

		opeResult, err := controllerutil.CreateOrUpdate(ctx, r.Client, svc, func() error {
			usage := utils.GetMachineTypeUsage(machine.Status.AvailableMachines, mt.Name)
			if usage == nil {
				logger.Info(fmt.Sprintf("skipped to reconcile Service for machineType, %s", mt.Name))
				return nil
			}
			svc = utils.GenerateService(mt.Name, machineGroup)
			return controllerutil.SetOwnerReference(machine, svc, r.Scheme)
		})

		if err != nil {
			return fmt.Errorf("failed to reconcile Service for machineType, %s; %v", mt.Name, err)
		}
		if opeResult == controllerutil.OperationResultNone {
			logger.Info(fmt.Sprintf("reconciled Service for machineType, %s", mt.Name))
		}
		if opeResult == controllerutil.OperationResultUpdated {
			logger.Info(fmt.Sprintf("updated Service machineType, %s", mt.Name))
		}
	}
	return nil
}

func (r *MachineReconciler) updateStatus(ctx context.Context, machine *imperatorv1alpha1.Machine) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	machineGroup := utils.GetMachineGroup(machine.Labels)

	var desiredMachineTypeNum map[string]int32
	for _, mt := range machine.Spec.MachineTypes {
		desiredMachineTypeNum[mt.Name] = mt.Available
	}

	originAvailableMachineStatus := machine.Status.AvailableMachines
	for _, statusMT := range machine.Status.AvailableMachines {

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

		statusMT.Usage.Ready = 0
		for _, p := range reservationPods.Items {
			containerStatus := p.Status.ContainerStatuses[0].State
			if containerStatus.Running != nil {
				statusMT.Usage.Ready++
			}
			if containerStatus.Waiting != nil {
				if containerStatus.Waiting.Reason != "CrashLoopBackOff" {
					statusMT.Usage.Ready++
				}
			}
		}

		// looking for guest Pods
		guestPods := &corev1.PodList{}
		if err := r.List(ctx, guestPods, &client.ListOptions{
			LabelSelector: labels.SelectorFromSet(guestRoleLabels),
		}); err != nil {
			return ctrl.Result{}, err
		}

		statusMT.Usage.Used = 0
		for _, p := range guestPods.Items {
			containerStatus := p.Status.ContainerStatuses[0].State
			if containerStatus.Running != nil {
				statusMT.Usage.Used++
			}
			if containerStatus.Waiting != nil {
				if containerStatus.Waiting.Reason != "CrashLoopBackOff" {
					statusMT.Usage.Used++
				}
			}
		}

		// set Usage.Maximum
		statusMT.Usage.Maximum = desiredMachineTypeNum[statusMT.Name]
	}

	if !cmp.Equal(machine.Status.AvailableMachines, originAvailableMachineStatus) {
		r.Recorder.Eventf(machine, corev1.EventTypeNormal, "Updated", consts.KindMachine, machine.Name)
		meta.SetStatusCondition(&machine.Status.Conditions, metav1.Condition{
			Type:               imperatorv1alpha1.ConditionReady,
			Status:             metav1.ConditionTrue,
			LastTransitionTime: metav1.Now(),
			Reason:             metav1.StatusSuccess,
			Message:            "update status conditions",
		})
		if err := r.Status().Update(ctx, machine, &client.UpdateOptions{}); err != nil {
			return ctrl.Result{Requeue: true}, err
		}
		logger.Info(cmp.Diff(originAvailableMachineStatus, machine.Status.AvailableMachines))
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *MachineReconciler) SetupWithManager(mgr ctrl.Manager) error {
	podHandler := handler.EnqueueRequestsFromMapFunc(r.podReconcileTrigger)

	return ctrl.NewControllerManagedBy(mgr).
		For(&imperatorv1alpha1.Machine{}).
		Owns(&imperatorv1alpha1.MachineNodePool{}).
		Owns(&appsv1.StatefulSet{}).
		Watches(&source.Kind{Type: &corev1.Pod{}}, podHandler).
		Complete(r)
}

func (r *MachineReconciler) podReconcileTrigger(o client.Object) []reconcile.Request {
	return r.podReconcileRequest(o)
}

func (r *MachineReconciler) podReconcileRequest(o client.Object) []reconcile.Request {
	podLabels := o.GetLabels()
	machineGroupName, exist := podLabels[consts.MachineGroupKey]
	if !exist {
		return nil
	}

	if podRole, exist := podLabels[consts.PodRoleKey]; !exist || podRole != consts.PodRoleGuest {
		return nil
	}

	machines := &imperatorv1alpha1.MachineList{}
	if err := r.List(context.Background(), machines, &client.ListOptions{
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
