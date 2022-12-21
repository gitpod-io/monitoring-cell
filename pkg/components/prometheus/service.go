package prometheus

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	monitoringv1alpha1 "github.com/gitpod-io/monitoring-cell/api/v1alpha1"
)

func Service(cell *monitoringv1alpha1.Cell) *corev1.Service {
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", Name, cell.Name),
			Namespace: cell.Namespace,
			Labels:    cell.Labels,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: cell.APIVersion,
					Kind:       cell.Kind,
					Name:       cell.Name,
					UID:        cell.UID,
				},
			},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       "web",
					Port:       9090,
					TargetPort: intstr.FromString("web"),
				},
				{
					Name:       "reloader-web",
					Port:       8080,
					TargetPort: intstr.FromString("reloader-web"),
				},
			},
			Selector: map[string]string{
				"app.kubernetes.io/name": "prometheus",
			},
		},
	}
}
