package gitpod

import (
	"fmt"

	monitoringv1alpha1 "github.com/gitpod-io/monitoring-cell/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Services(cell *monitoringv1alpha1.Cell) []*corev1.Service {
	var services []*corev1.Service
	for _, target := range targets {
		services = append(services, &corev1.Service{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Service",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-%s", App, target),
				Namespace: cell.Spec.GitpodNamespace,
				Labels:    labels(target),
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
						Name: "metrics",
						Port: 9500,
					},
				},
				Selector: map[string]string{
					"component": target,
				},
			},
		})
	}

	services = append(services, messagebusService(cell))
	services = append(services, proxyCaddyService(cell))

	return services
}

func messagebusService(cell *monitoringv1alpha1.Cell) *corev1.Service {
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", App, "messagebus"),
			Namespace: cell.Spec.GitpodNamespace,
			Labels:    labels("messagebus"),
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
					Name: "metrics",
					Port: 9419,
				},
			},
			Selector: map[string]string{
				"app.kubernetes.io/name": "rabbitmq",
			},
		},
	}
}

func proxyCaddyService(cell *monitoringv1alpha1.Cell) *corev1.Service {
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "gitpod-proxy-caddy",
			Namespace: cell.Spec.GitpodNamespace,
			Labels:    labels("proxy-caddy"),
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
					Name: "caddy-metrics",
					Port: 8003,
				},
			},
			Selector: map[string]string{
				"component": "proxy",
			},
		},
	}
}
