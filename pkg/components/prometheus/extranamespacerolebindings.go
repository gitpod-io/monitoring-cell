package prometheus

import (
	"fmt"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	monitoringv1alpha1 "github.com/gitpod-io/monitoring-cell/api/v1alpha1"
)

// extraNamespaceRoleBindings and extraNamespaceRoles are used to give permission to prometheus to scrape metrics
// from endpoints in other namespaces.
func ExtraNamespaceRoleBindings(cell *monitoringv1alpha1.Cell) []*rbacv1.RoleBinding {
	var extraRoleBindings []*rbacv1.RoleBinding

	extraRoleBindings = append(extraRoleBindings,
		rolebindingFactory(cell.Namespace, cell),
		rolebindingFactory("default", cell),
		rolebindingFactory("kube-system", cell),
	)

	return extraRoleBindings
}

func rolebindingFactory(ns string, cell *monitoringv1alpha1.Cell) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "RoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", Name, cell.Name),
			Namespace: ns,
			Labels:    cell.Labels,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     fmt.Sprintf("%s-%s", Name, cell.Name),
		},
		Subjects: []rbacv1.Subject{
			{
				Kind: "ServiceAccount",
				Name: fmt.Sprintf("%s-%s", Name, cell.Name),
				// Here we associate the service account used by prometheus
				// which lives in the same namespace as prometheus, and not the role.
				Namespace: cell.Namespace,
			},
		},
	}
}
