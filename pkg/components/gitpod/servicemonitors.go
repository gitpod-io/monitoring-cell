package gitpod

import (
	"fmt"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	monitoringv1alpha1 "github.com/gitpod-io/monitoring-cell/api/v1alpha1"
)

func ServiceMonitors(targets []string, cell *monitoringv1alpha1.Cell) []*monitoringv1.ServiceMonitor {
	var serviceMonitors []*monitoringv1.ServiceMonitor
	for _, target := range targets {
		serviceMonitors = append(serviceMonitors, &monitoringv1.ServiceMonitor{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "monitoring.coreos.com/v1",
				Kind:       "ServiceMonitor",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-%s", App, target),
				Namespace: cell.Namespace,
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
			Spec: monitoringv1.ServiceMonitorSpec{
				Endpoints: []monitoringv1.Endpoint{
					{
						BearerTokenFile: "/var/run/secrets/kubernetes.io/serviceaccount/token",
						Interval:        "60s",
						Port:            "metrics",
						// MetricRelabelConfigs: should be build from spec.Droplist
					},
				},
				JobLabel: "app.kubernetes.io/component",
				NamespaceSelector: monitoringv1.NamespaceSelector{
					MatchNames: []string{cell.Spec.GitpodNamespace},
				},
				Selector: metav1.LabelSelector{
					MatchLabels: labels(target),
				},
			},
		})
	}

	serviceMonitors = append(serviceMonitors, messagebusServiceMonitor(cell))
	serviceMonitors = append(serviceMonitors, proxyCaddyServiceMonitor(cell))

	return serviceMonitors
}

func messagebusServiceMonitor(cell *monitoringv1alpha1.Cell) *monitoringv1.ServiceMonitor {
	return &monitoringv1.ServiceMonitor{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "monitoring.coreos.com/v1",
			Kind:       "ServiceMonitor",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", App, "messagebus"),
			Namespace: cell.Namespace,
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
		Spec: monitoringv1.ServiceMonitorSpec{
			Endpoints: []monitoringv1.Endpoint{
				{
					BearerTokenFile: "/var/run/secrets/kubernetes.io/serviceaccount/token",
					Interval:        "60s",
					Port:            "metrics",
					// MetricRelabelConfigs: should be build from spec.Droplist
				},
			},
			JobLabel: "app.kubernetes.io/component",
			NamespaceSelector: monitoringv1.NamespaceSelector{
				MatchNames: []string{cell.Spec.GitpodNamespace},
			},
			Selector: metav1.LabelSelector{
				MatchLabels: labels("messagebus"),
			},
		},
	}
}

func proxyCaddyServiceMonitor(cell *monitoringv1alpha1.Cell) *monitoringv1.ServiceMonitor {
	return &monitoringv1.ServiceMonitor{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "monitoring.coreos.com/v1",
			Kind:       "ServiceMonitor",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "gitpod-proxy-caddy",
			Namespace: cell.Namespace,
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
		Spec: monitoringv1.ServiceMonitorSpec{
			Endpoints: []monitoringv1.Endpoint{
				{
					BearerTokenFile: "/var/run/secrets/kubernetes.io/serviceaccount/token",
					Interval:        "60s",
					Port:            "metrics",
					// MetricRelabelConfigs: should be build from spec.Droplist
				},
			},
			JobLabel: "app.kubernetes.io/component",
			NamespaceSelector: monitoringv1.NamespaceSelector{
				MatchNames: []string{cell.Spec.GitpodNamespace},
			},
			Selector: metav1.LabelSelector{
				MatchLabels: labels("proxy-caddy"),
			},
		},
	}
}
