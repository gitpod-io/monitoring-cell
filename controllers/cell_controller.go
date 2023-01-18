/*
Copyright 2022.

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
	"time"

	"github.com/go-logr/logr"

	monitoringv1alpha1 "github.com/gitpod-io/monitoring-cell/api/v1alpha1"
	"github.com/gitpod-io/monitoring-cell/pkg/components/gitpod"
	kubernetes "github.com/gitpod-io/monitoring-cell/pkg/components/kubernetes"
	kubestatemetrics "github.com/gitpod-io/monitoring-cell/pkg/components/kubestate-metrics"
	nodeexporter "github.com/gitpod-io/monitoring-cell/pkg/components/node-exporter"
	"github.com/gitpod-io/monitoring-cell/pkg/components/prometheus"
	prometheusoperator "github.com/gitpod-io/monitoring-cell/pkg/components/prometheus-operator"
	pomonitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	POOwnerKey = ".metadata.controller"
	apiGVStr   = monitoringv1alpha1.GroupVersion.String()
)

// CellReconciler reconciles a Cell object
type CellReconciler struct {
	client.Client
	PodRESTClient rest.Interface
	RESTConfig    *rest.Config
	Scheme        *runtime.Scheme
	Logger        logr.Logger
}

//+kubebuilder:rbac:groups=monitoring.gitpod.io,resources=cells,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=monitoring.gitpod.io,resources=cells/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=monitoring.gitpod.io,resources=cells/finalizers,verbs=update
//+kubebuilder:rbac:groups=,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=,resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterroles,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterrolebindings,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=monitoring.coreos.com,resources=prometheuses,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=monitoring.coreos.com,resources=prometheusrules,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=monitoring.coreos.com,resources=servicemonitors,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.1/pkg/reconcile
func (r *CellReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.Logger = log.FromContext(ctx)
	var cell monitoringv1alpha1.Cell

	if err := r.Get(ctx, req.NamespacedName, &cell); err != nil {
		r.Logger.Error(err, "Unable to fetch Cell")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Reconcile Prometheus Operator -- Continue
	if err := r.reconcilePrometheusOperator(ctx, &cell, req); err != nil {
		r.Logger.Error(err, "Unable to reconcile Prometheus Operator")
		return ctrl.Result{}, err
	}
	// Reconcile Prometheus -- Continue
	if err := r.reconcilePrometheus(ctx, &cell, req); err != nil {
		r.Logger.Error(err, "Unable to reconcile Prometheus")
		return ctrl.Result{}, err
	}
	// Reconcile Exporters -- Continue
	if err := r.reconcileExporters(ctx, &cell, req); err != nil {
		r.Logger.Error(err, "Unable to reconcile Exporters")
		return ctrl.Result{}, err
	}
	// Reconcile Gitpod servicemonitors -- Continue
	if err := r.reconcileGitpodServiceMonitors(ctx, &cell, req); err != nil {
		r.Logger.Error(err, "Unable to reconcile Gitpod ServiceMonitors")
		return ctrl.Result{}, err
	}

	// Update Status, and if not ready, requeue reconciliation
	if err := r.updateStatus(ctx, &cell); err != nil {
		r.Logger.Error(err, "Unable to update Cell status")
		return ctrl.Result{}, err
	}

	if !r.isCellReady(ctx, cell) {
		r.Logger.Info("Cell is not ready, requeuing")
		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CellReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &appsv1.Deployment{}, POOwnerKey, func(rawObject client.Object) []string {
		deployment := rawObject.(*appsv1.Deployment)
		owner := metav1.GetControllerOf(deployment)
		if owner == nil {
			return nil
		}

		if owner.APIVersion != apiGVStr || owner.Kind != "Cell" {
			return nil
		}

		return []string{owner.Name}
	}); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &corev1.Service{}, POOwnerKey, func(rawObject client.Object) []string {
		service := rawObject.(*corev1.Service)
		owner := metav1.GetControllerOf(service)
		if owner == nil {
			return nil
		}

		if owner.APIVersion != apiGVStr || owner.Kind != "Cell" {
			return nil
		}

		return []string{owner.Name}
	}); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &corev1.ServiceAccount{}, POOwnerKey, func(rawObject client.Object) []string {
		serviceAccount := rawObject.(*corev1.ServiceAccount)
		owner := metav1.GetControllerOf(serviceAccount)
		if owner == nil {
			return nil
		}

		if owner.APIVersion != apiGVStr || owner.Kind != "Cell" {
			return nil
		}

		return []string{owner.Name}
	}); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &rbacv1.ClusterRole{}, POOwnerKey, func(rawObject client.Object) []string {
		clusterRole := rawObject.(*rbacv1.ClusterRole)
		owner := metav1.GetControllerOf(clusterRole)
		if owner == nil {
			return nil
		}

		if owner.APIVersion != apiGVStr || owner.Kind != "Cell" {
			return nil
		}

		return []string{owner.Name}
	}); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &rbacv1.ClusterRoleBinding{}, POOwnerKey, func(rawObject client.Object) []string {
		clusterRoleBinding := rawObject.(*rbacv1.ClusterRoleBinding)
		owner := metav1.GetControllerOf(clusterRoleBinding)
		if owner == nil {
			return nil
		}

		if owner.APIVersion != apiGVStr || owner.Kind != "Cell" {
			return nil
		}

		return []string{owner.Name}
	}); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &pomonitoringv1.ServiceMonitor{}, POOwnerKey, func(rawObject client.Object) []string {
		serviceMonitor := rawObject.(*pomonitoringv1.ServiceMonitor)
		owner := metav1.GetControllerOf(serviceMonitor)
		if owner == nil {
			return nil
		}

		if owner.APIVersion != apiGVStr || owner.Kind != "Cell" {
			return nil
		}

		return []string{owner.Name}
	}); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &pomonitoringv1.Prometheus{}, POOwnerKey, func(rawObject client.Object) []string {
		prometheus := rawObject.(*pomonitoringv1.Prometheus)
		owner := metav1.GetControllerOf(prometheus)
		if owner == nil {
			return nil
		}

		if owner.APIVersion != apiGVStr || owner.Kind != "Cell" {
			return nil
		}

		return []string{owner.Name}
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&monitoringv1alpha1.Cell{}).
		Complete(r)
}

func (r *CellReconciler) updateStatus(ctx context.Context, cell *monitoringv1alpha1.Cell) error {
	poReady, err := r.isPrometheusOperatorReady(ctx, cell)
	if err != nil {
		r.Logger.Error(err, "Failed to get Prometheus-operator Status")
		return err
	}
	cell.Status.PrometheusOperatorReady = &poReady

	prometheusReady, err := r.isPrometheusReady(ctx, cell)
	if err != nil {
		r.Logger.Error(err, "Failed to get Prometheus Status")
		return err
	}
	cell.Status.PrometheusReady = &prometheusReady

	if prometheusReady {
		neReady, err := r.isExporterReady(ctx, cell, `up{job="node-exporter"} == 1`, 1)
		if err != nil {
			r.Logger.Error(err, "Failed to fetch node-exporter's metrics")
			return err
		}
		cell.Status.NodeExporterReady = &neReady

		ksmReady, err := r.isExporterReady(ctx, cell, `up{job="kube-state-metrics"} == 1`, 2)
		if err != nil {
			r.Logger.Error(err, "Failed to fetch kubestate-metrics' metrics")
			return err
		}
		cell.Status.KubeStateMetricsReady = &ksmReady

		kubeletReady, err := r.isExporterReady(ctx, cell, `up{job="kubelet"} == 1`, 3)
		if err != nil {
			r.Logger.Error(err, "Failed to fetch kubelet's' metrics")
			return err
		}
		cell.Status.KubeletReady = &kubeletReady

		apiserverReady, err := r.isExporterReady(ctx, cell, `up{job="apiserver"} == 1`, 1)
		if err != nil {
			r.Logger.Error(err, "Failed to fetch apiserver's' metrics")
			return err
		}
		cell.Status.APIServerReady = &apiserverReady
	}

	return r.Status().Update(ctx, cell)
}

func (r *CellReconciler) reconcilePrometheusOperator(ctx context.Context, cell *monitoringv1alpha1.Cell, req ctrl.Request) error {

	/** Cluster Role **/
	var clusterRole rbacv1.ClusterRole
	err := r.Get(ctx, types.NamespacedName{Name: fmt.Sprintf("%s-%s", prometheusoperator.Name, cell.Name), Namespace: cell.Namespace}, &clusterRole)
	if client.IgnoreNotFound(err) != nil {
		r.Logger.Error(err, "unable to get child clusterrole")
		return err
	}
	desiredClusterRole := prometheusoperator.ClusterRole(cell)
	if err != nil && apierrors.IsNotFound(err) {
		if err := r.Create(ctx, desiredClusterRole); err != nil {
			r.Logger.Error(err, "failed to create prometheus-operator's ClusterRole")
			return err
		}
	} else {
		clusterRole.Labels = desiredClusterRole.Labels
		clusterRole.Name = desiredClusterRole.Name
		clusterRole.Rules = desiredClusterRole.Rules
		if err := r.Update(ctx, &clusterRole); err != nil {
			r.Logger.Error(err, "failed to update prometheus-operator's ClusterRole")
			return err
		}
	}

	/** ClusterRoleBinding **/
	var clusterRoleBinding rbacv1.ClusterRoleBinding
	err = r.Get(ctx, types.NamespacedName{Name: fmt.Sprintf("%s-%s", prometheusoperator.Name, cell.Name), Namespace: cell.Namespace}, &clusterRoleBinding)
	if client.IgnoreNotFound(err) != nil {
		r.Logger.Error(err, "unable to get child clusterrolebinding")
		return err
	}
	desiredClusterRoleBinding := prometheusoperator.ClusterRoleBinding(cell)
	if err != nil && apierrors.IsNotFound(err) {
		if err := r.Create(ctx, desiredClusterRoleBinding); err != nil {
			r.Logger.Error(err, "failed to create prometheus-operator's ClusterRoleBinding")
			return err
		}
	} else {
		clusterRoleBinding.Labels = desiredClusterRoleBinding.Labels
		clusterRoleBinding.Name = desiredClusterRoleBinding.Name
		clusterRoleBinding.Subjects = desiredClusterRoleBinding.Subjects
		clusterRoleBinding.RoleRef = desiredClusterRoleBinding.RoleRef
		if err := r.Update(ctx, &clusterRoleBinding); err != nil {
			r.Logger.Error(err, "failed to update prometheus-operator's ClusterRoleBinding")
			return err
		}
	}

	/** ServiceAccount **/
	var serviceAccount corev1.ServiceAccount
	err = r.Get(ctx, types.NamespacedName{Name: fmt.Sprintf("%s-%s", prometheusoperator.Name, cell.Name), Namespace: cell.Namespace}, &serviceAccount)
	if client.IgnoreNotFound(err) != nil {
		r.Logger.Error(err, "unable to get child serviceaccount")
		return err
	}
	desiredServiceAccount := prometheusoperator.ServiceAccount(cell)
	if err != nil && apierrors.IsNotFound(err) {
		if err := r.Create(ctx, desiredServiceAccount); err != nil {
			r.Logger.Error(err, "failed to create prometheus-operator's ServiceAccount")
			return err
		}
	} else {
		serviceAccount.Labels = desiredServiceAccount.Labels
		serviceAccount.Name = desiredServiceAccount.Name
		serviceAccount.AutomountServiceAccountToken = desiredServiceAccount.AutomountServiceAccountToken
		if err := r.Update(ctx, &serviceAccount); err != nil {
			r.Logger.Error(err, "failed to update prometheus-operator's ServiceAccount")
			return err
		}
	}

	/** Service **/
	var service corev1.Service
	err = r.Get(ctx, types.NamespacedName{Name: fmt.Sprintf("%s-%s", prometheusoperator.Name, cell.Name), Namespace: cell.Namespace}, &service)
	if client.IgnoreNotFound(err) != nil {
		r.Logger.Error(err, "unable to get child service")
		return err
	}
	desiredService := prometheusoperator.Service(cell)
	if err != nil && apierrors.IsNotFound(err) {
		if err := r.Create(ctx, desiredService); err != nil {
			r.Logger.Error(err, "failed to create prometheus-operator's Service")
			return err
		}
	} else {
		service.Labels = desiredService.Labels
		service.Name = desiredService.Name
		service.Spec = desiredService.Spec
		if err := r.Update(ctx, &service); err != nil {
			r.Logger.Error(err, "failed to update prometheus-operator's Service")
			return err
		}
	}

	/** Deployment **/
	var deployment appsv1.Deployment
	err = r.Get(ctx, types.NamespacedName{Name: fmt.Sprintf("%s-%s", prometheusoperator.Name, cell.Name), Namespace: cell.Namespace}, &deployment)
	if client.IgnoreNotFound(err) != nil {
		r.Logger.Error(err, "unable to get child deployment")
		return err
	}

	desiredDeployment := prometheusoperator.Deployment(cell)
	if err != nil && apierrors.IsNotFound(err) {
		if err := r.Create(ctx, desiredDeployment); err != nil {
			r.Logger.Error(err, "failed to create prometheus-operator's Deployment")
			return err
		}
	} else {
		deployment.Labels = desiredDeployment.Labels
		deployment.Name = desiredDeployment.Name
		deployment.Spec = desiredDeployment.Spec
		if err := r.Update(ctx, &deployment); err != nil {
			r.Logger.Error(err, "failed to update prometheus-operator's deployment")
			return err
		}
	}

	/** ServiceMonitor **/
	var serviceMonitor pomonitoringv1.ServiceMonitor
	err = r.Get(ctx, types.NamespacedName{Name: fmt.Sprintf("%s-%s", prometheusoperator.Name, cell.Name), Namespace: cell.Namespace}, &serviceMonitor)
	if client.IgnoreNotFound(err) != nil {
		r.Logger.Error(err, "unable to get child servicemonitor")
		return err
	}

	desiredServiceMonitor := prometheusoperator.ServiceMonitor(cell)
	if err != nil && apierrors.IsNotFound(err) {
		if err := r.Create(ctx, desiredServiceMonitor); err != nil {
			r.Logger.Error(err, "failed to create prometheus-operator's ServiceMonitor")
			return err
		}
	} else {
		serviceMonitor.Labels = desiredServiceMonitor.Labels
		serviceMonitor.Name = desiredServiceMonitor.Name
		serviceMonitor.Spec = desiredServiceMonitor.Spec
		if err := r.Update(ctx, &serviceMonitor); err != nil {
			r.Logger.Error(err, "failed to update prometheus-operator's ServiceMonitor")
			return err
		}
	}

	return nil
}

func (r *CellReconciler) isPrometheusOperatorReady(ctx context.Context, cell *monitoringv1alpha1.Cell) (bool, error) {
	var deployment appsv1.Deployment
	err := r.Get(ctx, types.NamespacedName{Name: fmt.Sprintf("%s-%s", prometheusoperator.Name, cell.Name), Namespace: cell.Namespace}, &deployment)
	if client.IgnoreNotFound(err) != nil {
		r.Logger.Error(err, "unable to get child deployment")
		return false, err
	}

	if apierrors.IsNotFound(err) || deployment.Status.AvailableReplicas < 1 {
		r.Logger.Info("prometheus-operator not ready")
		return false, nil
	}

	return true, nil
}

func (r *CellReconciler) reconcilePrometheus(ctx context.Context, cell *monitoringv1alpha1.Cell, req ctrl.Request) error {
	/** ClusterRole **/
	var clusterRole rbacv1.ClusterRole
	desiredClusterRole := prometheus.ClusterRole(cell)
	err := r.Get(ctx, types.NamespacedName{Name: desiredClusterRole.Name, Namespace: cell.Namespace}, &clusterRole)
	if client.IgnoreNotFound(err) != nil {
		r.Logger.Error(err, "unable to get child clusterrole")
		return err
	}
	if err != nil && apierrors.IsNotFound(err) {
		if err := r.Create(ctx, desiredClusterRole); err != nil {
			r.Logger.Error(err, "failed to create prometheus's ClusterRole")
			return err
		}
	} else {
		clusterRole.Labels = desiredClusterRole.Labels
		clusterRole.Name = desiredClusterRole.Name
		clusterRole.Rules = desiredClusterRole.Rules
		if err := r.Update(ctx, &clusterRole); err != nil {
			r.Logger.Error(err, "failed to update prometheus's ClusterRole")
			return err
		}
	}

	/** ClusterRoleBinding **/
	var clusterRoleBinding rbacv1.ClusterRoleBinding
	desiredClusterRoleBinding := prometheus.ClusterRoleBinding(cell)
	err = r.Get(ctx, types.NamespacedName{Name: desiredClusterRoleBinding.Name, Namespace: cell.Namespace}, &clusterRoleBinding)
	if client.IgnoreNotFound(err) != nil {
		r.Logger.Error(err, "unable to get child clusterRoleBinding")
		return err
	}
	if err != nil && apierrors.IsNotFound(err) {
		if err := r.Create(ctx, desiredClusterRoleBinding); err != nil {
			r.Logger.Error(err, "failed to create prometheus's ClusterRoleBinding")
			return err
		}
	} else {
		clusterRoleBinding.Labels = desiredClusterRoleBinding.Labels
		clusterRoleBinding.Name = desiredClusterRoleBinding.Name
		clusterRoleBinding.Subjects = desiredClusterRoleBinding.Subjects
		clusterRoleBinding.RoleRef = desiredClusterRoleBinding.RoleRef
		if err := r.Update(ctx, &clusterRoleBinding); err != nil {
			r.Logger.Error(err, "failed to update prometheus's ClusterRoleBinding")
			return err
		}
	}

	/** Roles **/
	desiredRoles := prometheus.Roles(cell)
	var currentRole rbacv1.Role
	for _, role := range desiredRoles {
		err = r.Get(ctx, types.NamespacedName{Name: role.Name, Namespace: role.Namespace}, &currentRole)
		if client.IgnoreNotFound(err) != nil {
			r.Logger.Error(err, "unable to get child Roles")
			return err
		}
		if err != nil && apierrors.IsNotFound(err) {
			if err := r.Create(ctx, role); err != nil {
				r.Logger.Error(err, "failed to create prometheus's Role")
				return err
			}
		} else {
			currentRole.Labels = role.Labels
			currentRole.Name = role.Name
			currentRole.Rules = role.Rules
			if err := r.Update(ctx, &currentRole); err != nil {
				r.Logger.Error(err, "failed to update prometheus's Role")
				return err
			}
		}
	}

	/** RoleBindings **/
	desiredRoleBindings := prometheus.RoleBindings(cell)
	var currentRoleBinding rbacv1.RoleBinding
	for _, roleBinding := range desiredRoleBindings {
		err = r.Get(ctx, types.NamespacedName{Name: roleBinding.Name, Namespace: roleBinding.Namespace}, &currentRoleBinding)
		if client.IgnoreNotFound(err) != nil {
			r.Logger.Error(err, "unable to get child RoleBindings")
			return err
		}
		if err != nil && apierrors.IsNotFound(err) {
			if err := r.Create(ctx, roleBinding); err != nil {
				r.Logger.Error(err, "failed to create prometheus's RoleBinding")
				return err
			}
		} else {
			currentRoleBinding.Labels = roleBinding.Labels
			currentRoleBinding.Name = roleBinding.Name
			currentRoleBinding.Subjects = roleBinding.Subjects
			currentRoleBinding.RoleRef = roleBinding.RoleRef
			if err := r.Update(ctx, &currentRole); err != nil {
				r.Logger.Error(err, "failed to update prometheus's RoleBinding")
				return err
			}
		}
	}

	/** ServiceAccount **/
	var serviceAccount corev1.ServiceAccount
	desiredServiceAccount := prometheus.ServiceAccount(cell)
	err = r.Get(ctx, types.NamespacedName{Name: desiredServiceAccount.Name, Namespace: cell.Namespace}, &serviceAccount)
	if client.IgnoreNotFound(err) != nil {
		r.Logger.Error(err, "unable to get child serviceaccount")
		return err
	}
	if err != nil && apierrors.IsNotFound(err) {
		if err := r.Create(ctx, desiredServiceAccount); err != nil {
			r.Logger.Error(err, "failed to create prometheus's ServiceAccount")
			return err
		}
	} else {
		serviceAccount.Labels = desiredServiceAccount.Labels
		serviceAccount.Name = desiredServiceAccount.Name
		serviceAccount.AutomountServiceAccountToken = desiredServiceAccount.AutomountServiceAccountToken
		if err := r.Update(ctx, &serviceAccount); err != nil {
			r.Logger.Error(err, "failed to update prometheus's ServiceAccount")
			return err
		}
	}

	/** Service **/
	var service corev1.Service
	desiredService := prometheus.Service(cell)
	err = r.Get(ctx, types.NamespacedName{Name: desiredService.Name, Namespace: cell.Namespace}, &service)
	if client.IgnoreNotFound(err) != nil {
		r.Logger.Error(err, "unable to get child service")
		return err
	}
	if err != nil && apierrors.IsNotFound(err) {
		if err := r.Create(ctx, desiredService); err != nil {
			r.Logger.Error(err, "failed to create prometheus's Service")
			return err
		}
	} else {
		service.Labels = desiredService.Labels
		service.Name = desiredService.Name
		service.Spec = desiredService.Spec
		if err := r.Update(ctx, &service); err != nil {
			r.Logger.Error(err, "failed to update prometheus's Service")
			return err
		}
	}

	/** ServiceMonitor **/
	var serviceMonitor pomonitoringv1.ServiceMonitor
	desiredServiceMonitor := prometheus.ServiceMonitor(cell)
	err = r.Get(ctx, types.NamespacedName{Name: desiredServiceMonitor.Name, Namespace: cell.Namespace}, &serviceMonitor)
	if client.IgnoreNotFound(err) != nil {
		r.Logger.Error(err, "unable to get child servicemonitor")
		return err
	}

	if err != nil && apierrors.IsNotFound(err) {
		if err := r.Create(ctx, desiredServiceMonitor); err != nil {
			r.Logger.Error(err, "failed to create prometheus's ServiceMonitor")
			return err
		}
	} else {
		serviceMonitor.Labels = desiredServiceMonitor.Labels
		serviceMonitor.Name = desiredServiceMonitor.Name
		serviceMonitor.Spec = desiredServiceMonitor.Spec
		if err := r.Update(ctx, &serviceMonitor); err != nil {
			r.Logger.Error(err, "failed to update prometheus's ServiceMonitor")
			return err
		}
	}

	/** Prometheus **/
	// We can't name the variable 'prometheus' because it conflicts with the package named 'prometheus'
	var p pomonitoringv1.Prometheus
	desiredPrometheus := prometheus.Prometheus(cell)
	err = r.Get(ctx, types.NamespacedName{Name: desiredPrometheus.Name, Namespace: cell.Namespace}, &p)
	if client.IgnoreNotFound(err) != nil {
		r.Logger.Error(err, "unable to get child prometheus")
		return err
	}

	if err != nil && apierrors.IsNotFound(err) {
		if err := r.Create(ctx, desiredPrometheus); err != nil {
			r.Logger.Error(err, "failed to create prometheus instance")
			return err
		}
	} else {
		p.Labels = desiredPrometheus.Labels
		p.Name = desiredPrometheus.Name
		p.Spec = desiredPrometheus.Spec
		if err := r.Update(ctx, &p); err != nil {
			r.Logger.Error(err, "failed to update prometheus instance")
			return err
		}
	}

	return nil
}

func (r *CellReconciler) isPrometheusReady(ctx context.Context, cell *monitoringv1alpha1.Cell) (bool, error) {
	var p pomonitoringv1.Prometheus
	err := r.Get(ctx, types.NamespacedName{Name: fmt.Sprintf("%s-%s", prometheus.Name, cell.Name), Namespace: cell.Namespace}, &p)
	if client.IgnoreNotFound(err) != nil {
		r.Logger.Error(err, "unable to get child prometheus")
		return false, err
	}

	if apierrors.IsNotFound(err) || p.Status.AvailableReplicas < 1 {
		r.Logger.Info("prometheus not ready")
		return false, nil
	}

	return true, nil
}

func (r *CellReconciler) reconcileGitpodServiceMonitors(ctx context.Context, cell *monitoringv1alpha1.Cell, req ctrl.Request) error {
	/** NetworkPolicies **/
	desirednps := gitpod.NetworkPolicies(cell)
	var currentnp networkv1.NetworkPolicy
	for _, np := range desirednps {
		err := r.Get(ctx, types.NamespacedName{Name: np.Name, Namespace: np.Namespace}, &currentnp)
		if client.IgnoreNotFound(err) != nil {
			r.Logger.Error(err, "unable to get child NetworkPolicy")
			return err
		}
		if err != nil && apierrors.IsNotFound(err) {
			if err := r.Create(ctx, np); err != nil {
				r.Logger.Error(err, "failed to create NetworkPolicy")
				return err
			}
		} else {
			currentnp.Labels = np.Labels
			currentnp.Name = np.Name
			currentnp.Spec = np.Spec
			if err := r.Update(ctx, &currentnp); err != nil {
				r.Logger.Error(err, "failed to update NetworkPolicy")
				return err
			}
		}
	}

	/** Services **/
	desiredservices := gitpod.Services(cell)
	var currentService corev1.Service
	for _, svc := range desiredservices {
		err := r.Get(ctx, types.NamespacedName{Name: svc.Name, Namespace: svc.Namespace}, &currentService)
		if client.IgnoreNotFound(err) != nil {
			r.Logger.Error(err, "unable to get child Service")
			return err
		}
		if err != nil && apierrors.IsNotFound(err) {
			if err := r.Create(ctx, svc); err != nil {
				r.Logger.Error(err, "failed to create Service")
				return err
			}
		} else {
			currentService.Labels = svc.Labels
			currentService.Name = svc.Name
			currentService.Spec = svc.Spec
			if err := r.Update(ctx, &currentService); err != nil {
				r.Logger.Error(err, "failed to update Service")
				return err
			}
		}
	}

	/** ServicesMonitors **/
	desiredServiceMonitors := gitpod.ServiceMonitors(cell)
	var currentServiceMonitor pomonitoringv1.ServiceMonitor
	for _, servicemonitor := range desiredServiceMonitors {
		err := r.Get(ctx, types.NamespacedName{Name: servicemonitor.Name, Namespace: servicemonitor.Namespace}, &currentServiceMonitor)
		if client.IgnoreNotFound(err) != nil {
			r.Logger.Error(err, "unable to get child ServiceMonitor")
			return err
		}
		if err != nil && apierrors.IsNotFound(err) {
			if err := r.Create(ctx, servicemonitor); err != nil {
				r.Logger.Error(err, "failed to create ServiceMonitor")
				return err
			}
		} else {
			currentServiceMonitor.Labels = servicemonitor.Labels
			currentServiceMonitor.Name = servicemonitor.Name
			currentServiceMonitor.Spec = servicemonitor.Spec
			if err := r.Update(ctx, &currentServiceMonitor); err != nil {
				r.Logger.Error(err, "failed to update ServiceMonitor")
				return err
			}
		}
	}

	return nil
}

func (r *CellReconciler) reconcileExporters(ctx context.Context, cell *monitoringv1alpha1.Cell, req ctrl.Request) error {
	/** Cluster Roles **/
	var desiredClusterRoles []*rbacv1.ClusterRole
	desiredClusterRoles = append(desiredClusterRoles,
		nodeexporter.ClusterRole(cell),
		kubestatemetrics.ClusterRole(cell),
	)
	var currentClusterRole rbacv1.ClusterRole
	for _, clusterRole := range desiredClusterRoles {
		err := r.Get(ctx, types.NamespacedName{Name: clusterRole.Name, Namespace: clusterRole.Namespace}, &currentClusterRole)
		if client.IgnoreNotFound(err) != nil {
			r.Logger.Error(err, "unable to get child clusterrole")
			return err
		}
		if err != nil && apierrors.IsNotFound(err) {
			if err := r.Create(ctx, clusterRole); err != nil {
				r.Logger.Error(err, "failed to create exporter ClusterRole")
				return err
			}
		} else {
			currentClusterRole.Labels = clusterRole.Labels
			currentClusterRole.Name = clusterRole.Name
			currentClusterRole.Rules = clusterRole.Rules
			if err := r.Update(ctx, &currentClusterRole); err != nil {
				r.Logger.Error(err, "failed to update exporter ClusterRole")
				return err
			}
		}
	}

	/** ClusterRoleBinding **/
	var desiredClusterRoleBindings []*rbacv1.ClusterRoleBinding
	desiredClusterRoleBindings = append(desiredClusterRoleBindings,
		nodeexporter.ClusterRoleBinding(cell),
		kubestatemetrics.ClusterRoleBinding(cell),
	)
	var currentClusterRoleBinding rbacv1.ClusterRoleBinding
	for _, clusterRoleBinding := range desiredClusterRoleBindings {
		err := r.Get(ctx, types.NamespacedName{Name: clusterRoleBinding.Name, Namespace: clusterRoleBinding.Namespace}, &currentClusterRoleBinding)
		if client.IgnoreNotFound(err) != nil {
			r.Logger.Error(err, "unable to get child clusterrolebinding")
			return err
		}
		if err != nil && apierrors.IsNotFound(err) {
			if err := r.Create(ctx, clusterRoleBinding); err != nil {
				r.Logger.Error(err, "failed to create exporter ClusterRoleBinding")
				return err
			}
		} else {
			currentClusterRoleBinding.Labels = clusterRoleBinding.Labels
			currentClusterRoleBinding.Name = clusterRoleBinding.Name
			currentClusterRoleBinding.Subjects = clusterRoleBinding.Subjects
			currentClusterRoleBinding.RoleRef = clusterRoleBinding.RoleRef
			if err := r.Update(ctx, &currentClusterRoleBinding); err != nil {
				r.Logger.Error(err, "failed to update exporter ClusterRoleBinding")
				return err
			}
		}
	}

	/** ServiceAccount **/
	var desiredServiceAccounts []*corev1.ServiceAccount
	desiredServiceAccounts = append(desiredServiceAccounts,
		nodeexporter.ServiceAccount(cell),
		kubestatemetrics.ServiceAccount(cell),
	)
	var currentServiceAccount corev1.ServiceAccount
	for _, serviceAccount := range desiredServiceAccounts {
		err := r.Get(ctx, types.NamespacedName{Name: serviceAccount.Name, Namespace: serviceAccount.Namespace}, &currentServiceAccount)
		if client.IgnoreNotFound(err) != nil {
			r.Logger.Error(err, "unable to get child serviceaccount")
			return err
		}
		if err != nil && apierrors.IsNotFound(err) {
			if err := r.Create(ctx, serviceAccount); err != nil {
				r.Logger.Error(err, "failed to create exporter ServiceAccount")
				return err
			}
		} else {
			currentServiceAccount.Labels = serviceAccount.Labels
			currentServiceAccount.Name = serviceAccount.Name
			currentServiceAccount.AutomountServiceAccountToken = serviceAccount.AutomountServiceAccountToken
			if err := r.Update(ctx, &currentServiceAccount); err != nil {
				r.Logger.Error(err, "failed to update exporter ServiceAccount")
				return err
			}
		}
	}

	/** Service **/
	var desiredServices []*corev1.Service
	desiredServices = append(desiredServices,
		nodeexporter.Service(cell),
		kubestatemetrics.Service(cell),
	)
	var currentService corev1.Service
	for _, service := range desiredServices {
		err := r.Get(ctx, types.NamespacedName{Name: service.Name, Namespace: service.Namespace}, &currentService)
		if client.IgnoreNotFound(err) != nil {
			r.Logger.Error(err, "unable to get child service")
			return err
		}
		if err != nil && apierrors.IsNotFound(err) {
			if err := r.Create(ctx, service); err != nil {
				r.Logger.Error(err, "failed to create exporter Service")
				return err
			}
		} else {
			currentService.Labels = service.Labels
			currentService.Name = service.Name
			currentService.Spec = service.Spec
			if err := r.Update(ctx, &currentService); err != nil {
				r.Logger.Error(err, "failed to update exporter Service")
				return err
			}
		}
	}

	/** Daemonset **/
	var daemonset appsv1.DaemonSet
	desiredDaemonset := nodeexporter.Daemonset(cell)
	err := r.Get(ctx, types.NamespacedName{Name: desiredDaemonset.Name, Namespace: desiredDaemonset.Namespace}, &daemonset)
	if client.IgnoreNotFound(err) != nil {
		r.Logger.Error(err, "unable to get child deployment")
		return err
	}
	if err != nil && apierrors.IsNotFound(err) {
		if err := r.Create(ctx, desiredDaemonset); err != nil {
			r.Logger.Error(err, "failed to create node-exporter's Daemonset")
			return err
		}
	} else {
		daemonset.Labels = desiredDaemonset.Labels
		daemonset.Name = desiredDaemonset.Name
		daemonset.Spec = desiredDaemonset.Spec
		if err := r.Update(ctx, &daemonset); err != nil {
			r.Logger.Error(err, "failed to update node-exporter's Daemonset")
			return err
		}
	}

	/** Deployment **/
	var deployment appsv1.Deployment
	desiredDeployment := kubestatemetrics.Deployment(cell)
	err = r.Get(ctx, types.NamespacedName{Name: desiredDeployment.Name, Namespace: desiredDeployment.Namespace}, &deployment)
	if client.IgnoreNotFound(err) != nil {
		r.Logger.Error(err, "unable to get child deployment")
		return err
	}
	if err != nil && apierrors.IsNotFound(err) {
		if err := r.Create(ctx, desiredDeployment); err != nil {
			r.Logger.Error(err, "failed to create kubestate-metrics's Deployment")
			return err
		}
	} else {
		deployment.Labels = desiredDeployment.Labels
		deployment.Name = desiredDeployment.Name
		deployment.Spec = desiredDeployment.Spec
		if err := r.Update(ctx, &daemonset); err != nil {
			r.Logger.Error(err, "failed to update kubestate-metrics's Daemonset")
			return err
		}
	}

	/** ServiceMonitor **/
	var desiredServiceMonitors []*pomonitoringv1.ServiceMonitor
	desiredServiceMonitors = append(desiredServiceMonitors, kubernetes.ServiceMonitors(cell)...)
	desiredServiceMonitors = append(desiredServiceMonitors,
		nodeexporter.ServiceMonitor(cell),
		kubestatemetrics.ServiceMonitor(cell),
	)

	var currentServiceMonitor pomonitoringv1.ServiceMonitor
	for _, servicemonitor := range desiredServiceMonitors {
		err = r.Get(ctx, types.NamespacedName{Name: servicemonitor.Name, Namespace: servicemonitor.Namespace}, &currentServiceMonitor)
		if client.IgnoreNotFound(err) != nil {
			r.Logger.Error(err, "unable to get child servicemonitor")
			return err
		}

		if err != nil && apierrors.IsNotFound(err) {
			if err := r.Create(ctx, servicemonitor); err != nil {
				r.Logger.Error(err, "failed to create exporter ServiceMonitor")
				return err
			}
		} else {
			currentServiceMonitor.Labels = servicemonitor.Labels
			currentServiceMonitor.Name = servicemonitor.Name
			currentServiceMonitor.Spec = servicemonitor.Spec
			if err := r.Update(ctx, &currentServiceMonitor); err != nil {
				r.Logger.Error(err, "failed to update exporter ServiceMonitor")
				return err
			}
		}
	}

	return nil
}

func (r *CellReconciler) isExporterReady(ctx context.Context, cell *monitoringv1alpha1.Cell, query string, expectedResult int) (bool, error) {

	rsp, err := prometheus.Query(query, cell, r.PodRESTClient)
	if err != nil {
		return false, err
	}

	if rsp != expectedResult {
		r.Logger.Error(err, "querying for exporter metrics returned unexpected result. Check cell statu or Prometheus targets page for further information")
		return false, nil
	}

	return true, nil
}

func (r *CellReconciler) isCellReady(ctx context.Context, cell monitoringv1alpha1.Cell) bool {
	return *cell.Status.PrometheusOperatorReady &&
		*cell.Status.PrometheusReady &&
		*cell.Status.APIServerReady &&
		*cell.Status.KubeletReady &&
		*cell.Status.NodeExporterReady &&
		*cell.Status.KubeStateMetricsReady
}
