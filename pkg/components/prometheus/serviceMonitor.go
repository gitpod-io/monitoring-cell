package prometheus

import (
	"fmt"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	monitoringv1alpha1 "github.com/gitpod-io/monitoring-cell/api/v1alpha1"
)

func ServiceMonitor(cell *monitoringv1alpha1.Cell) *monitoringv1.ServiceMonitor {
	return &monitoringv1.ServiceMonitor{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "monitoring.coreos.com/v1",
			Kind:       "ServiceMonitor",
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
		Spec: monitoringv1.ServiceMonitorSpec{
			Endpoints: []monitoringv1.Endpoint{
				{
					Port:     "web",
					Interval: "60s",
					// MetricRelabelConfigs: drop from spec.Droplist
				},
				{
					Port:     "reloader-web",
					Interval: "60s",
					// MetricRelabelConfigs: drop from spec.Droplist
				},
			},
			Selector: metav1.LabelSelector{
				MatchLabels: cell.Labels,
			},
		},
	}
}
