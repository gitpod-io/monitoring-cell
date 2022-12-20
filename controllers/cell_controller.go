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

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	monitoringv1alpha1 "github.com/gitpod-io/monitoring-cell/api/v1alpha1"
	prometheusoperator "github.com/gitpod-io/monitoring-cell/pkg/components/prometheus-operator"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	POOwnerKey = ".metadata.controller"
	apiGVStr   = monitoringv1alpha1.GroupVersion.String()
)

// CellReconciler reconciles a Cell object
type CellReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Logger logr.Logger
}

//+kubebuilder:rbac:groups=monitoring.gitpod.io,resources=cells,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=monitoring.gitpod.io,resources=cells/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=monitoring.gitpod.io,resources=cells/finalizers,verbs=update
//+kubebuilder:rbac:groups=,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=,resources=serviceaccounts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterroles,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterrolebindings,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.1/pkg/reconcile
func (r *CellReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.Logger = log.FromContext(ctx)
	var cell monitoringv1alpha1.Cell
	isPrometheusReady := pointer.Bool(false)
	if err := r.Get(ctx, req.NamespacedName, &cell); err != nil {
		r.Logger.Error(err, "Unable to fetch Cell")

		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	err := r.reconcilePrometheusOperator(ctx, &cell, req)
	if err != nil {
		r.Logger.Error(err, "Failed to reconcile Prometheus-Operator")
		return ctrl.Result{}, err
	}

	cell.Status.PrometheusReady = isPrometheusReady
	if err := r.Status().Update(ctx, &cell); err != nil {
		r.Logger.Error(err, "Unable to update Cell status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CellReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &appsv1.Deployment{}, POOwnerKey, func(rawObject client.Object) []string {
		// grab de deployment object, extract the owner
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
		// grab de deployment object, extract the owner
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
		// grab de deployment object, extract the owner
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
		// grab de deployment object, extract the owner
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
		// grab de deployment object, extract the owner
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

	return ctrl.NewControllerManagedBy(mgr).
		For(&monitoringv1alpha1.Cell{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&rbacv1.ClusterRole{}).
		Owns(&rbacv1.ClusterRoleBinding{}).
		Complete(r)
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
	if apierrors.IsNotFound(err) {
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
	if apierrors.IsNotFound(err) {
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
	if apierrors.IsNotFound(err) {
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
	if apierrors.IsNotFound(err) {
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
	if apierrors.IsNotFound(err) {
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

	return nil
}
